package moderation

import (
	tele "gopkg.in/telebot.v3"
)

func (m *Module) handlePin(c tele.Context) error {
	if !m.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a message to pin/unpin it.")
	}

	msg := c.Message().ReplyTo
	err := c.Bot().Pin(msg)
	if err != nil {
		return c.Send("Failed to pin: " + err.Error())
	}

	return nil
}
