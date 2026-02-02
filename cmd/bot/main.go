package main

import (
	"log"

	"lappbot/internal/bot"
	"lappbot/internal/config"
	"lappbot/internal/modules/captcha"
	"lappbot/internal/modules/filters"
	"lappbot/internal/modules/greeting"
	"lappbot/internal/modules/moderation"
	"lappbot/internal/modules/utility"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v3"
)

func main() {
	cfg := config.Load()

	if err := store.RunMigrations(cfg); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	db, err := store.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	b, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	greetingModule := greeting.New(b, db)
	greetingModule.Register()

	captchaModule := captcha.New(b, db)
	captchaModule.Register()

	moderationModule := moderation.New(b, db)
	moderationModule.Register()

	utilityModule := utility.New(b, cfg)
	utilityModule.Register()

	filtersModule := filters.NewFilters(b, db)
	filtersModule.Register()

	b.Bot.Handle(tele.OnUserJoined, func(c tele.Context) error {
		if err := greetingModule.OnUserJoined(c); err != nil {
			log.Println("Greeting error:", err)
		}
		if err := captchaModule.OnUserJoined(c); err != nil {
			log.Println("Captcha error:", err)
		}
		return nil
	})

	b.Bot.Handle(tele.OnUserLeft, func(c tele.Context) error {
		if err := greetingModule.OnUserLeft(c); err != nil {
			log.Println("Goodbye error:", err)
		}
		return nil
	})

	b.Start()
}
