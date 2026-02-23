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
	m.Bot.Handle("/loggroup", m.handleLogGroup)
	m.Bot.Handle("/setlog", m.handleSetLog)
	m.Bot.Handle("/unsetlog", m.handleUnsetLog)
	m.Bot.Handle("/log", m.handleLogCategory)
	m.Bot.Handle("/nolog", m.handleNoLogCategory)
	m.Bot.Handle("/logcategories", m.handleLogCategories)
}

func (m *Module) handleLogGroup(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	if g.LogChannelID == 0 {
		return c.Send("No log group set.")
	}

	return c.Send("Log group ID: " + strconv.FormatInt(g.LogChannelID, 10))
}

func (m *Module) handleSetLog(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, target, c.Sender()) {
		return nil
	}

	if len(c.Args) == 0 {
		return c.Send("Usage: /setlog <group_id>")
	}

	groupID, err := strconv.ParseInt(c.Args[0], 10, 64)
	if err != nil {
		return c.Send("Invalid group ID. Must be a number.")
	}

	err = m.Store.SetLogChannel(target.ID, groupID)
	if err != nil {
		return c.Send("Failed to set log group.")
	}

	m.Bot.Raw("sendMessage", map[string]any{
		"chat_id": groupID,
		"text":    "Log group set for group " + target.Title,
	})

	return c.Send("Log group set to ID: " + strconv.FormatInt(groupID, 10))
}

func (m *Module) handleUnsetLog(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender()) {
		return nil
	}

	err = m.Store.SetLogChannel(target.ID, 0)
	if err != nil {
		return c.Send("Failed to unset log group.")
	}
	return c.Send("Log group unset.")
}

func (m *Module) handleLogCategory(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /log <category> [category...]")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	var categories []string
	json.Unmarshal([]byte(g.LogCategories), &categories)

	validCategories := map[string]bool{
		"settings": true, "admin": true, "user": true,
		"automated": true, "reports": true, "other": true,
	}

	added := []string{}
	for _, arg := range args {
		arg = strings.ToLower(arg)
		if arg == "all" {
			for cat := range validCategories {
				exists := false
				for _, existing := range categories {
					if existing == cat {
						exists = true
						break
					}
				}
				if !exists {
					categories = append(categories, cat)
					added = append(added, cat)
				}
			}
			break
		}

		if !validCategories[arg] {
			continue
		}

		exists := false
		for _, cat := range categories {
			if cat == arg {
				exists = true
				break
			}
		}

		if !exists {
			categories = append(categories, arg)
			added = append(added, arg)
		}
	}

	if len(added) == 0 {
		return c.Send("No new categories enabled. valid categories: " + strings.Join(getMapKeys(validCategories), ", "))
	}

	err = m.Store.SetLogCategories(target.ID, categories)
	if err != nil {
		return c.Send("Failed to update log categories.")
	}

	return c.Send("Enabled categories: " + strings.Join(added, ", "))
}

func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (m *Module) handleNoLogCategory(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /nolog <category> [category...]")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil {
		return c.Send("Error fetching group info.")
	}

	var categories []string
	json.Unmarshal([]byte(g.LogCategories), &categories)

	validCategories := map[string]bool{
		"settings": true, "admin": true, "user": true,
		"automated": true, "reports": true, "other": true,
	}

	removed := []string{}

	if strings.ToLower(args[0]) == "all" {
		categories = []string{}
		removed = append(removed, "all")
	} else {
		toRemove := make(map[string]bool)
		for _, arg := range args {
			arg = strings.ToLower(arg)
			if validCategories[arg] {
				toRemove[arg] = true
				removed = append(removed, arg)
			}
		}

		newCategories := make([]string, 0)
		for _, cat := range categories {
			if !toRemove[cat] {
				newCategories = append(newCategories, cat)
			}
		}
		categories = newCategories
	}

	if len(removed) == 0 {
		return c.Send("No valid categories to disable. valid categories: " + strings.Join(getMapKeys(validCategories), ", "))
	}

	err = m.Store.SetLogCategories(target.ID, categories)
	if err != nil {
		return c.Send("Failed to update log categories.")
	}

	return c.Send("Disabled categories: " + strings.Join(removed, ", "))
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

	// If no log channel is set, return early
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
