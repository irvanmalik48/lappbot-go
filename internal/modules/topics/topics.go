package topics

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"strings"
)

type Module struct {
	Bot *bot.Bot
	Cfg *config.Config
}

func New(b *bot.Bot, cfg *config.Config) *Module {
	return &Module{Bot: b, Cfg: cfg}
}

func (m *Module) Register() {
	m.Bot.Handle("/actiontopic", m.handleActionTopic)
	m.Bot.Handle("/setactiontopic", m.handleSetActionTopic)
	m.Bot.Handle("/newtopic", m.handleNewTopic)
	m.Bot.Handle("/renametopic", m.handleRenameTopic)
	m.Bot.Handle("/closetopic", m.handleCloseTopic)
	m.Bot.Handle("/reopentopic", m.handleReopenTopic)
	m.Bot.Handle("/deletetopic", m.handleDeleteTopic)
}

func (m *Module) handleActionTopic(c *bot.Context) error {
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

	return c.Send(fmt.Sprintf("Action topic ID: `%d`", *group.ActionTopicID), "Markdown")
}

func (m *Module) handleSetActionTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message.ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	err := m.Bot.Store.SetActionTopic(c.Chat().ID, topicID)
	if err != nil {
		return c.Send("Error setting action topic.")
	}

	return c.Send(fmt.Sprintf("Action topic set to current topic (ID: `%d`).", topicID), "Markdown")
}

func (m *Module) handleNewTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	name := c.Args
	if len(name) == 0 {
		return c.Send("Usage: /newtopic <name>")
	}

	topicName := strings.Join(name, " ")

	req := map[string]any{
		"chat_id": c.Chat().ID,
		"name":    topicName,
	}

	err := m.Bot.Raw("createForumTopic", req)
	if err != nil {
		return c.Send("Error creating topic: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Topic created: %s", topicName))
}

func (m *Module) handleRenameTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	name := c.Args
	if len(name) == 0 {
		return c.Send("Usage: /renametopic <name>")
	}

	topicID := c.Message.ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	topicName := strings.Join(name, " ")
	req := map[string]any{
		"chat_id":           c.Chat().ID,
		"message_thread_id": topicID,
		"name":              topicName,
	}

	err := m.Bot.Raw("editForumTopic", req)
	if err != nil {
		return c.Send("Error renaming topic: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Topic renamed to: %s", topicName))
}

func (m *Module) handleCloseTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message.ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	req := map[string]any{
		"chat_id":           c.Chat().ID,
		"message_thread_id": topicID,
	}

	err := m.Bot.Raw("closeForumTopic", req)
	if err != nil {
		return c.Send("Error closing topic: " + err.Error())
	}

	return c.Send("Topic closed.")
}

func (m *Module) handleReopenTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message.ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	req := map[string]any{
		"chat_id":           c.Chat().ID,
		"message_thread_id": topicID,
	}

	err := m.Bot.Raw("reopenForumTopic", req)
	if err != nil {
		return c.Send("Error reopening topic: " + err.Error())
	}

	return c.Send("Topic reopened.")
}

func (m *Module) handleDeleteTopic(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	topicID := c.Message.ThreadID
	if topicID == 0 {
		return c.Send("This command must be used in a topic.")
	}

	req := map[string]any{
		"chat_id":           c.Chat().ID,
		"message_thread_id": topicID,
	}

	err := m.Bot.Raw("deleteForumTopic", req)
	if err != nil {
		return c.Send("Error deleting topic: " + err.Error())
	}

	return nil
}
