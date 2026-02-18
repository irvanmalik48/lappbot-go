package antiflood

import (
	"context"
	"strconv"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"

	"github.com/valkey-io/valkey-go"
)

type Module struct {
	Bot    *bot.Bot
	Store  *store.Store
	Logger *logging.Module
}

func New(b *bot.Bot, s *store.Store, l *logging.Module) *Module {
	return &Module{Bot: b, Store: s, Logger: l}
}

func (m *Module) Register() {
	m.Bot.Use(m.CheckFlood)
	m.Bot.Handle("/flood", m.handleFlood)
	m.Bot.Handle("/setflood", m.handleSetFlood)
	m.Bot.Handle("/setfloodtimer", m.handleSetFloodTimer)
	m.Bot.Handle("/floodmode", m.handleFloodMode)
	m.Bot.Handle("/clearflood", m.handleClearFlood)
}

func (m *Module) CheckFlood(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
		if c.Chat().Type == "private" {
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
			key := "flood:consecutive:" + strconv.FormatInt(c.Chat().ID, 10) + ":" + strconv.FormatInt(c.Sender().ID, 10)

			cmds := make(valkey.Commands, 0, 2)
			cmds = append(cmds, m.Store.Valkey.B().Incr().Key(key).Build())
			cmds = append(cmds, m.Store.Valkey.B().Expire().Key(key).Seconds(5).Build())

			resps := m.Store.Valkey.DoMulti(context.Background(), cmds...)
			val, _ := resps[0].AsInt64()

			if val >= int64(group.AntifloodConsecutiveLimit) {
				m.takeAction(c, group)
				m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())
				return nil
			}
		}

		if group.AntifloodTimerLimit > 0 && group.AntifloodTimerDuration != "" {
			duration, _ := time.ParseDuration(group.AntifloodTimerDuration)
			if duration > 0 {
				key := "flood:timer:" + strconv.FormatInt(c.Chat().ID, 10) + ":" + strconv.FormatInt(c.Sender().ID, 10)

				script := `
			local val = redis.call("INCR", KEYS[1])
			if val == 1 then
				redis.call("EXPIRE", KEYS[1], ARGV[1])
			end
			return val
			`

				val, err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Eval().Script(script).Numkeys(1).Key(key).Arg(strconv.FormatInt(int64(duration.Seconds()), 10)).Build()).AsInt64()
				if err != nil {
					val = 0
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

func (m *Module) takeAction(c *bot.Context, group *store.Group) {
	parts := strings.Split(group.AntifloodAction, " ")
	action := parts[0]
	duration := "1h"
	if len(parts) > 1 {
		duration = parts[1]
	}

	var err error
	var until time.Time
	var permissions map[string]bool

	logMsg := "Antiflood triggered for " + c.Sender().FirstName + " (ID: " + strconv.FormatInt(c.Sender().ID, 10) + ")\nAction: " + action

	switch action {
	case "ban":
		err = c.Bot.Raw("banChatMember", map[string]any{
			"chat_id": c.Chat().ID,
			"user_id": c.Sender().ID,
		})
	case "kick":
		err = c.Bot.Raw("unbanChatMember", map[string]any{
			"chat_id": c.Chat().ID,
			"user_id": c.Sender().ID,
		})
	case "mute":
		permissions = map[string]bool{"can_send_messages": false}
		err = c.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     c.Chat().ID,
			"user_id":     c.Sender().ID,
			"permissions": permissions,
			"until_date":  0,
		})
	case "tban":
		d, _ := time.ParseDuration(duration)
		until = time.Now().Add(d)
		err = c.Bot.Raw("banChatMember", map[string]any{
			"chat_id":    c.Chat().ID,
			"user_id":    c.Sender().ID,
			"until_date": until.Unix(),
		})
		logMsg += "\nDuration: " + duration
	case "tmute":
		d, _ := time.ParseDuration(duration)
		until = time.Now().Add(d)
		permissions = map[string]bool{"can_send_messages": false}
		err = c.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     c.Chat().ID,
			"user_id":     c.Sender().ID,
			"permissions": permissions,
			"until_date":  until.Unix(),
		})
		logMsg += "\nDuration: " + duration
	default:
		permissions = map[string]bool{"can_send_messages": false}
		err = c.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     c.Chat().ID,
			"user_id":     c.Sender().ID,
			"permissions": permissions,
			"until_date":  0,
		})
	}

	if err != nil {
		c.Send("Failed to execute flood action (" + action + ") on " + c.Sender().FirstName + ": " + err.Error())
		return
	}

	m.Logger.Log(c.Chat().ID, "automated", logMsg)

	c.Send("Anti-flood triggered. Action: " + action + " on " + c.Sender().FirstName + ".")

	if group.AntifloodDelete {
		c.Delete()
	}
}

func (m *Module) handleFlood(c *bot.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	info := "**Antiflood Settings:**\n" +
		"Consecutive: " + strconv.Itoa(group.AntifloodConsecutiveLimit) + "\n" +
		"Timer: " + strconv.Itoa(group.AntifloodTimerLimit) + " in " + group.AntifloodTimerDuration + "\n" +
		"Action: " + group.AntifloodAction + "\n" +
		"Clear Flood: " + strconv.FormatBool(group.AntifloodDelete)

	return c.Send(info, "Markdown")
}

func (m *Module) handleSetFlood(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args
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
	m.Logger.Log(c.Chat().ID, "settings", "Antiflood consecutive limit set to "+arg+" by "+c.Sender().FirstName)
	return c.Send("Antiflood consecutive limit set to " + arg + ".")
}

func (m *Module) handleSetFloodTimer(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /setfloodtimer <count> <duration> OR /setfloodtimer off")
	}

	if strings.ToLower(args[0]) == "off" || strings.ToLower(args[0]) == "no" {
		m.Store.SetAntifloodTimer(c.Chat().ID, 0, "")
		m.Logger.Log(c.Chat().ID, "settings", "Timed antiflood disabled by "+c.Sender().FirstName)
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
	m.Logger.Log(c.Chat().ID, "settings", "Timed antiflood set to "+strconv.Itoa(count)+" in "+args[1]+" by "+c.Sender().FirstName)
	return c.Send("Timed antiflood set: " + strconv.Itoa(count) + " messages in " + args[1] + ".")
}

func (m *Module) handleFloodMode(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /floodmode <action> [duration]")
	}

	action := strings.Join(args, " ")
	m.Store.SetAntifloodAction(c.Chat().ID, action)
	m.Logger.Log(c.Chat().ID, "settings", "Antiflood action set to "+action+" by "+c.Sender().FirstName)
	return c.Send("Antiflood action set to: " + action)
}

func (m *Module) handleClearFlood(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("Admin only.")
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /clearflood <yes/no>")
	}

	arg := strings.ToLower(args[0])
	enabled := arg == "yes" || arg == "on"

	m.Store.SetAntifloodDelete(c.Chat().ID, enabled)
	m.Logger.Log(c.Chat().ID, "settings", "Antiflood message deletion set to "+strconv.FormatBool(enabled)+" by "+c.Sender().FirstName)
	return c.Send("Clear flood set to: " + strconv.FormatBool(enabled))
}
