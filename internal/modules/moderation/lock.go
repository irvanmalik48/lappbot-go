package moderation

import (
	tele "gopkg.in/telebot.v4"
)

func (m *Module) handleLock(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	currentRights := c.Chat().Permissions
	currentRights.CanSendMessages = false

	err := m.Bot.Bot.SetGroupPermissions(c.Chat(), *currentRights)
	if err != nil {
		return c.Send("Failed to lock group: " + err.Error())
	}

	return c.Send("Group locked. Members cannot send messages.")
}

func (m *Module) handleUnlock(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	currentRights := c.Chat().Permissions
	currentRights.CanSendMessages = true

	err := m.Bot.Bot.SetGroupPermissions(c.Chat(), *currentRights)
	if err != nil {
		return c.Send("Failed to unlock group: " + err.Error())
	}

	return c.Send("Group unlocked. Members can send messages.")
}
