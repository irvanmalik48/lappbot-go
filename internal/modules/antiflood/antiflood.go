package antiflood

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v4"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Bot.Use(m.CheckFlood)
	m.Bot.Bot.Handle("/flood", m.handleFlood)
	m.Bot.Bot.Handle("/setflood", m.handleSetFlood)
	m.Bot.Bot.Handle("/setfloodtimer", m.handleSetFloodTimer)
	m.Bot.Bot.Handle("/floodmode", m.handleFloodMode)
	m.Bot.Bot.Handle("/clearflood", m.handleClearFlood)
}

func (m *Module) CheckFlood(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().Type == tele.ChatPrivate {
			return next(c)
		}
		if m.Bot.IsAdmin(c.Chat(), c.Sender()) {
			return next(c)
		}

		group, err := m.Store.GetGroup(c.Chat().ID)
		if err != nil || group == nil {
			return next(c)
		}

		if group.AntifloodConsecutiveLimit > 0 {
			key := fmt.Sprintf("flood:consecutive:%d:%d", c.Chat().ID, c.Sender().ID)
			val, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Incr().Key(key).Build()).AsInt64()
			m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Expire().Key(key).Seconds(5).Build())

			if val >= int64(group.AntifloodConsecutiveLimit) {
				m.takeAction(c, group)
				m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())
				return nil
			}
		}

		if group.AntifloodTimerLimit > 0 && group.AntifloodTimerDuration != "" {
			duration, _ := time.ParseDuration(group.AntifloodTimerDuration)
			if duration > 0 {
				key := fmt.Sprintf("flood:timer:%d:%d", c.Chat().ID, c.Sender().ID)

				val, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Incr().Key(key).Build()).AsInt64()

				if val == 1 {
					m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Expire().Key(key).Seconds(int64(duration.Seconds())).Build())
				}

				if val >= int64(group.AntifloodTimerLimit) {
					m.takeAction(c, group)
					m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())
					return nil
				}
			}
		}

		return next(c)
	}
}

func (m *Module) takeAction(c tele.Context, group *store.Group) {
	parts := strings.Split(group.AntifloodAction, " ")
	action := parts[0]
	duration := "1h"
	if len(parts) > 1 {
		duration = parts[1]
	}

	var err error
	var until time.Time

	switch action {
	case "ban":
		err = c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Sender()})
	case "kick":
		err = c.Bot().Unban(c.Chat(), c.Sender())
	case "mute":
		err = c.Bot().Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: tele.Rights{CanSendMessages: false}, RestrictedUntil: tele.Forever()})
	case "tban":
		d, _ := time.ParseDuration(duration)
		until = time.Now().Add(d)
		err = c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Sender(), RestrictedUntil: until.Unix()})
	case "tmute":
		d, _ := time.ParseDuration(duration)
		until = time.Now().Add(d)
		err = c.Bot().Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: tele.Rights{CanSendMessages: false}, RestrictedUntil: until.Unix()})
	default:
		err = c.Bot().Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: tele.Rights{CanSendMessages: false}, RestrictedUntil: tele.Forever()})
	}

	if err != nil {
		c.Send(fmt.Sprintf("Failed to execute flood action (%s) on %s: %v", action, c.Sender().FirstName, err))
		return
	}

	c.Send(fmt.Sprintf("Anti-flood triggered. Action: %s on %s.", action, c.Sender().FirstName))

	if group.AntifloodDelete {
		c.Delete()
	}
}

func (m *Module) handleFlood(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	info := fmt.Sprintf("**Antiflood Settings:**\n"+
		"Consecutive: %d\n"+
		"Timer: %d in %s\n"+
		"Action: %s\n"+
		"Clear Flood: %t",
		group.AntifloodConsecutiveLimit,
		group.AntifloodTimerLimit, group.AntifloodTimerDuration,
		group.AntifloodAction,
		group.AntifloodDelete)

	return c.Send(info, tele.ModeMarkdown)
}

func (m *Module) handleSetFlood(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /setflood <number/off>")
	}

	arg := strings.ToLower(args[0])
	val := 0
	if arg != "off" && arg != "no" {
		var err error
		val, err = strconv.Atoi(arg)
		if err != nil {
			return c.Send("Invalid number.")
		}
	}

	m.Store.SetAntifloodConsecutiveLimit(c.Chat().ID, val)
	return c.Send(fmt.Sprintf("Antiflood consecutive limit set to %v.", arg))
}

func (m *Module) handleSetFloodTimer(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /setfloodtimer <count> <duration> OR /setfloodtimer off")
	}

	if strings.ToLower(args[0]) == "off" || strings.ToLower(args[0]) == "no" {
		m.Store.SetAntifloodTimer(c.Chat().ID, 0, "")
		return c.Send("Timed antiflood disabled.")
	}

	if len(args) < 2 {
		return c.Send("Usage: /setfloodtimer <count> <duration>")
	}

	count, err := strconv.Atoi(args[0])
	if err != nil {
		return c.Send("Invalid count.")
	}

	_, err = time.ParseDuration(args[1])
	if err != nil {
		return c.Send("Invalid duration.")
	}

	m.Store.SetAntifloodTimer(c.Chat().ID, count, args[1])
	return c.Send(fmt.Sprintf("Timed antiflood set: %d messages in %s.", count, args[1]))
}

func (m *Module) handleFloodMode(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /floodmode <action> [duration]")
	}

	action := strings.Join(args, " ")
	m.Store.SetAntifloodAction(c.Chat().ID, action)
	return c.Send(fmt.Sprintf("Antiflood action set to: %s", action))
}

func (m *Module) handleClearFlood(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /clearflood <yes/no>")
	}

	arg := strings.ToLower(args[0])
	enabled := arg == "yes" || arg == "on"

	m.Store.SetAntifloodDelete(c.Chat().ID, enabled)
	return c.Send(fmt.Sprintf("Clear flood set to: %v", enabled))
}
