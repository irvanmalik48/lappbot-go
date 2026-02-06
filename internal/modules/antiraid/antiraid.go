package antiraid

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
	m.Bot.Bot.Handle("/antiraid", m.handleAntiraid)
	m.Bot.Bot.Handle("/raidtime", m.handleRaidTime)
	m.Bot.Bot.Handle("/raidactiontime", m.handleRaidActionTime)
	m.Bot.Bot.Handle("/autoantiraid", m.handleAutoAntiraid)
	m.Bot.Bot.Handle(tele.OnUserJoined, m.handleUserJoined)
}

func (m *Module) handleUserJoined(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil || group == nil {
		return nil
	}

	if group.AntiraidUntil != nil && group.AntiraidUntil.After(time.Now()) {
		return m.banUser(c, group.RaidActionTime)
	}

	if group.AutoAntiraidThreshold > 0 {
		key := fmt.Sprintf("antiraid:joins:%d:%d", c.Chat().ID, time.Now().Unix()/60)
		val, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Incr().Key(key).Build()).AsInt64()
		m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Expire().Key(key).Seconds(65).Build())

		if val >= int64(group.AutoAntiraidThreshold) {
			until := time.Now().Add(6 * time.Hour)
			m.Store.SetAntiraidUntil(c.Chat().ID, &until)
			c.Send(fmt.Sprintf("🚨 **ANTI-RAID AUTOMATICALLY ENABLED** 🚨\nMore than %d joins in the last minute.\nAnti-raid enabled for 6 hours.", group.AutoAntiraidThreshold), tele.ModeMarkdown)

			return m.banUser(c, group.RaidActionTime)
		}
	}

	return nil
}

func (m *Module) banUser(c tele.Context, durationStr string) error {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		duration = 1 * time.Hour
	}

	until := time.Now().Add(duration)
	err = c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Sender(), RestrictedUntil: until.Unix()})
	if err == nil {
		c.Delete()
	}
	return err
}

func (m *Module) handleAntiraid(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /antiraid <time/off/no>")
	}

	arg := strings.ToLower(args[0])
	if arg == "off" || arg == "no" {
		m.Store.SetAntiraidUntil(c.Chat().ID, nil)
		return c.Send("Anti-raid mode disabled.")
	}

	duration, err := time.ParseDuration(arg)
	if err != nil {
		return c.Send("Invalid duration format. Example: 3h, 30m.")
	}

	until := time.Now().Add(duration)
	m.Store.SetAntiraidUntil(c.Chat().ID, &until)
	return c.Send(fmt.Sprintf("Anti-raid enabled until %s.", until.Format(time.RFC822)))
}

func (m *Module) handleRaidTime(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	return c.Send("Default raid duration is 6h. Please specify duration using /antiraid <time>.")
}

func (m *Module) handleRaidActionTime(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args()
	if len(args) == 0 {
		group, _ := m.Store.GetGroup(c.Chat().ID)
		return c.Send(fmt.Sprintf("Current raid action (ban) time: %s", group.RaidActionTime))
	}

	duration := args[0]
	_, err := time.ParseDuration(duration)
	if err != nil {
		return c.Send("Invalid duration format.")
	}

	m.Store.SetRaidActionTime(c.Chat().ID, duration)
	return c.Send(fmt.Sprintf("Raid action time set to %s.", duration))
}

func (m *Module) handleAutoAntiraid(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /autoantiraid <number/off/no>")
	}

	arg := strings.ToLower(args[0])
	if arg == "off" || arg == "no" {
		m.Store.SetAutoAntiraidThreshold(c.Chat().ID, 0)
		return c.Send("Automatic anti-raid disabled.")
	}

	threshold, err := strconv.Atoi(arg)
	if err != nil || threshold < 0 {
		return c.Send("Invalid number.")
	}

	m.Store.SetAutoAntiraidThreshold(c.Chat().ID, threshold)
	return c.Send(fmt.Sprintf("Automatic anti-raid set to trigger at %d joins/minute.", threshold))
}
