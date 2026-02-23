package moderation

import (
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
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to warn them.")
	}

	target := c.Message.ReplyTo.From
	if m.Bot.IsAdmin(targetChat, target) {
		return c.Send("Cannot warn an admin.")
	}

	reason := c.Args
	reasonStr := "No reason provided"
	if len(reason) > 0 {
		reasonStr = strings.Join(reason, " ")
	}

	_, err = m.Store.AddWarn(target.ID, targetChat.ID, reasonStr, c.Sender().ID)
	if err != nil {
		return c.Send("Error adding warn: " + err.Error())
	}

	m.Logger.Log(targetChat.ID, "admin", "Warned "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")\nReason: "+reasonStr)

	if deleteMessage {
		m.Bot.Raw("deleteMessage", map[string]any{
			"chat_id":    targetChat.ID,
			"message_id": c.Message.ReplyTo.ID,
		})
		c.Delete()
	}

	return m.checkPunish(c, targetChat, target, reasonStr, silent)
}

func (m *Module) checkPunish(c *bot.Context, targetChat *bot.Chat, target *bot.User, reason string, silent bool) error {
	group, err := m.Store.GetGroup(targetChat.ID)
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

	count, err := m.Store.GetActiveWarns(target.ID, targetChat.ID, since)
	if err != nil {
		return err
	}

	msg := "User " + mention(target) + " has been warned.\nReason: " + reason + "\nTotal Warns: " + strconv.Itoa(count) + "/" + strconv.Itoa(group.WarnLimit)

	if count >= group.WarnLimit {
		m.Store.ResetWarns(target.ID, targetChat.ID)

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
			err = m.Bot.Raw("banChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID})
			m.Logger.Log(c.Chat().ID, "admin", "Warn removed from "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") by "+c.Sender().FirstName)
			msg += "\nAction: Banned."
		case "kick":
			err = m.Bot.Raw("unbanChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID})
			m.Logger.Log(c.Chat().ID, "admin", "Kicked "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") for reaching warn limit.")
			msg += "\nAction: Kicked."
		case "mute":
			permissions := map[string]bool{"can_send_messages": false}
			err = m.Bot.Raw("restrictChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID, "permissions": permissions, "until_date": 0})
			m.Logger.Log(c.Chat().ID, "admin", "Muted "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") for reaching warn limit.")
			msg += "\nAction: Muted."
		case "tban":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d).Unix()
			err = m.Bot.Raw("banChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID, "until_date": until})
			m.Logger.Log(c.Chat().ID, "admin", "Warns reset for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") by "+c.Sender().FirstName)
			msg += "\nAction: Banned for " + duration + "."
		case "tmute":
			d, _ := time.ParseDuration(duration)
			until := time.Now().Add(d).Unix()
			permissions := map[string]bool{"can_send_messages": false}
			err = m.Bot.Raw("restrictChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID, "permissions": permissions, "until_date": until})
			m.Logger.Log(c.Chat().ID, "admin", "Timed Mute for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") for reaching warn limit.\nDuration: "+duration)
			msg += "\nAction: Muted for " + duration + "."
		default:
			err = m.Bot.Raw("unbanChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": target.ID})
			m.Logger.Log(c.Chat().ID, "admin", "Kicked "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+") for reaching warn limit (Default).")
			msg += "\nAction: Kicked (Default)."
		}

		if err != nil {
			msg += "\nFailed to execute punishment."
		}
	} else {
		markup := &bot.ReplyMarkup{}
		btn := bot.InlineKeyboardButton{
			Text:         "Remove Warn",
			CallbackData: "btn_remove_warn|" + strconv.FormatInt(target.ID, 10),
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

	m.Logger.Log(c.Chat().ID, "admin", "All warns reset by "+c.Sender().FirstName)
	return c.Send("Last warn removed for " + targetName + ".")
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
	m.Logger.Log(c.Chat().ID, "admin", "Reset user warns for "+mention(target)+" (ID: "+strconv.FormatInt(target.ID, 10)+")")
	return c.Send("Warns reset for "+mention(target)+".", "Markdown")
}

func (m *Module) handleResetAllWarns(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	err := m.Store.ResetAllWarns(c.Chat().ID)
	if err != nil {
		return c.Send("Error resetting all warns: " + err.Error())
	}
	m.Logger.Log(c.Chat().ID, "admin", "Reset all warns in chat")
	return c.Send("All warnings in this chat have been reset.")
}

func (m *Module) handleWarnings(c *bot.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	msg := "**Warnings Settings:**\nLimit: " + strconv.Itoa(group.WarnLimit) + "\nAction: " + group.WarnAction + "\nDuration: " + group.WarnDuration
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
	return c.Send("Warn action set to: " + action)
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
	return c.Send("Warn limit set to: " + strconv.Itoa(limit))
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
	return c.Send("Warn duration set to: " + duration)
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
	return c.Send("You have " + strconv.Itoa(count) + "/" + strconv.Itoa(group.WarnLimit) + " warns.")
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
	m.Logger.Log(c.Chat().ID, "admin", "Removed warn for user ID "+strconv.FormatInt(targetID, 10)+" via button")
	return c.Respond("Warn removed.")
}
