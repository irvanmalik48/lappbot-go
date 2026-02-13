package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"lappbot/internal/bot"
	"lappbot/internal/config"
	"lappbot/internal/modules/antiflood"
	"lappbot/internal/modules/antiraid"
	"lappbot/internal/modules/captcha"
	"lappbot/internal/modules/connections"
	"lappbot/internal/modules/cursed"
	"lappbot/internal/modules/filters"
	"lappbot/internal/modules/greeting"
	"lappbot/internal/modules/moderation"
	"lappbot/internal/modules/notes"
	"lappbot/internal/modules/purge"
	"lappbot/internal/modules/topics"
	"lappbot/internal/modules/utility"
	"lappbot/internal/store"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	cfg := config.Load()

	st, err := store.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize store")
	}

	if err := store.RunMigrations(cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	b, err := bot.New(cfg, st)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create bot")
	}

	utility.New(b, cfg).Register()
	moderation.New(b, st).Register()
	greeting.New(b, st).Register()
	captcha.New(b, st).Register()
	filters.New(b, st).Register()
	antiflood.New(b, st).Register()
	antiraid.New(b, st).Register()
	connections.New(b, st).Register()
	topics.New(b, cfg).Register()
	cursed.New(b, cfg).Register()
	notes.New(b, st).Register()
	purge.New(b, st).Register()

	if cfg.UseWebhook {
		b.StartWebhook()
	} else {
		b.StartLongPolling()
	}
}
