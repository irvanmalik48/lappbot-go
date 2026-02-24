package moderation

import (
	"lappbot/internal/bot"
)

func (m *Module) handlePin(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, targetChat, c.Sender(), "can_pin_messages") {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a message to pin/unpin it.")
	}

	err = m.Bot.Raw("pinChatMessage", map[string]any{
		"chat_id":    targetChat.ID,
		"message_id": c.Message.ReplyTo.ID,
	})
	if err != nil {
		return c.Send("Failed to pin message.")
	}

	m.Logger.Log(targetChat.ID, "admin", "Message pinned by "+c.Sender().FirstName)

	return nil
}
