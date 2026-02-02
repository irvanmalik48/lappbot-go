package moderation

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handleWarn(c tele.Context) error {
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

	count, err := m.Store.AddWarn(target.ID, c.Chat().ID, reasonStr, c.Sender().ID)
	if err != nil {
		return c.Send("Error adding warn: " + err.Error())
	}

	msg := fmt.Sprintf("User %s has been warned.\nReason: %s\nTotal Warns: %d/3", mention(target), reasonStr, count)

	if count >= 3 {
		err := m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: target})
		if err != nil {
			msg += "\nFailed to ban user (limit reached)."
		} else {
			msg += "\nUser banned (limit reached)."
			m.Bot.Bot.Unban(c.Chat(), target)
			m.Store.ResetWarns(target.ID, c.Chat().ID)
		}
		return c.Send(msg, tele.ModeMarkdown)
	}

	markup := &tele.ReplyMarkup{}
	btnRemoveWarn := markup.Data("Remove Warn", "btn_remove_warn", fmt.Sprintf("%d", target.ID))
	markup.Inline(markup.Row(btnRemoveWarn))

	return c.Send(msg, markup, tele.ModeMarkdown)
}

func (m *Module) handleUnwarn(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to remove their last warn.")
	}
	target := c.Message().ReplyTo.Sender

	err := m.Store.RemoveLastWarn(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Error removing warn: " + err.Error())
	}

	count, err := m.Store.GetWarnCount(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Warn removed, but failed to get new count.")
	}

	return c.Send(fmt.Sprintf("Warn removed for %s.\nTotal Warns: %d/3", mention(target), count), tele.ModeMarkdown)
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

func (m *Module) handleMyWarns(c tele.Context) error {
	count, err := m.Store.GetWarnCount(c.Sender().ID, c.Chat().ID)
	if err != nil {
		return c.Send("Error retrieving warns.")
	}
	return c.Send(fmt.Sprintf("You have %d warns.", count))
}
