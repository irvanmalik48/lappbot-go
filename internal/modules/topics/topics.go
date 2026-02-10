package topics

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"strings"

	tele "gopkg.in/telebot.v4"
)

type Module struct {
	Bot *bot.Bot
	Cfg *config.Config
}

func New(b *bot.Bot, cfg *config.Config) *Module {
	return &Module{Bot: b, Cfg: cfg}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle("/actiontopic", m.handleActionTopic)
	m.Bot.Bot.Handle("/setactiontopic", m.handleSetActionTopic)
	m.Bot.Bot.Handle("/newtopic", m.handleNewTopic)
	m.Bot.Bot.Handle("/renametopic", m.handleRenameTopic)
	m.Bot.Bot.Handle("/closetopic", m.handleCloseTopic)
	m.Bot.Bot.Handle("/reopentopic", m.handleReopenTopic)
	m.Bot.Bot.Handle("/deletetopic", m.handleDeleteTopic)
}

func (m *Module) handleActionTopic(c tele.Context) error {
	group, err := m.Bot.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return c.Send("Error fetching group data.")
	}
	if group == nil {
		return c.Send("Group not found.")
	}

	if group.ActionTopicID == nil {
		return c.Send("No action topic set.")
	}

	return c.Send(fmt.Sprintf("Action topic ID: `%d`", *group.ActionTopicID), tele.ModeMarkdown)
}

func (m *Module) handleSetActionTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := int64(c.Message().ThreadID)
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	err := m.Bot.Store.SetActionTopic(c.Chat().ID, topicID)
	if err != nil {
		return c.Send("Error setting action topic.")
	}

	return c.Send(fmt.Sprintf("Action topic set to current topic (ID: `%d`).", topicID), tele.ModeMarkdown)
}

func (m *Module) handleNewTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	name := c.Args()
	if len(name) == 0 {
		return c.Send("Usage: /newtopic <name>")
	}

	topicName := strings.Join(name, " ")
	topic, err := m.Bot.Bot.CreateTopic(c.Chat(), &tele.Topic{Name: topicName})
	if err != nil {
		return c.Send("Error creating topic: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Topic created: %s", topic.Name))
}

func (m *Module) handleRenameTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	name := c.Args()
	if len(name) == 0 {
		return c.Send("Usage: /renametopic <name>")
	}

	topicID := c.Message().ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	topicName := strings.Join(name, " ")
	err := m.Bot.Bot.EditTopic(c.Chat(), &tele.Topic{ThreadID: topicID, Name: topicName})
	if err != nil {
		return c.Send("Error renaming topic: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Topic renamed to: %s", topicName))
}

func (m *Module) handleCloseTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message().ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	err := m.Bot.Bot.CloseTopic(c.Chat(), &tele.Topic{ThreadID: topicID})
	if err != nil {
		return c.Send("Error closing topic: " + err.Error())
	}

	return c.Send("Topic closed.")
}

func (m *Module) handleReopenTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message().ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	err := m.Bot.Bot.ReopenTopic(c.Chat(), &tele.Topic{ThreadID: topicID})
	if err != nil {
		return c.Send("Error reopening topic: " + err.Error())
	}

	return c.Send("Topic reopened.")
}

func (m *Module) handleDeleteTopic(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message().ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	err := m.Bot.Bot.DeleteTopic(c.Chat(), &tele.Topic{ThreadID: topicID})
	if err != nil {
		return c.Send("Error deleting topic: " + err.Error())
	}

	return nil
}
