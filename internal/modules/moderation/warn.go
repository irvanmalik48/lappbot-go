package moderation

import (
	"fmt"
	"lappbot/internal/bot"
	"strconv"
	"strings"
	"time"
)

func (m *Module) handleWarn(c *bot.Context) error {
	return m.warnUser(c, false, false)
}

func (m *Module) handleDWarn(c *bot.Context) error {
	return m.warnUser(c, true, false)
}

func (m *Module) handleSWarn(c *bot.Context) error {
	return m.warnUser(c, true, true)
}

func (m *Module) warnUser(c *bot.Context, deleteMessage, silent bool) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to warn them.")
	}

	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(c.Chat(), target) {
		return c.Send("Cannot warn an admin.")
	}

	reason := c.Args
	reasonStr := "No reason provided"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	_, err := m.Store.AddWarn(target.ID, c.Chat().ID, reasonStr, c.Sender().ID)
	if err != nil {
		return c.Send("Error adding warn: " + err.Error())
	}

	if deleteMessage {
		m.Bot.Raw("deleteMessage", map[string]interface{}{
			"chat_id":    c.Chat().ID,
			"message_id": c.Message.ReplyTo.ID,
		})
		c.Delete()
	}

	return m.checkPunish(c, target, reasonStr, silent)
}

func (m *Module) checkPunish(c *bot.Context, target *bot.User, reason string, silent bool) error {
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
			err = m.Bot.Raw("banChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID})
			msg += "\nAction: Banned."
		case "kick":
			err = m.Bot.Raw("unbanChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID})
			msg += "\nAction: Kicked."
		case "mute":
			permissions := map[string]bool{"can_send_messages": false}
			err = m.Bot.Raw("restrictChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID, "permissions": permissions, "until_date": 0})
			msg += "\nAction: Muted."
		case "tban":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d).Unix()
			err = m.Bot.Raw("banChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID, "until_date": until})
			msg += fmt.Sprintf("\nAction: Banned for %s.", duration)
		case "tmute":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d).Unix()
			permissions := map[string]bool{"can_send_messages": false}
			err = m.Bot.Raw("restrictChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID, "permissions": permissions, "until_date": until})
			msg += fmt.Sprintf("\nAction: Muted for %s.", duration)
		default:
			err = m.Bot.Raw("unbanChatMember", map[string]interface{}{"chat_id": c.Chat().ID, "user_id": target.ID})
			msg += "\nAction: Kicked (Default)."
		}

		if err != nil {
			msg += "\nFailed to execute punishment."
		}
	} else {
		markup := &bot.ReplyMarkup{}
		btn := bot.InlineKeyboardButton{
			Text:         "Remove Warn",
			CallbackData: fmt.Sprintf("btn_remove_warn|%d", target.ID),
		}
		markup.InlineKeyboard = [][]bot.InlineKeyboardButton{{btn}}

		if !silent {
			return c.Send(msg, markup, "Markdown")
		}
		return nil
	}

	if !silent {
		return c.Send(msg, "Markdown")
	}
	return nil
}

func (m *Module) handleRmWarn(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	var targetID int64
	var targetName string

	if c.Message.ReplyTo != nil {
		targetID = c.Message.ReplyTo.From.ID
		targetName = c.Message.ReplyTo.From.FirstName
	} else {
		return c.Send("Reply to a user to remove their last warn.")
	}

	err := m.Store.RemoveLastWarn(targetID, c.Chat().ID)
	if err != nil {
		return c.Send("Error removing warn: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Last warn removed for %s.", targetName))
}

func (m *Module) handleResetWarns(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to reset their warns.")
	}
	target := c.Message.ReplyTo.From

	err := m.Store.ResetWarns(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Error resetting warns.")
	}
	return c.Send(fmt.Sprintf("Warns reset for %s.", mention(target)), "Markdown")
}

func (m *Module) handleResetAllWarns(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	err := m.Store.ResetAllWarns(c.Chat().ID)
	if err != nil {
		return c.Send("Error resetting all warns: " + err.Error())
	}
	return c.Send("All warnings in this chat have been reset.")
}

func (m *Module) handleWarnings(c *bot.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("**Warnings Settings:**\nLimit: %d\nAction: %s\nDuration: %s", group.WarnLimit, group.WarnAction, group.WarnDuration)
	return c.Send(msg, "Markdown")
}

func (m *Module) handleWarnMode(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args
	if len(args) == 0 {
		return m.handleWarnings(c)
	}

	action := strings.Join(args, " ")
	m.Store.SetWarnAction(c.Chat().ID, action)
	return c.Send(fmt.Sprintf("Warn action set to: %s", action))
}

func (m *Module) handleWarnLimit(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args
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

func (m *Module) handleWarnTime(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	args := c.Args
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

func (m *Module) handleMyWarns(c *bot.Context) error {
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

func (m *Module) onRemoveWarnBtn(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Respond("Admins only.")
	}

	parts := strings.Split(c.Data(), "|")
	if len(parts) < 2 {
		return c.Respond("Invalid data.")
	}

	targetIDStr := parts[1]
	targetID, _ := strconv.ParseInt(targetIDStr, 10, 64)

	err := m.Store.RemoveLastWarn(targetID, c.Chat().ID)
	if err != nil {
		return c.Respond("Error removing warn.")
	}

	c.Delete()
	return c.Respond("Warn removed.")
}
