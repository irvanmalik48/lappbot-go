package moderation

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handleMute(c tele.Context) error {
	return m.muteUser(c, false)
}

func (m *Module) handleSilentMute(c tele.Context) error {
	return m.muteUser(c, true)
}

func (m *Module) muteUser(c tele.Context, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to mute them.")
	}
	target := c.Message().ReplyTo.Sender
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot mute an admin.")
	}

	reason := c.Args()
	reasonStr := "Manual Mute"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	rights := tele.Rights{
		CanSendMessages: false,
		CanSendMedia:    false,
		CanSendPolls:    false,
		CanSendOther:    false,
	}

	m.Store.BanUser(target.ID, c.Chat().ID, time.Time{}, reasonStr, c.Sender().ID, "mute")

	err := m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: target, Rights: rights})
	if err != nil {
		return c.Send("Error muting user: " + err.Error())
	}

	if silent {
		c.Delete()
		return nil
	}
	return c.Send(fmt.Sprintf("%s muted.\nReason: %s", mention(target), reasonStr), tele.ModeMarkdown)
}

func (m *Module) handleTimedMute(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args()
	if len(args) < 1 {
		return c.Send("Usage: /tmute <duration> [reason] (Reply to user)")
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to mute them.")
	}
	target := c.Message().ReplyTo.Sender
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

	rights := tele.Rights{
		CanSendMessages: false,
		CanSendMedia:    false,
		CanSendPolls:    false,
		CanSendOther:    false,
	}

	m.Store.BanUser(target.ID, c.Chat().ID, until, reasonStr, c.Sender().ID, "mute")

	err = m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: target, Rights: rights, RestrictedUntil: until.Unix()})
	if err != nil {
		return c.Send("Error muting user: " + err.Error())
	}

	return c.Send(fmt.Sprintf("%s muted for %s.\nReason: %s", mention(target), durationStr, reasonStr), tele.ModeMarkdown)
}

func (m *Module) handleRealmMute(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	if !c.Message().IsReply() {
		return c.Send("Reply to a user to realm mute them.")
	}
	target := c.Message().ReplyTo.Sender

	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot realm mute an admin of this group.")
	}

	reason := c.Args()
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

	rights := tele.Rights{
		CanSendMessages: false,
		CanSendMedia:    false,
		CanSendPolls:    false,
		CanSendOther:    false,
	}

	for _, g := range groups {
		chat := &tele.Chat{ID: g.TelegramID}

		m.Store.BanUser(target.ID, g.TelegramID, time.Time{}, reasonStr, c.Sender().ID, "mute")

		err := m.Bot.Bot.Restrict(chat, &tele.ChatMember{User: target, Rights: rights})
		if err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	return c.Send(fmt.Sprintf("Realm Mute Executed.\nTarget: %s\nMuted in: %d groups\nFailed in: %d groups\nReason: %s",
		mention(target), successCount, failCount, reasonStr), tele.ModeMarkdown)
}
