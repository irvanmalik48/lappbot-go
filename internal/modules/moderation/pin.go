package moderation

import (
	"lappbot/internal/bot"
)

func (m *Module) handlePin(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a message to pin/unpin it.")
	}

	err := m.Bot.Raw("pinChatMessage", map[string]any{
		"chat_id":    c.Chat().ID,
		"message_id": c.Message.ReplyTo.ID,
	})
	if err != nil {
		return c.Send("Failed to pin message.")
	}

	m.Logger.Log(c.Chat().ID, "admin", "Message pinned by "+c.Sender().FirstName)

	return nil
}
