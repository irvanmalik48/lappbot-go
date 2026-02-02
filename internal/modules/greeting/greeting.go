package greeting

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/store"
	"strings"

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
	m.Bot.Bot.Handle("/welcome", m.handleWelcomeCommand)
	m.Bot.Bot.Handle("/goodbye", m.handleGoodbyeCommand)
}

func (m *Module) OnUserJoined(c tele.Context) error {
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
		return c.Send(m.replacePlaceholders(group.GreetingMessage, c.Sender()), tele.ModeMarkdown)
	}

	return nil
}

func (m *Module) OnUserLeft(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}
	if group == nil {
		return nil
	}

	if group.GoodbyeEnabled && group.GoodbyeMessage != "" {
		return c.Send(m.replacePlaceholders(group.GoodbyeMessage, c.Sender()), tele.ModeMarkdown)
	}

	return nil
}

func (m *Module) replacePlaceholders(msg string, user *tele.User) string {
	msg = strings.ReplaceAll(msg, "{firstname}", fmt.Sprintf("[%s](tg://user?id=%d)", user.FirstName, user.ID))
	msg = strings.ReplaceAll(msg, "{username}", user.Username)
	msg = strings.ReplaceAll(msg, "{userid}", fmt.Sprintf("%d", user.ID))
	return msg
}

func (m *Module) handleWelcomeCommand(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

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

func (m *Module) handleGoodbyeCommand(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /goodbye <on|off> [message]")
	}

	switch args[0] {
	case "on":
		if len(args) < 2 {
			return c.Send("Please provide a goodbye message.")
		}
		msg := ""
		for i := 1; i < len(args); i++ {
			msg += args[i] + " "
		}
		err := m.Store.UpdateGroupGoodbye(c.Chat().ID, true, msg)
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		return c.Send("Goodbye message enabled.")
	case "off":
		err := m.Store.UpdateGroupGoodbye(c.Chat().ID, false, "")
		if err != nil {
			return c.Send("Error updating setting: " + err.Error())
		}
		return c.Send("Goodbye message disabled.")
	default:
		return c.Send("Invalid argument. Use 'on' or 'off'.")
	}
}
