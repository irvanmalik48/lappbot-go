package moderation

import (
	"lappbot/internal/bot"
	"strconv"
	"strings"
	"time"
)

func (m *Module) handleKick(c *bot.Context) error {
	return m.kickUser(c, false)
}

func (m *Module) handleSilentKick(c *bot.Context) error {
	return m.kickUser(c, true)
}

func (m *Module) kickUser(c *bot.Context, silent bool) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to kick them.")
	}
	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(targetChat, target) {
		return c.Send("Cannot kick an admin.")
	}

	reason := c.Args
	reasonStr := "No reason provided"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	err = m.Bot.Raw("unbanChatMember", map[string]any{
		"chat_id": targetChat.ID,
		"user_id": target.ID,
	})
	if err != nil {
		return c.Send("Error kicking user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	m.Logger.Log(targetChat.ID, "admin", "Kicked "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)
	return c.Send(mention(target)+" kicked.\nReason: "+reasonStr, "Markdown")
}

func (m *Module) handleBan(c *bot.Context) error {
	return m.banUser(c, false)
}

func (m *Module) handleSilentBan(c *bot.Context) error {
	return m.banUser(c, true)
}

func (m *Module) banUser(c *bot.Context, silent bool) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to ban them.")
	}
	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(targetChat, target) {
		return c.Send("Cannot ban an admin.")
	}

	reason := c.Args
	reasonStr := "Manual Ban"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	m.Store.BanUser(target.ID, targetChat.ID, time.Time{}, reasonStr, c.Sender().ID, "ban")

	err = m.Bot.Raw("banChatMember", map[string]any{
		"chat_id": targetChat.ID,
		"user_id": target.ID,
	})
	if err != nil {
		return c.Send("Error banning user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	m.Logger.Log(targetChat.ID, "admin", "Banned "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)
	return c.Send(mention(target)+" banned.\nReason: "+reasonStr, "Markdown")
}

func (m *Module) handleUnban(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to unban them.")
	}
	target := c.Message.ReplyTo.From

	err = m.Bot.Raw("unbanChatMember", map[string]any{
		"chat_id":        targetChat.ID,
		"user_id":        target.ID,
		"only_if_banned": true,
	})
	if err != nil {
		return c.Send("Failed to unban user: " + err.Error())
	}

	m.Logger.Log(targetChat.ID, "admin", "Unbanned "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")")
	return c.Send(mention(target)+" unbanned.", "Markdown")
}

func (m *Module) handleTimedBan(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) < 1 {
		return c.Send("Usage: /tban <duration> [reason] (Reply to user)")
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to ban them.")
	}
	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(targetChat, target) {
		return c.Send("Cannot ban an admin.")
	}

	durationStr := args[0]
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return c.Send("Invalid duration format (e.g., 1h, 30m).")
	}

	until := time.Now().Add(duration)
	reasonStr := "Timed Ban"
	if len(args) > 1 {
		reasonStr = strings.Join(args[1:], " ")
	}

	m.Store.BanUser(target.ID, targetChat.ID, until, reasonStr, c.Sender().ID, "ban")

	err = m.Bot.Raw("banChatMember", map[string]any{
		"chat_id":    targetChat.ID,
		"user_id":    target.ID,
		"until_date": until.Unix(),
	})
	if err != nil {
		return c.Send("Error banning user: " + err.Error())
	}

	m.Logger.Log(targetChat.ID, "admin", "Timed Ban for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nDuration: "+durationStr+"\nReason: "+reasonStr)
	return c.Send(mention(target)+" banned for "+durationStr+".\nReason: "+reasonStr, "Markdown")
}

func (m *Module) handleRealmBan(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}

	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to realm ban them.")
	}
	target := c.Message.ReplyTo.From

	if m.Bot.IsAdmin(targetChat, target) {
		return c.Send("Cannot realm ban an admin of this group.")
	}

	reason := c.Args
	reasonStr := "Realm Ban"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	groups, err := m.Store.GetAllGroups()
	if err != nil {
		return c.Send("Failed to fetch groups: " + err.Error())
	}

	successCount := 0
	failCount := 0

	for _, g := range groups {
		m.Store.BanUser(target.ID, g.TelegramID, time.Time{}, reasonStr, c.Sender().ID, "ban")

		err := m.Bot.Raw("banChatMember", map[string]any{
			"chat_id": g.TelegramID,
			"user_id": target.ID,
		})
		if err == nil {
			m.Logger.Log(g.TelegramID, "admin", "Realm Ban for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)
			successCount++
		} else {
			failCount++
		}
	}

	return c.Send("Realm Ban Executed.\nTarget: "+mention(target)+"\nBanned in: "+strconv.Itoa(successCount)+" groups\nFailed in: "+strconv.Itoa(failCount)+" groups\nReason: "+reasonStr, "Markdown")
}
