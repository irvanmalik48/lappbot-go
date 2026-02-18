package greeting

import (
	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/modules/utility"
	"lappbot/internal/store"
	"strings"
)

type Module struct {
	Bot    *bot.Bot
	Store  *store.Store
	Logger *logging.Module
}

func New(b *bot.Bot, s *store.Store, l *logging.Module) *Module {
	return &Module{Bot: b, Store: s, Logger: l}
}

func (m *Module) Register() {
	m.Bot.Handle("/welcome", m.handleWelcomeCommand)
	m.Bot.Handle("/goodbye", m.handleGoodbyeCommand)

	m.Bot.Handle("new_chat_members", m.OnUserJoined)
	m.Bot.Handle("left_chat_member", m.OnUserLeft)
}

func (m *Module) OnUserJoined(c *bot.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}
	if group == nil {
		err = m.Store.CreateGroup(c.Chat().ID, c.Chat().Title)
		if err != nil {
			return err
		}

		group, err = m.Store.GetGroup(c.Chat().ID)
		if err != nil {
			return err
		}
	}

	if group.GreetingEnabled && group.GreetingMessage != "" {
		for _, u := range c.Update.Message.NewChatMembers {
			userPtr := &u
			c.Send(utility.ReplacePlaceholders(group.GreetingMessage, userPtr), "Markdown")
		}
	}

	return nil
}

func (m *Module) OnUserLeft(c *bot.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}
	if group == nil {
		return nil
	}

	if group.GoodbyeEnabled && group.GoodbyeMessage != "" {
		if c.Update.Message.LeftChatMember != nil {
			c.Send(utility.ReplacePlaceholders(group.GoodbyeMessage, c.Update.Message.LeftChatMember), "Markdown")
		}
	}

	return nil
}

func (m *Module) handleWelcomeCommand(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /welcome <on|off|text> [message]")
	}

	switch args[0] {
	case "on":
		err := m.Store.SetGreetingStatus(c.Chat().ID, true)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Welcome message enabled by "+c.Sender().FirstName)
		return c.Send("Welcome message enabled.")
	case "off":
		err := m.Store.SetGreetingStatus(c.Chat().ID, false)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Welcome message disabled by "+c.Sender().FirstName)
		return c.Send("Welcome message disabled.")
	case "text":
		msg := ""
		if len(args) < 2 {
			if c.Message.ReplyTo != nil && c.Message.ReplyTo.Text != "" {
				msg = c.Message.ReplyTo.Text
			} else {
				return c.Send("Please provide a welcome message or reply to one.")
			}
		} else {
			for i := 1; i < len(args); i++ {
				msg += args[i] + " "
			}
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return c.Send("Message cannot be empty.")
		}

		err := m.Store.SetGreetingMessage(c.Chat().ID, msg)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Welcome message set by "+c.Sender().FirstName)
		return c.Send("Welcome message set.")
	default:
		return c.Send("Invalid argument. Use 'on', 'off', or 'text'.")
	}
}

func (m *Module) handleGoodbyeCommand(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /goodbye <on|off|text> [message]")
	}

	switch args[0] {
	case "on":
		err := m.Store.SetGoodbyeStatus(c.Chat().ID, true)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Goodbye message enabled by "+c.Sender().FirstName)
		return c.Send("Goodbye message enabled.")
	case "off":
		err := m.Store.SetGoodbyeStatus(c.Chat().ID, false)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Goodbye message disabled by "+c.Sender().FirstName)
		return c.Send("Goodbye message disabled.")
	case "text":
		msg := ""
		if len(args) < 2 {
			if c.Message.ReplyTo != nil && c.Message.ReplyTo.Text != "" {
				msg = c.Message.ReplyTo.Text
			} else {
				return c.Send("Please provide a goodbye message or reply to one.")
			}
		} else {
			for i := 1; i < len(args); i++ {
				msg += args[i] + " "
			}
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return c.Send("Message cannot be empty.")
		}

		err := m.Store.SetGoodbyeMessage(c.Chat().ID, msg)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		m.Logger.Log(c.Chat().ID, "settings", "Goodbye message set by "+c.Sender().FirstName)
		return c.Send("Goodbye message set.")
	default:
		return c.Send("Invalid argument. Use 'on', 'off', or 'text'.")
	}
}
