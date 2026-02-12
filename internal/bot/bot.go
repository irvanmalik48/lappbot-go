package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lappbot/internal/config"
	"lappbot/internal/store"
)

type HandlerFunc func(*Context) error

type Bot struct {
	Token       string
	APIURL      string
	Client      *http.Client
	Store       *store.Store
	Cfg         *config.Config
	StartTime   time.Time
	Handlers    map[string]HandlerFunc
	Middleware  []func(HandlerFunc) HandlerFunc
	bufferPool  sync.Pool
	contextPool sync.Pool
}

func New(cfg *config.Config, store *store.Store) (*Bot, error) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
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
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 512))
			},
		},
		contextPool: sync.Pool{
			New: func() interface{} {
				return &Context{}
			},
		},
	}, nil
}

func (b *Bot) Handle(endpoint string, h HandlerFunc) {
	b.Handlers[endpoint] = h
}

func (b *Bot) Use(m func(HandlerFunc) HandlerFunc) {
	b.Middleware = append(b.Middleware, m)
}

func (b *Bot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := b.contextPool.Get().(*Context)
	ctx.Reset(b, &update)
	defer b.contextPool.Put(ctx)

	if update.Message != nil {
		ctx.Message = update.Message

		if len(update.Message.NewChatMembers) > 0 {
			if h, ok := b.Handlers["new_chat_members"]; ok {
				go b.process(h, ctx)
			}
		}
		if update.Message.LeftChatMember != nil {
			if h, ok := b.Handlers["left_chat_member"]; ok {
				go b.process(h, ctx)
			}
		}

		parts := strings.Fields(ctx.Message.Text)
		if len(parts) > 0 {
			ctx.Args = parts[1:]
			cmd := parts[0]
			if idx := strings.Index(cmd, "@"); idx != -1 {
				cmd = cmd[:idx]
			}
			if h, ok := b.Handlers[cmd]; ok {
				go b.process(h, ctx)
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		if h, ok := b.Handlers["on_text"]; ok {
			go b.process(h, ctx)
			w.WriteHeader(http.StatusOK)
			return
		}

	} else if update.CallbackQuery != nil {
		ctx.Callback = update.CallbackQuery
		data := ctx.Callback.Data

		if h, ok := b.Handlers[data]; ok {
			go b.process(h, ctx)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (b *Bot) process(h HandlerFunc, ctx *Context) {
	final := h
	for i := len(b.Middleware) - 1; i >= 0; i-- {
		final = b.Middleware[i](final)
	}
	if err := final(ctx); err != nil {
		log.Printf("Handler error: %v", err)
	}
}

func (b *Bot) Raw(method string, payload interface{}) error {
	url := b.APIURL + "/bot" + b.Token + "/" + method

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return err
	}

	resp, err := b.Client.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
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

	req := map[string]interface{}{
		"chat_id": chat.ID,
		"user_id": user.ID,
	}
	url := b.APIURL + "/bot" + b.Token + "/getChatMember"

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	json.NewEncoder(buf).Encode(req)

	resp, err := b.Client.Post(url, "application/json", buf)
	b.bufferPool.Put(buf)

	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var res struct {
		Ok     bool       `json:"ok"`
		Result ChatMember `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
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
	var body []byte
	if id, err := strconv.ParseInt(identity, 10, 64); err == nil {
		body = fmt.Appendf(nil, `{"chat_id": %d}`, id)
	} else {
		jsonIdentity, _ := json.Marshal(identity)
		body = fmt.Appendf(nil, `{"chat_id": %s}`, jsonIdentity)
	}

	url := b.APIURL + "/bot" + b.Token + "/getChat"
	resp, err := b.Client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Ok     bool `json:"ok"`
		Result Chat `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
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

func (b *Bot) Start() {
	log.Println("Bot started in Long Polling mode")
	var offset int64 = 0
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			log.Printf("Error getting updates: %v", err)
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

func (b *Bot) getUpdates(offset int64) ([]Update, error) {
	req := map[string]interface{}{
		"offset":  offset,
		"timeout": 30,
	}
	url := b.APIURL + "/bot" + b.Token + "/getUpdates"

	buf := b.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(req); err != nil {
		return nil, err
	}

	resp, err := b.Client.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Ok     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
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
	defer b.contextPool.Put(ctx)

	if update.Message != nil {
		ctx.Message = update.Message

		if len(update.Message.NewChatMembers) > 0 {
			if h, ok := b.Handlers["new_chat_members"]; ok {
				go b.process(h, ctx)
			}
		}
		if update.Message.LeftChatMember != nil {
			if h, ok := b.Handlers["left_chat_member"]; ok {
				go b.process(h, ctx)
			}
		}

		parts := strings.Fields(ctx.Message.Text)
		if len(parts) > 0 {
			ctx.Args = parts[1:]
			cmd := parts[0]
			if idx := strings.Index(cmd, "@"); idx != -1 {
				cmd = cmd[:idx]
			}
			if h, ok := b.Handlers[cmd]; ok {
				go b.process(h, ctx)
				return
			}
		}

		if h, ok := b.Handlers["on_text"]; ok {
			go b.process(h, ctx)
			return
		}

	} else if update.CallbackQuery != nil {
		ctx.Callback = update.CallbackQuery
		data := ctx.Callback.Data

		if h, ok := b.Handlers[data]; ok {
			go b.process(h, ctx)
			return
		}
	}
}
