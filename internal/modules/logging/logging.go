package logging

import (
	"lappbot/internal/bot"
	"lappbot/internal/store"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Handle("/logchannel", m.handleLogChannel)
	m.Bot.Handle("/setlog", m.handleSetLog)
	m.Bot.Handle("/unsetlog", m.handleUnsetLog)
	m.Bot.Handle("/log", m.handleLogCategory)
	m.Bot.Handle("/nolog", m.handleNoLogCategory)
	m.Bot.Handle("/logcategories", m.handleLogCategories)
	m.Bot.Use(m.checkForwardedLogChannel)
}

func (m *Module) checkForwardedLogChannel(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
		if c.Message.ForwardFromChat != nil && c.Message.ForwardFromChat.Type == "channel" {
			if strings.HasPrefix(c.Message.Text, "/setlog") {
				target, err := m.Bot.GetTargetChat(c)
				if err != nil {
					return next(c)
				}

				if !m.Bot.IsAdmin(target, c.Sender()) {
					return next(c)
				}

				channel := c.Message.ForwardFromChat

				err = m.Store.SetLogChannel(target.ID, channel.ID)
				if err != nil {
					return c.Send("Failed to set log channel.")
				}

				m.Bot.Raw("sendMessage", map[string]any{
					"chat_id": channel.ID,
					"text":    "Log channel set for group " + target.Title,
				})

				return c.Send("Log channel set to " + channel.Title + " (ID: " + strconv.FormatInt(channel.ID, 10) + ")")
			}
		}
		return next(c)
	}
}

func (m *Module) handleLogChannel(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	if g.LogChannelID == 0 {
		return c.Send("No log channel set.")
	}

	return c.Send("Log channel ID: " + strconv.FormatInt(g.LogChannelID, 10))
}

func (m *Module) handleSetLog(c *bot.Context) error {
	if c.Message.ForwardFromChat == nil {
		return c.Send("To set a log channel:\n1. Add me to the channel as admin.\n2. Send /setlog in the channel.\n3. Forward that message here.")
	}
	return nil
}

func (m *Module) handleUnsetLog(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}

	err = m.Store.SetLogChannel(target.ID, 0)
	if err != nil {
		return c.Send("Failed to unset log channel.")
	}
	return c.Send("Log channel unset.")
}

func (m *Module) handleLogCategory(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /log <category>")
	}
	category := strings.ToLower(args[0])

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	var categories []string
	json.Unmarshal([]byte(g.LogCategories), &categories)

	for _, cat := range categories {
		if cat == category {
			return c.Send("Category '" + category + "' is already logged.")
		}
	}

	categories = append(categories, category)
	err = m.Store.SetLogCategories(target.ID, categories)
	if err != nil {
		return c.Send("Failed to update log categories.")
	}

	return c.Send("Category '" + category + "' enabled.")
}

func (m *Module) handleNoLogCategory(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /nolog <category>")
	}
	category := strings.ToLower(args[0])

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	var categories []string
	json.Unmarshal([]byte(g.LogCategories), &categories)

	newCategories := make([]string, 0)
	found := false
	for _, cat := range categories {
		if cat == category {
			found = true
			continue
		}
		newCategories = append(newCategories, cat)
	}

	if !found {
		return c.Send("Category '" + category + "' was not logged.")
	}

	err = m.Store.SetLogCategories(target.ID, newCategories)
	if err != nil {
		return c.Send("Failed to update log categories.")
	}

	return c.Send("Category '" + category + "' disabled.")
}

func (m *Module) handleLogCategories(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	var categories []string
	json.Unmarshal([]byte(g.LogCategories), &categories)

	if len(categories) == 0 {
		return c.Send("No categories logged.")
	}

	return c.Send("Logged categories: " + strings.Join(categories, ", "))
}

func (m *Module) Log(chatID int64, category, message string) {
	group, err := m.Store.GetGroup(chatID)
	if err != nil || group == nil {
		return
	}

	if group.LogChannelID == 0 {
		return
	}

	var categories []string
	if err := json.Unmarshal([]byte(group.LogCategories), &categories); err != nil {
		return
	}

	found := false
	for _, c := range categories {
		if c == category {
			found = true
			break
		}
	}

	if !found {
		return
	}

	m.Bot.Raw("sendMessage", map[string]any{
		"chat_id": group.LogChannelID,
		"text":    "[" + strings.ToUpper(category) + "] " + message,
	})
}
