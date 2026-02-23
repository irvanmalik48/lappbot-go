package bot

import (
	"bytes"
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"

	"lappbot/internal/config"
	"lappbot/internal/store"

	"golang.org/x/time/rate"
)

type HandlerFunc func(*Context) error

type Bot struct {
	Token       string
	APIURL      string
	Client      *fasthttp.Client
	Store       *store.Store
	Cfg         *config.Config
	StartTime   time.Time
	Handlers    map[string]HandlerFunc
	Middleware  []func(HandlerFunc) HandlerFunc
	bufferPool  sync.Pool
	contextPool sync.Pool
	limiter     *rate.Limiter
	Me          *User
}

func New(cfg *config.Config, store *store.Store) (*Bot, error) {
	client := &fasthttp.Client{
		Name:                "Lappland",
		MaxConnsPerHost:     100,
		ReadTimeout:         90 * time.Second,
		WriteTimeout:        10 * time.Second,
		MaxIdleConnDuration: 90 * time.Second,
	}

	return &Bot{
		Token:     cfg.BotToken,
		APIURL:    cfg.BotAPIURL,
		Client:    client,
		Store:     store,
		Cfg:       cfg,
		StartTime: time.Now(),
		Handlers:  make(map[string]HandlerFunc),
		bufferPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 512))
			},
		},
		contextPool: sync.Pool{
			New: func() any {
				return &Context{}
			},
		},
		limiter: rate.NewLimiter(rate.Limit(100), 200),
	}, nil
}

func (b *Bot) GetMe() (*User, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/getMe")

	if err := b.Client.Do(req, resp); err != nil {
		return nil, err
	}

	var res struct {
		Ok     bool `json:"ok"`
		Result User `json:"result"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("ok=false")
	}
	return &res.Result, nil
}

func (b *Bot) Handle(endpoint string, h HandlerFunc) {
	b.Handlers[endpoint] = h
}

func (b *Bot) Use(m func(HandlerFunc) HandlerFunc) {
	b.Middleware = append(b.Middleware, m)
}

func (b *Bot) RequestHandler(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	if b.Cfg.WebhookSecret != "" && string(ctx.Request.Header.Peek("X-Telegram-Bot-Api-Secret-Token")) != b.Cfg.WebhookSecret {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		return
	}

	if !b.limiter.Allow() {
		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
		return
	}

	var update Update
	if err := json.Unmarshal(ctx.PostBody(), &update); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString("dedup:")
	buf.WriteString(strconv.FormatInt(update.UpdateID, 10))
	dedupKey := buf.String()
	b.bufferPool.Put(buf)

	isMember, err := b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Get().Key(dedupKey).Build()).AsInt64()
	if err == nil && isMember == 1 {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}
	b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Set().Key(dedupKey).Value("1").Ex(time.Hour).Build())

	b.processUpdate(&update)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (b *Bot) process(h HandlerFunc, ctx *Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Bytes("stack", debug.Stack()).Msg("Panic in handler")
		}
		b.contextPool.Put(ctx)
	}()

	final := h
	for i := len(b.Middleware) - 1; i >= 0; i-- {
		final = b.Middleware[i](final)
	}
	if err := final(ctx); err != nil {
		log.Error().Err(err).Msg("Handler error")
	}
}

func (b *Bot) Raw(method string, payload any) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/" + method)

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return err
	}
	req.SetBody(buf.Bytes())

	return b.Client.Do(req, resp)
}

func (b *Bot) CheckAdmin(c *Context, chat *Chat, user *User) bool {
	if b.IsAdmin(chat, user) {
		return true
	}
	c.Send("You must be an admin to use this command.")
	return false
}

func (b *Bot) IsAdmin(chat *Chat, user *User) bool {
	keyBuf := make([]byte, 0, 64)
	keyBuf = append(keyBuf, "admin:"...)
	keyBuf = strconv.AppendInt(keyBuf, chat.ID, 10)
	keyBuf = append(keyBuf, ':')
	keyBuf = strconv.AppendInt(keyBuf, user.ID, 10)
	key := string(keyBuf)

	val, err := b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Get().Key(key).Build()).ToString()
	if err == nil {
		return val == "1"
	}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/getChatMember")

	reqData := map[string]any{
		"chat_id": chat.ID,
		"user_id": user.ID,
	}

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(reqData); err != nil {
		return false
	}
	req.SetBody(buf.Bytes())

	if b.Client == nil {
		return false
	}
	if err := b.Client.Do(req, resp); err != nil {
		return false
	}

	var res struct {
		Ok     bool       `json:"ok"`
		Result ChatMember `json:"result"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return false
	}

	isAdmin := res.Result.Status == "administrator" || res.Result.Status == "creator"
	v := "0"
	if isAdmin {
		v = "1"
	}
	b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Set().Key(key).Value(v).Ex(2*time.Minute).Build())

	return isAdmin
}

func (b *Bot) InvalidateAdminCache(chatID, userID int64) {
	key := "admin:" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
	b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Del().Key(key).Build())
}

