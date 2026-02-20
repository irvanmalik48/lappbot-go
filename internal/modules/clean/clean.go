package clean

import (
	"lappbot/internal/bot"
	"lappbot/internal/store"
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

var adminCommands = map[string]bool{
	"/ban": true, "/mute": true, "/kick": true, "/warn": true,
	"/tban": true, "/tmute": true, "/unban": true, "/unmute": true,
	"/pin": true, "/lock": true, "/unlock": true, "/promote": true,
	"/demote": true, "/approve": true, "/unapprove": true, "/bl": true,
	"/unbl": true, "/purge": true, "/spurge": true, "/del": true,
	"/purgefrom": true, "/purgeto": true,
	"/dwarn": true, "/swarn": true, "/unwarn": true, "/rmwarn": true,
	"/resetwarn": true, "/resetallwarns": true,
	"/newtopic": true, "/renametopic": true, "/closetopic": true,
	"/reopentopic": true, "/deletetopic": true, "/actiontopic": true,
}

var settingsCommands = map[string]bool{
	"/setlog": true, "/unsetlog": true, "/log": true, "/nolog": true,
	"/logcategories": true, "/logchannel": true,
	"/welcome": true, "/goodbye": true, "/captcha": true, "/antiraid": true,
	"/raidtime": true, "/raidactiontime": true, "/autoantiraid": true,
	"/flood": true, "/setflood": true, "/setfloodtimer": true,
	"/floodmode": true, "/clearflood": true, "/warnlimit": true,
	"/warnmode": true, "/warntime": true, "/setactiontopic": true,
	"/cleancommand": true, "/keepcommand": true, "/cleancommandtypes": true,
}

var userCommands = map[string]bool{
	"/start": true, "/help": true, "/ping": true, "/version": true,
	"/id": true, "/info": true, "/get": true,
	"/zalgo": true, "/uwuify": true, "/emojify": true, "/leetify": true,
	"/warns": true, "/warnings": true,
}

var reportCommands = map[string]bool{
	"/report": true,
}

var otherCommands = map[string]bool{
	"/notes": true, "/saved": true, "/privatenotes": true,
	"/save": true, "/clear": true, "/clearall": true,
	"/filter": true, "/stop": true, "/filters": true,
	"/connect": true, "/disconnect": true, "/reconnect": true, "/connection": true,
}

func (m *Module) Register() {
	m.Bot.Handle("/cleancommand", m.handleCleanCommand)
	m.Bot.Handle("/keepcommand", m.handleKeepCommand)
	m.Bot.Handle("/cleancommandtypes", m.handleCleanCommandTypes)
	m.Bot.Handle("unknown_command", m.handleUnknownCommand)
	m.Bot.Use(m.checkCleanCommand)
}

func (m *Module) checkCleanCommand(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
		if c.Message == nil {
			return next(c)
		}

		target, err := m.Bot.GetTargetChat(c)
		if err != nil {
			return next(c)
		}

		g, err := m.Store.GetGroup(target.ID)
		if err != nil || g == nil {
			return next(c)
		}

		var cleanTypes []string
		json.Unmarshal([]byte(g.CleanCommands), &cleanTypes)

		if len(cleanTypes) == 0 {
			return next(c)
		}

		shouldDelete := false
		cmd := strings.Split(c.Message.Text, " ")[0]
		if idx := strings.Index(cmd, "@"); idx != -1 {
			cmd = cmd[:idx]
		}

		for _, t := range cleanTypes {
			if t == "all" {
				shouldDelete = true
				break
			}
			if t == "admin" && adminCommands[cmd] {
				shouldDelete = true
				break
			}
			if t == "settings" && settingsCommands[cmd] {
				shouldDelete = true
				break
			}
			if t == "user" && userCommands[cmd] {
				shouldDelete = true
				break
			}
			if t == "reports" && reportCommands[cmd] {
				shouldDelete = true
				break
			}
			if t == "other" && otherCommands[cmd] {
				shouldDelete = true
				break
			}
		}

		if shouldDelete {
			c.Delete()
		}

		return next(c)
	}
}

func (m *Module) handleUnknownCommand(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return nil
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil || g == nil {
		return nil
	}

	var cleanTypes []string
	json.Unmarshal([]byte(g.CleanCommands), &cleanTypes)

	shouldDelete := false
	for _, t := range cleanTypes {
		if t == "all" || t == "other" {
			shouldDelete = true
			break
		}
	}

	if shouldDelete {
		c.Delete()
	}
	return nil
}

func (m *Module) handleCleanCommand(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /cleancommand <type> [type...]")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil || g == nil {
		return c.Send("Error fetching group info.")
	}

	var cleanTypes []string
	json.Unmarshal([]byte(g.CleanCommands), &cleanTypes)

	validTypes := map[string]bool{"all": true, "admin": true, "settings": true, "user": true, "automated": true, "reports": true, "other": true}
	added := []string{}

	for _, arg := range args {
		arg = strings.ToLower(arg)
		if !validTypes[arg] {
			continue
		}

		exists := false
		for _, t := range cleanTypes {
			if t == arg {
				exists = true
				break
			}
		}

		if !exists {
			cleanTypes = append(cleanTypes, arg)
			added = append(added, arg)
		}
	}

	if len(added) == 0 {
		return c.Send("No valid types provided. Available types: all, admin, settings, user, automated, reports, other")
	}

	err = m.Store.SetCleanCommands(target.ID, cleanTypes)
	if err != nil {
		return c.Send("Failed to update settings.")
	}

	return c.Send("Now cleaning: " + strings.Join(added, ", "))
}

func (m *Module) handleKeepCommand(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /keepcommand <type> [type...]")
	}

	g, err := m.Store.GetGroup(target.ID)
	if err != nil || g == nil {
		return c.Send("Error fetching group info.")
	}

	var cleanTypes []string
	json.Unmarshal([]byte(g.CleanCommands), &cleanTypes)

	removed := []string{}
	newTypes := []string{}

	for _, t := range cleanTypes {
		shouldRemove := false
		for _, arg := range args {
			if strings.EqualFold(t, arg) {
				shouldRemove = true
				removed = append(removed, t)
				break
			}
		}
		if !shouldRemove {
			newTypes = append(newTypes, t)
		}
	}

	if len(removed) == 0 {
		return c.Send("No types removed.")
	}

	err = m.Store.SetCleanCommands(target.ID, newTypes)
	if err != nil {
		return c.Send("Failed to update settings.")
	}

	return c.Send("Stopped cleaning: " + strings.Join(removed, ", "))
}

func (m *Module) handleCleanCommandTypes(c *bot.Context) error {
	return c.Send("Available command types: all, admin, settings, user, automated, reports, other")
}
