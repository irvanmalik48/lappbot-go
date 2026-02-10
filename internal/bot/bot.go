package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"lappbot/internal/config"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v4"
)

type Bot struct {
	Bot       *tele.Bot
	Store     *store.Store
	Cfg       *config.Config
	StartTime time.Time
}

func New(cfg *config.Config, store *store.Store) (*Bot, error) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 0,
	}

	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 60 * time.Second},
		Client: client,
		URL:    cfg.BotAPIURL,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Bot:       b,
		Store:     store,
		Cfg:       cfg,
		StartTime: time.Now(),
	}, nil
}

func (b *Bot) Start() {
	log.Printf("Bot %s started", b.Bot.Me.Username)
	b.Bot.Start()
}

func (b *Bot) IsAdmin(chat *tele.Chat, user *tele.User) bool {
	key := fmt.Sprintf("admin:%d:%d", chat.ID, user.ID)
	val, err := b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Get().Key(key).Build()).ToString()
	if err == nil {
		return val == "1"
	}

	member, err := b.Bot.ChatMemberOf(chat, user)
	if err != nil {
		return false
	}

	isAdmin := member.Role == tele.Administrator || member.Role == tele.Creator

	v := "0"
	if isAdmin {
		v = "1"
	}

	b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Set().Key(key).Value(v).Ex(2*time.Minute).Build())

	return isAdmin
}

func (b *Bot) InvalidateAdminCache(chatID, userID int64) {
	key := fmt.Sprintf("admin:%d:%d", chatID, userID)
	b.Store.Valkey.Do(context.Background(), b.Store.Valkey.B().Del().Key(key).Build())
}

func (b *Bot) GetTargetChat(c tele.Context) (*tele.Chat, error) {
	if c.Chat().Type != tele.ChatPrivate {
		return c.Chat(), nil
	}

	targetID, err := b.Store.GetConnection(c.Sender().ID)
	if err != nil || targetID == 0 {
		return c.Chat(), nil
	}

	targetChat, err := b.Bot.ChatByID(targetID)
	if err != nil {
		return c.Chat(), nil
	}
	return targetChat, nil
}

func (b *Bot) ResolveChat(identity string) (*tele.Chat, error) {
	chat, err := b.Bot.ChatByUsername(identity)
	if err == nil {
		return chat, nil
	}
	// Try by ID
	id, err := strconv.ParseInt(identity, 10, 64)
	if err == nil {
		chat, err = b.Bot.ChatByID(id)
		if err == nil {
			return chat, nil
		}
	}
	return nil, err
}
