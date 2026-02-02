package bot

import (
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
