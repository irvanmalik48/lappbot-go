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
	"lappbot/internal/modules/logging"
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

	logger := logging.New(b, st)
	logger.Register()

	captcha.New(b, st, logger).Register()
	filters.New(b, st, logger).Register()
	antiflood.New(b, st, logger).Register()
	antiraid.New(b, st, logger).Register()
	connections.New(b, st, logger).Register()
	utility.New(b, cfg, logger).Register()
	cursed.New(b, cfg, logger).Register()
	greeting.New(b, st, logger).Register()
	purge.New(b, st, logger).Register()
	moderation.New(b, st, logger).Register()
	notes.New(b, st, logger).Register()
	topics.New(b, cfg, logger).Register()

	if cfg.UseWebhook {
		b.StartWebhook()
	} else {
		b.StartLongPolling()
	}
}
