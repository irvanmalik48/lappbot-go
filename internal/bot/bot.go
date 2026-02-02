package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"lappbot/internal/config"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	Bot   *tele.Bot
	Store *store.Store
	Cfg   *config.Config
}

func New(cfg *config.Config, store *store.Store) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Bot:   b,
		Store: store,
		Cfg:   cfg,
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
