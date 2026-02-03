package moderation

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handleWarn(c tele.Context) error {
	return m.warnUser(c, false, false)
}

func (m *Module) handleDWarn(c tele.Context) error {
	return m.warnUser(c, true, false)
}

func (m *Module) handleSWarn(c tele.Context) error {
	return m.warnUser(c, true, true)
}

func (m *Module) warnUser(c tele.Context, deleteMessage, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to warn them.")
	}

	target := c.Message().ReplyTo.Sender
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot warn an admin.")
	}

	reason := c.Args()
	reasonStr := "No reason provided"
	if len(reason) > 0 {
		reasonStr = ""
		for _, s := range reason {
			reasonStr += s + " "
		}
	}

	_, err := m.Store.AddWarn(target.ID, c.Chat().ID, reasonStr, c.Sender().ID)
	if err != nil {
		return c.Send("Error adding warn: " + err.Error())
	}

	if deleteMessage {
		c.Bot().Delete(c.Message().ReplyTo)
		c.Delete()
	}

	return m.checkPunish(c, target, reasonStr, silent)
}

func (m *Module) checkPunish(c tele.Context, target *tele.User, reason string, silent bool) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	var since time.Time
	if group.WarnDuration != "" && group.WarnDuration != "off" {
		d, err := time.ParseDuration(group.WarnDuration)
		if err == nil {
			since = time.Now().Add(-d)
		}
	}

	count, err := m.Store.GetActiveWarns(target.ID, c.Chat().ID, since)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("User %s has been warned.\nReason: %s\nTotal Warns: %d/%d", mention(target), reason, count, group.WarnLimit)

	if count >= group.WarnLimit {
		m.Store.ResetWarns(target.ID, c.Chat().ID)

		var err error
		action := group.WarnAction
		if action == "" {
			action = "kick"
		}

		parts := strings.Split(action, " ")
		actType := parts[0]
		duration := ""
		if len(parts) > 1 {
			duration = parts[1]
		}

		switch actType {
		case "ban":
			err = m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: target})
			msg += "\nAction: Banned."
		case "kick":
			err = m.Bot.Bot.Unban(c.Chat(), target)
			msg += "\nAction: Kicked."
		case "mute":
			err = m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: target, Rights: tele.Rights{CanSendMessages: false}, RestrictedUntil: tele.Forever()})
			msg += "\nAction: Muted."
		case "tban":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d)
			err = m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: target, RestrictedUntil: until.Unix()})
			msg += fmt.Sprintf("\nAction: Banned for %s.", duration)
		case "tmute":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d)
			err = m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: target, Rights: tele.Rights{CanSendMessages: false}, RestrictedUntil: until.Unix()})
			msg += fmt.Sprintf("\nAction: Muted for %s.", duration)
		default:
			err = m.Bot.Bot.Unban(c.Chat(), target)
			msg += "\nAction: Kicked (Default)."
		}

		if err != nil {
			msg += "\nFailed to execute punishment."
		}
	} else {
		markup := &tele.ReplyMarkup{}
		btnRemoveWarn := markup.Data("Remove Warn", "btn_remove_warn", fmt.Sprintf("%d", target.ID))
		markup.Inline(markup.Row(btnRemoveWarn))

		if !silent {
			return c.Send(msg, markup, tele.ModeMarkdown)
		}
		return nil
	}

	if !silent {
		return c.Send(msg, tele.ModeMarkdown)
	}
	return nil
}

func (m *Module) handleRmWarn(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	var targetID int64
	var targetName string

	if c.Message().IsReply() {
		targetID = c.Message().ReplyTo.Sender.ID
		targetName = c.Message().ReplyTo.Sender.FirstName
	} else {
		return c.Send("Reply to a user to remove their last warn.")
	}

	err := m.Store.RemoveLastWarn(targetID, c.Chat().ID)
	if err != nil {
		return c.Send("Error removing warn: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Last warn removed for %s.", targetName))
}

func (m *Module) handleResetWarns(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to reset their warns.")
	}
	target := c.Message().ReplyTo.Sender

	err := m.Store.ResetWarns(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Error resetting warns.")
	}
	return c.Send(fmt.Sprintf("Warns reset for %s.", mention(target)), tele.ModeMarkdown)
}

func (m *Module) handleResetAllWarns(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	err := m.Store.ResetAllWarns(c.Chat().ID)
	if err != nil {
		return c.Send("Error resetting all warns: " + err.Error())
	}
	return c.Send("All warnings in this chat have been reset.")
}

func (m *Module) handleWarnings(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("**Warnings Settings:**\nLimit: %d\nAction: %s\nDuration: %s", group.WarnLimit, group.WarnAction, group.WarnDuration)
	return c.Send(msg, tele.ModeMarkdown)
}

func (m *Module) handleWarnMode(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args()
	if len(args) == 0 {
		return m.handleWarnings(c)
	}

	action := strings.Join(args, " ")
	m.Store.SetWarnAction(c.Chat().ID, action)
	return c.Send(fmt.Sprintf("Warn action set to: %s", action))
}

func (m *Module) handleWarnLimit(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /warnlimit <number>")
	}

	limit, err := strconv.Atoi(args[0])
	if err != nil || limit < 1 {
		return c.Send("Invalid limit.")
	}

	m.Store.SetWarnLimit(c.Chat().ID, limit)
	return c.Send(fmt.Sprintf("Warn limit set to: %d", limit))
}

func (m *Module) handleWarnTime(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /warntime <duration/off>")
	}

	duration := args[0]
	if duration != "off" {
		_, err := time.ParseDuration(duration)
		if err != nil {
			return c.Send("Invalid duration.")
		}
	}

	m.Store.SetWarnDuration(c.Chat().ID, duration)
	return c.Send(fmt.Sprintf("Warn duration set to: %s", duration))
}

func (m *Module) handleMyWarns(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	var since time.Time
	if group.WarnDuration != "" && group.WarnDuration != "off" {
		d, err := time.ParseDuration(group.WarnDuration)
		if err == nil {
			since = time.Now().Add(-d)
		}
	}

	count, err := m.Store.GetActiveWarns(c.Sender().ID, c.Chat().ID, since)
	if err != nil {
		return c.Send("Error retrieving warns.")
	}
	return c.Send(fmt.Sprintf("You have %d/%d warns.", count, group.WarnLimit))
}

func (m *Module) onRemoveWarnBtn(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Respond(&tele.CallbackResponse{Text: "Admins only."})
	}

	targetIDStr := c.Data()
	targetID, _ := strconv.ParseInt(targetIDStr, 10, 64)

	err := m.Store.RemoveLastWarn(targetID, c.Chat().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Error removing warn."})
	}

	c.Delete()
	return c.Respond(&tele.CallbackResponse{Text: "Warn removed."})
}
