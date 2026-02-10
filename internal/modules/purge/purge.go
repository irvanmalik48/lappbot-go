package purge

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	"github.com/valkey-io/valkey-go"
	tele "gopkg.in/telebot.v4"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle("/purge", m.handlePurge)
	m.Bot.Bot.Handle("/spurge", m.handleSPurge)
	m.Bot.Bot.Handle("/del", m.handleDel)
	m.Bot.Bot.Handle("/purgefrom", m.handlePurgeFrom)
	m.Bot.Bot.Handle("/purgeto", m.handlePurgeTo)
}

func (m *Module) handlePurge(c tele.Context) error {
	if !c.Message().IsReply() {
		return c.Send("Reply to a message to purge from.")
	}
	args := c.Args()
	limit := 0
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}
	startID := c.Message().ReplyTo.ID
	endID := c.Message().ID
	var toDelete []tele.Editable
	if limit > 0 {
		for i := 1; i <= limit; i++ {
			toDelete = append(toDelete, &tele.Message{ID: startID + i, Chat: c.Chat()})
		}
	} else {
		for i := startID; i < endID; i++ {
			toDelete = append(toDelete, &tele.Message{ID: i, Chat: c.Chat()})
		}
	}
	toDelete = append(toDelete, &tele.Message{ID: endID, Chat: c.Chat()})
	c.Bot().DeleteMany(toDelete)

	msg, err := c.Bot().Send(c.Chat(), "Purge complete.")
	if err == nil {
		go func() {
			time.Sleep(3 * time.Second)
			c.Bot().Delete(msg)
		}()
	}
	return nil
}

func (m *Module) handleSPurge(c tele.Context) error {
	if !c.Message().IsReply() {
		return nil
	}
	args := c.Args()
	limit := 0
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}
	startID := c.Message().ReplyTo.ID
	endID := c.Message().ID
	var toDelete []tele.Editable
	if limit > 0 {
		for i := 1; i <= limit; i++ {
			toDelete = append(toDelete, &tele.Message{ID: startID + i, Chat: c.Chat()})
		}
	} else {
		for i := startID; i < endID; i++ {
			toDelete = append(toDelete, &tele.Message{ID: i, Chat: c.Chat()})
		}
	}
	toDelete = append(toDelete, &tele.Message{ID: endID, Chat: c.Chat()})
	c.Bot().DeleteMany(toDelete)
	return nil
}

func (m *Module) handleDel(c tele.Context) error {
	if !c.Message().IsReply() {
		return nil
	}
	c.Bot().Delete(c.Message().ReplyTo)
	c.Delete()
	return nil
}

func (m *Module) handlePurgeFrom(c tele.Context) error {
	if !c.Message().IsReply() {
		return c.Send("Reply to a message to mark as purge start.")
	}
	key := fmt.Sprintf("purgefrom:%d", c.Chat().ID)
	err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(key).Value(strconv.Itoa(c.Message().ReplyTo.ID)).Ex(time.Minute*5).Build()).Error()
	if err != nil {
		return c.Send("Failed to set purge start point.")
	}
	c.Delete()
	msg, err := c.Bot().Send(c.Chat(), "Purge start marked. Reply to another message with /purgeto to purge range.")
	if err == nil {
		go func() {
			time.Sleep(5 * time.Second)
			c.Bot().Delete(msg)
		}()
	}
	return nil
}

func (m *Module) handlePurgeTo(c tele.Context) error {
	if !c.Message().IsReply() {
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
	endID := c.Message().ReplyTo.ID
	if startID > endID {
		startID, endID = endID, startID
	}
	var toDelete []tele.Editable
	for i := startID; i <= endID; i++ {
		toDelete = append(toDelete, &tele.Message{ID: i, Chat: c.Chat()})
	}
	toDelete = append(toDelete, &tele.Message{ID: c.Message().ID, Chat: c.Chat()})
	c.Bot().DeleteMany(toDelete)

	m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())
	msg, err := c.Bot().Send(c.Chat(), "Range purge complete.")
	if err == nil {
		go func() {
			time.Sleep(3 * time.Second)
			c.Bot().Delete(msg)
		}()
	}
	return nil
}
