package greeting

import (
	"lappbot/internal/bot"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v3"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle(tele.OnUserJoined, m.onUserJoined)
	m.Bot.Bot.Handle("/welcome", m.handleWelcomeCommand)
}

func (m *Module) onUserJoined(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}
	if group == nil {
		return nil
	}

	if group.GreetingEnabled && group.GreetingMessage != "" {
		return c.Send(group.GreetingMessage)
	}

	return nil
}

func (m *Module) handleWelcomeCommand(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /welcome <on|off> [message]")
	}

	switch args[0] {
	case "on":
		if len(args) < 2 {
			return c.Send("Please provide a welcome message.")
		}
		msg := ""
		for i := 1; i < len(args); i++ {
			msg += args[i] + " "
		}
		err := m.Store.UpdateGroupGreeting(c.Chat().ID, true, msg)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		return c.Send("Welcome message enabled.")
	case "off":
		err := m.Store.UpdateGroupGreeting(c.Chat().ID, false, "")
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		return c.Send("Welcome message disabled.")
	default:
		return c.Send("Invalid argument. Use 'on' or 'off'.")
	}
}
