package moderation

import (
	tele "gopkg.in/telebot.v4"
)

func (m *Module) handlePin(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
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
