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

	msgID := c.Message.ReplyTo.ID

	err := m.Bot.Raw("pinChatMessage", map[string]any{
		"chat_id":    c.Chat().ID,
		"message_id": msgID,
	})
	if err != nil {
		return c.Send("Failed to pin: " + err.Error())
	}

	return nil
}
