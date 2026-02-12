package purge

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	"github.com/valkey-io/valkey-go"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Handle("/purge", m.handlePurge)
	m.Bot.Handle("/spurge", m.handlePurge)
	m.Bot.Handle("/del", m.handleDel)
	m.Bot.Handle("/purgefrom", m.handlePurgeFrom)
	m.Bot.Handle("/purgeto", m.handlePurgeTo)
}

func (m *Module) deleteMessages(chatID int64, messageIDs []int) {
	if len(messageIDs) == 0 {
		return
	}

	batchSize := 100
	for i := 0; i < len(messageIDs); i += batchSize {
		end := i + batchSize
		if end > len(messageIDs) {
			end = len(messageIDs)
		}
		batch := messageIDs[i:end]

		req := map[string]any{
			"chat_id":     chatID,
			"message_ids": batch,
		}
		m.Bot.Raw("deleteMessages", req)
	}
}

func (m *Module) handlePurge(c *bot.Context) error {
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a message to purge from.")
	}
	args := c.Args
	limit := 0
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}

	startID := int(c.Message.ReplyTo.ID)
	endID := int(c.Message.ID)

	var toDelete []int
	if limit > 0 {
		for i := 1; i <= limit; i++ {
			toDelete = append(toDelete, startID+i)
		}
	} else {
		for i := startID; i < endID; i++ {
			toDelete = append(toDelete, i)
		}
	}
	toDelete = append(toDelete, endID)

	m.deleteMessages(c.Chat().ID, toDelete)

	if c.Message.Text != "" && (c.Message.Text == "/spurge" || len(c.Message.Text) > 7 && c.Message.Text[:7] == "/spurge") {
		return nil
	}

	c.Send("Purge complete.")
	return nil
}

func (m *Module) handleDel(c *bot.Context) error {
	if c.Message.ReplyTo == nil {
		return nil
	}
	m.deleteMessages(c.Chat().ID, []int{int(c.Message.ReplyTo.ID)})
	c.Delete()
	return nil
}

func (m *Module) handlePurgeFrom(c *bot.Context) error {
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a message to mark as purge start.")
	}
	key := fmt.Sprintf("purgefrom:%d", c.Chat().ID)
	m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(key).Value(strconv.Itoa(int(c.Message.ReplyTo.ID))).Ex(time.Minute*5).Build())

	c.Delete()
	c.Send("Purge start marked. Reply to another message with /purgeto to purge range.")
	return nil
}

func (m *Module) handlePurgeTo(c *bot.Context) error {
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a message to mark as purge end.")
	}
	key := fmt.Sprintf("purgefrom:%d", c.Chat().ID)
	res, err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return c.Send("No purge start point found. Use /purgefrom first.")
		}
		return c.Send("Failed to get purge start point.")
	}
	startID, _ := strconv.Atoi(res)
	endID := int(c.Message.ReplyTo.ID)
	if startID > endID {
		startID, endID = endID, startID
	}

	var toDelete []int
	for i := startID; i <= endID; i++ {
		toDelete = append(toDelete, i)
	}
	toDelete = append(toDelete, int(c.Message.ID))

	m.deleteMessages(c.Chat().ID, toDelete)

	m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())
	c.Send("Range purge complete.")
	return nil
}