func (b *Bot) ResolveChat(identity string) (*Chat, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/getChat")

	var body []byte
	if id, err := strconv.ParseInt(identity, 10, 64); err == nil {
		body = fmt.Appendf(nil, `{"chat_id": %d}`, id)
	} else {
		jsonIdentity, _ := json.Marshal(identity)
		body = fmt.Appendf(nil, `{"chat_id": %s}`, jsonIdentity)
	}
	req.SetBody(body)

	if err := b.Client.Do(req, resp); err != nil {
		return nil, err
	}

	var res struct {
		Ok     bool `json:"ok"`
		Result Chat `json:"result"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("chat not found")
	}
	return &res.Result, nil
}

func (b *Bot) GetTargetChat(c *Context) (*Chat, error) {
	if c.Chat().Type == "private" {
		connectedChatID, err := b.Store.GetConnection(c.Sender().ID)
		if err == nil && connectedChatID != 0 {
			return &Chat{ID: connectedChatID, Type: "group", Title: "Connected Chat"}, nil
		}
	}

	return c.Chat(), nil
}

func (b *Bot) StartLongPolling() {
	if err := b.DeleteWebhook(); err != nil {
		log.Error().Err(err).Msg("Failed to delete webhook before long polling")
	}
	log.Info().Msg("Bot started in Long Polling mode")

	me, err := b.GetMe()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get bot info")
	}
	b.Me = me
	log.Info().Msgf("Bot started as %s (@%s)", b.Me.FirstName, b.Me.Username)

	var offset int64 = 0
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			log.Error().Err(err).Msg("Error getting updates")
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			b.processUpdate(&update)
		}
	}
}

func (b *Bot) StartWebhook() {
	if err := b.SetWebhook(b.Cfg.WebhookURL + b.Cfg.WebhookPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to set webhook")
	}

	me, err := b.GetMe()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get bot info")
	}
	b.Me = me
	log.Info().Msgf("Bot started as %s (@%s)", b.Me.FirstName, b.Me.Username)

	log.Info().Msgf("Bot started in Webhook mode on port %d", b.Cfg.WebhookPort)

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case b.Cfg.WebhookPath:
			b.RequestHandler(ctx)
		default:
			ctx.Error("not found", fasthttp.StatusNotFound)
		}
	}

	server := &fasthttp.Server{
		Handler: requestHandler,
		Logger:  &webhookLogger{},
	}

	if err := server.ListenAndServe(fmt.Sprintf(":%d", b.Cfg.WebhookPort)); err != nil {
		log.Fatal().Err(err).Msg("Error in Serve")
	}
}

type webhookLogger struct{}

func (wl *webhookLogger) Printf(format string, args ...any) {
	if strings.Contains(format, "error when serving connection") {
		return
	}
	log.Warn().Msgf("[fasthttp] "+format, args...)
}

func (b *Bot) SetWebhook(url string) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/setWebhook")

	reqData := map[string]any{
		"url":                  url,
		"drop_pending_updates": true,
	}

	if b.Cfg.WebhookSecret != "" {
		reqData["secret_token"] = b.Cfg.WebhookSecret
	}

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(reqData); err != nil {
		return err
	}
	req.SetBody(buf.Bytes())

	if err := b.Client.Do(req, resp); err != nil {
		return err
	}

	return nil
}

func (b *Bot) DeleteWebhook() error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/deleteWebhook")

	reqData := map[string]bool{
		"drop_pending_updates": true,
	}

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(reqData); err != nil {
		return err
	}
	req.SetBody(buf.Bytes())

	if err := b.Client.Do(req, resp); err != nil {
		return err
	}

	return nil
}

func (b *Bot) getUpdates(offset int64) ([]Update, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetRequestURI(b.APIURL + "/bot" + b.Token + "/getUpdates")

	reqData := map[string]any{
		"offset":  offset,
		"timeout": 30,
	}

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(reqData); err != nil {
		return nil, err
	}
	req.SetBody(buf.Bytes())

	if err := b.Client.Do(req, resp); err != nil {
		return nil, err
	}

	var res struct {
		Ok     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("ok=false")
	}
	return res.Result, nil
}

func (b *Bot) processUpdate(update *Update) {
	ctx := b.contextPool.Get().(*Context)
	ctx.Reset(b, update)

	if update.Message != nil || update.ChannelPost != nil {
		if update.Message != nil {
			ctx.Message = update.Message
		} else {
			ctx.Message = update.ChannelPost
		}

		if len(ctx.Message.NewChatMembers) > 0 {
			if h, ok := b.Handlers["new_chat_members"]; ok {
				go b.process(h, ctx)
			} else {
				b.contextPool.Put(ctx)
			}
		} else if update.Message.LeftChatMember != nil {
			if h, ok := b.Handlers["left_chat_member"]; ok {
				go b.process(h, ctx)
			} else {
				b.contextPool.Put(ctx)
			}
		} else {
			parts := strings.Fields(ctx.Message.Text)
			if len(parts) > 0 {
				ctx.Args = parts[1:]
				cmd := parts[0]
				if idx := strings.Index(cmd, "@"); idx != -1 {
					targetBot := cmd[idx+1:]
					if b.Me != nil && !strings.EqualFold(targetBot, b.Me.Username) {
						b.contextPool.Put(ctx)
						return
					}
					cmd = cmd[:idx]
				}
				if h, ok := b.Handlers[cmd]; ok {
					go b.process(h, ctx)
					return
				}
			}

			if h, ok := b.Handlers["unknown_command"]; ok {
				go b.process(h, ctx)
				return
			}
			if h, ok := b.Handlers["on_text"]; ok {
				go b.process(h, ctx)
				return
			}
			b.contextPool.Put(ctx)
		}

	} else if update.CallbackQuery != nil {
		ctx.Callback = update.CallbackQuery
		data := ctx.Callback.Data

		if h, ok := b.Handlers[data]; ok {
			go b.process(h, ctx)
			return
		}

		if idx := strings.Index(data, "|"); idx != -1 {
			if h, ok := b.Handlers[data[:idx]]; ok {
				go b.process(h, ctx)
				return
			}
		}

		b.contextPool.Put(ctx)
	} else {
		b.contextPool.Put(ctx)
	}
}
