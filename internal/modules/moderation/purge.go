package moderation

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handlePurge(c tele.Context) error {
	if !m.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a message to purge up to it.")
	}

	replyID := c.Message().ReplyTo.ID
	currentID := c.Message().ID

	c.Delete()

	var ids []int
	for i := currentID - 1; i >= replyID; i-- {
		ids = append(ids, i)
		if len(ids) >= 100 {
			break
		}
	}

	payload := map[string]interface{}{
		"chat_id":     c.Chat().ID,
		"message_ids": ids,
	}

	_, err := c.Bot().Raw("deleteMessages", payload)
	if err != nil {
		return c.Send("Failed to purge messages: " + err.Error())
	}

	msg, err := c.Bot().Send(c.Chat(), "Purged messages.")
	if err == nil {
		go func() {
			time.Sleep(3 * time.Second)
			c.Bot().Delete(msg)
		}()
	}
	return nil
}
