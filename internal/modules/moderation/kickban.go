package moderation

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handleKick(c tele.Context) error {
	return m.kickUser(c, false)
}

func (m *Module) handleSilentKick(c tele.Context) error {
	return m.kickUser(c, true)
}

func (m *Module) kickUser(c tele.Context, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to kick them.")
	}
	target := c.Message().ReplyTo.Sender
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot kick an admin.")
	}

	reason := c.Args()
	reasonStr := "No reason provided"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	err := m.Bot.Bot.Unban(c.Chat(), target)
	if err != nil {
		return c.Send("Error kicking user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	return c.Send(fmt.Sprintf("%s kicked.\nReason: %s", mention(target), reasonStr), tele.ModeMarkdown)
}

func (m *Module) handleBan(c tele.Context) error {
	return m.banUser(c, false)
}

func (m *Module) handleSilentBan(c tele.Context) error {
	return m.banUser(c, true)
}

func (m *Module) banUser(c tele.Context, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to ban them.")
	}
	target := c.Message().ReplyTo.Sender
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot ban an admin.")
	}

	reason := c.Args()
	reasonStr := "Manual Ban"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	m.Store.BanUser(target.ID, c.Chat().ID, time.Time{}, reasonStr, c.Sender().ID, "ban")

	err := m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: target})
	if err != nil {
		return c.Send("Error banning user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	return c.Send(fmt.Sprintf("%s banned.\nReason: %s", mention(target), reasonStr), tele.ModeMarkdown)
}

func (m *Module) handleTimedBan(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args()
	if len(args) < 1 {
		return c.Send("Usage: /tban <duration> [reason] (Reply to user)")
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to ban them.")
	}
	target := c.Message().ReplyTo.Sender
	if m.Bot.IsAdmin(c.Chat(), target) {
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

	m.Store.BanUser(target.ID, c.Chat().ID, until, reasonStr, c.Sender().ID, "ban")

	err = m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: target, RestrictedUntil: until.Unix()})
	if err != nil {
		return c.Send("Error banning user: " + err.Error())
	}

	return c.Send(fmt.Sprintf("%s banned for %s.\nReason: %s", mention(target), durationStr, reasonStr), tele.ModeMarkdown)
}

func (m *Module) handleRealmBan(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	if !c.Message().IsReply() {
		return c.Send("Reply to a user to realm ban them.")
	}
	target := c.Message().ReplyTo.Sender

	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot realm ban an admin of this group.")
	}

	reason := c.Args()
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
		chat := &tele.Chat{ID: g.TelegramID}

		m.Store.BanUser(target.ID, g.TelegramID, time.Time{}, reasonStr, c.Sender().ID, "ban")

		err := m.Bot.Bot.Ban(chat, &tele.ChatMember{User: target})
		if err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	return c.Send(fmt.Sprintf("Realm Ban Executed.\nTarget: %s\nBanned in: %d groups\nFailed in: %d groups\nReason: %s",
		mention(target), successCount, failCount, reasonStr), tele.ModeMarkdown)
}

func getAdditionalArgss(args []string, start int) string {
	if len(args) <= start {
		return ""
	}
	return strings.Join(args[start:], " ")
}
