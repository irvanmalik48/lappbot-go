package antiraid

import (
	"context"
	"strconv"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"
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
	m.Bot.Handle("/antiraid", m.handleAntiraid)
	m.Bot.Handle("/raidtime", m.handleRaidTime)
	m.Bot.Handle("/raidactiontime", m.handleRaidActionTime)
	m.Bot.Handle("/autoantiraid", m.handleAutoAntiraid)
	m.Bot.Handle("new_chat_members", m.handleUserJoined)
}

func (m *Module) handleUserJoined(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, targetChat, c.Sender()) {
		return nil
	}

	group, err := m.Store.GetGroup(targetChat.ID)
	if err != nil || group == nil {
		return nil
	}

	if group.AntiraidUntil != nil && group.AntiraidUntil.After(time.Now()) {
		for _, u := range c.Update.Message.NewChatMembers {
			m.banUserRaw(targetChat.ID, u.ID, group.RaidActionTime)
			m.Logger.Log(targetChat.ID, "automated", "Antiraid banned user: "+u.FirstName+" (ID: "+strconv.FormatInt(u.ID, 10)+")")
		}
		return nil
	}

	if group.AutoAntiraidThreshold > 0 {
		key := "antiraid:joins:" + strconv.FormatInt(targetChat.ID, 10) + ":" + strconv.FormatInt(time.Now().Unix()/60, 10)
		val, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Incr().Key(key).Build()).AsInt64()
		m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Expire().Key(key).Seconds(65).Build())

		if val >= int64(group.AutoAntiraidThreshold) {
			until := time.Now().Add(6 * time.Hour)
			m.Store.SetAntiraidUntil(targetChat.ID, &until)
			c.Send("🚨 **ANTI-RAID AUTOMATICALLY ENABLED** 🚨\nMore than "+strconv.Itoa(group.AutoAntiraidThreshold)+" joins in the last minute.\nAnti-raid enabled for 6 hours.", "Markdown")
			m.Logger.Log(targetChat.ID, "automated", "Auto-Antiraid triggered. Threshold: "+strconv.Itoa(group.AutoAntiraidThreshold)+". Enabled for 6h.")

			for _, u := range c.Update.Message.NewChatMembers {
				m.banUserRaw(targetChat.ID, u.ID, group.RaidActionTime)
				m.Logger.Log(targetChat.ID, "automated", "Antiraid banned user: "+u.FirstName+" (ID: "+strconv.FormatInt(u.ID, 10)+")")
			}
		}
	}

	return nil
}

func (m *Module) banUserRaw(chatID, userID int64, durationStr string) error {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		duration = 1 * time.Hour
	}

	until := time.Now().Add(duration).Unix()
	return m.Bot.Raw("banChatMember", map[string]any{
		"chat_id":    chatID,
		"user_id":    userID,
		"until_date": until,
	})
}

func (m *Module) handleAntiraid(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /antiraid <time/off/no>")
	}

	arg := strings.ToLower(args[0])
	if arg == "off" || arg == "no" {
		m.Store.SetAntiraidUntil(c.Chat().ID, nil)
		m.Logger.Log(c.Chat().ID, "settings", "Antiraid disabled by "+c.Sender().FirstName)
		return c.Send("Anti-raid mode disabled.")
	}

	duration, err := time.ParseDuration(arg)
	if err != nil {
		return c.Send("Invalid duration format. Example: 3h, 30m.")
	}

	until := time.Now().Add(duration)
	m.Store.SetAntiraidUntil(c.Chat().ID, &until)
	m.Logger.Log(c.Chat().ID, "settings", "Antiraid enabled until "+until.Format(time.RFC822)+" by "+c.Sender().FirstName)
	return c.Send("Anti-raid enabled until " + until.Format(time.RFC822) + ".")
}

func (m *Module) handleRaidTime(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	return c.Send("Default raid duration is 6h. Please specify duration using /antiraid <time>.")
}

func (m *Module) handleRaidActionTime(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
	if len(args) == 0 {
		group, _ := m.Store.GetGroup(c.Chat().ID)
		return c.Send("Current raid action (ban) time: " + group.RaidActionTime)
	}

	duration := args[0]
	_, err := time.ParseDuration(duration)
	if err != nil {
		return c.Send("Invalid duration format.")
	}

	m.Store.SetRaidActionTime(c.Chat().ID, duration)
	m.Logger.Log(c.Chat().ID, "settings", "Raid action time set to "+duration+" by "+c.Sender().FirstName)
	return c.Send("Raid action time set to " + duration + ".")
}

func (m *Module) handleAutoAntiraid(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /autoantiraid <number/off/no>")
	}

	arg := strings.ToLower(args[0])
	if arg == "off" || arg == "no" {
		m.Store.SetAutoAntiraidThreshold(c.Chat().ID, 0)
		m.Logger.Log(c.Chat().ID, "settings", "Auto-Antiraid disabled by "+c.Sender().FirstName)
		return c.Send("Automatic anti-raid disabled.")
	}

	threshold, err := strconv.Atoi(arg)
	if err != nil || threshold < 0 {
		return c.Send("Invalid number.")
	}

	m.Store.SetAutoAntiraidThreshold(c.Chat().ID, threshold)
	m.Logger.Log(c.Chat().ID, "settings", "Auto-Antiraid set to "+strconv.Itoa(threshold)+" joins/min by "+c.Sender().FirstName)
	return c.Send("Automatic anti-raid set to trigger at " + strconv.Itoa(threshold) + " joins/minute.")
}
