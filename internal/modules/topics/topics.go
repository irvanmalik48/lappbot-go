package topics

import (
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"lappbot/internal/modules/logging"
	"strconv"
	"strings"
)

type Module struct {
	Bot    *bot.Bot
	Cfg    *config.Config
	Logger *logging.Module
}

func New(b *bot.Bot, cfg *config.Config, l *logging.Module) *Module {
	return &Module{Bot: b, Cfg: cfg, Logger: l}
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
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, targetChat, c.Sender(), "can_manage_topics") {
		return nil
	}

	group, err := m.Bot.Store.GetGroup(targetChat.ID)
	if err != nil {
		return c.Send("Error fetching group data.")
	}
	if group == nil {
		return c.Send("Group not found.")
	}

	if group.ActionTopicID == nil {
		return c.Send("No action topic set.")
	}

	return c.Send("Action topic ID: `"+strconv.FormatInt(*group.ActionTopicID, 10)+"`", "Markdown")
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

	m.Logger.Log(c.Chat().ID, "other", "Action topic set to ID "+strconv.FormatInt(int64(topicID), 10)+" by "+c.Sender().FirstName)

	return c.Send("Action topic set to current topic (ID: `"+strconv.FormatInt(topicID, 10)+"`).", "Markdown")
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

	m.Logger.Log(c.Chat().ID, "other", "New topic created: "+topicName+" by "+c.Sender().FirstName)

	return c.Send("Topic created: " + topicName)
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

	m.Logger.Log(c.Chat().ID, "other", "Topic renamed to "+topicName+" by "+c.Sender().FirstName)

	return c.Send("Topic renamed to: " + topicName)
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

	m.Logger.Log(c.Chat().ID, "other", "Topic closed by "+c.Sender().FirstName)

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

	m.Logger.Log(c.Chat().ID, "other", "Topic reopened by "+c.Sender().FirstName)

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

	m.Logger.Log(c.Chat().ID, "other", "Topic deleted by "+c.Sender().FirstName)

	return nil
}
