package moderation

import (
	"lappbot/internal/bot"
	"strconv"
	"strings"
	"time"
)

func (m *Module) handleMute(c *bot.Context) error {
	return m.muteUser(c, false)
}

func (m *Module) handleSilentMute(c *bot.Context) error {
	return m.muteUser(c, true)
}

func (m *Module) muteUser(c *bot.Context, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to mute them.")
	}
	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot mute an admin.")
	}

	reason := c.Args
	reasonStr := "Manual Mute"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	m.Store.BanUser(target.ID, c.Chat().ID, time.Time{}, reasonStr, c.Sender().ID, "mute")

	permissions := map[string]bool{
		"can_send_messages":       false,
		"can_send_media_messages": false,
		"can_send_polls":          false,
		"can_send_other_messages": false,
	}

	err := m.Bot.Raw("restrictChatMember", map[string]any{
		"chat_id":     c.Chat().ID,
		"user_id":     target.ID,
		"permissions": permissions,
		"until_date":  0,
	})
	if err != nil {
		return c.Send("Error muting user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	m.Logger.Log(c.Chat().ID, "admin", "Muted "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)
	return c.Send(mention(target)+" muted.\nReason: "+reasonStr, "Markdown")
}

func (m *Module) handleUnmute(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to unmute them.")
	}
	target := c.Message.ReplyTo.From

	permissions := map[string]bool{
		"can_send_messages":         true,
		"can_send_media_messages":   true,
		"can_send_polls":            true,
		"can_send_other_messages":   true,
		"can_add_web_page_previews": true,
		"can_invite_users":          true,
	}

	err := m.Bot.Raw("restrictChatMember", map[string]any{
		"chat_id":     c.Chat().ID,
		"user_id":     target.ID,
		"permissions": permissions,
	})
	if err != nil {
		return c.Send("Failed to unmute user: " + err.Error())
	}

	m.Logger.Log(c.Chat().ID, "admin", "Unmuted "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")")
	return c.Send(mention(target)+" unmuted.", "Markdown")
}

func (m *Module) handleTimedMute(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) < 1 {
		return c.Send("Usage: /tmute <duration> [reason] (Reply to user)")
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to mute them.")
	}
	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot mute an admin.")
	}

	durationStr := args[0]
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return c.Send("Invalid duration format (e.g., 1h, 30m).")
	}

	until := time.Now().Add(duration)
	reasonStr := "Timed Mute"
	if len(args) > 1 {
		reasonStr = strings.Join(args[1:], " ")
	}

	m.Store.BanUser(target.ID, c.Chat().ID, until, reasonStr, c.Sender().ID, "mute")

	permissions := map[string]bool{
		"can_send_messages":       false,
		"can_send_media_messages": false,
		"can_send_polls":          false,
		"can_send_other_messages": false,
	}

	err = m.Bot.Raw("restrictChatMember", map[string]any{
		"chat_id":     c.Chat().ID,
		"user_id":     target.ID,
		"permissions": permissions,
		"until_date":  until.Unix(),
	})
	if err != nil {
		return c.Send("Error muting user: " + err.Error())
	}

	m.Logger.Log(c.Chat().ID, "mute", "Timed Mute for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nDuration: "+durationStr+"\nReason: "+reasonStr)
	return c.Send(mention(target)+" muted for "+durationStr+".\nReason: "+reasonStr, "Markdown")
}

func (m *Module) handleRealmMute(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to realm mute them.")
	}
	target := c.Message.ReplyTo.From

	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot realm mute an admin of this group.")
	}

	reason := c.Args
	reasonStr := "Realm Mute"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	groups, err := m.Store.GetAllGroups()
	if err != nil {
		return c.Send("Failed to fetch groups: " + err.Error())
	}

	successCount := 0
	failCount := 0

	permissions := map[string]bool{
		"can_send_messages":       false,
		"can_send_media_messages": false,
		"can_send_polls":          false,
		"can_send_other_messages": false,
	}

	for _, g := range groups {
		m.Store.BanUser(target.ID, g.TelegramID, time.Time{}, reasonStr, c.Sender().ID, "mute")

		err := m.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     g.TelegramID,
			"user_id":     target.ID,
			"permissions": permissions,
			"until_date":  0,
		})
		if err == nil {
			m.Logger.Log(g.TelegramID, "mute", "Realm Mute for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)
			successCount++
		} else {
			failCount++
		}
	}

	return c.Send("Realm Mute Executed.\nTarget: "+mention(target)+"\nMuted in: "+strconv.Itoa(successCount)+" groups\nFailed in: "+strconv.Itoa(failCount)+" groups\nReason: "+reasonStr, "Markdown")
}
