package filters

import (
	"fmt"
	"strings"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v3"
)

type FiltersModule struct {
	Bot   *bot.Bot
	Store *store.Store
}

func NewFilters(b *bot.Bot, s *store.Store) *FiltersModule {
	return &FiltersModule{Bot: b, Store: s}
}

func (m *FiltersModule) Register() {
	m.Bot.Bot.Handle("/filter", m.handleFilter)
	m.Bot.Bot.Handle("/stop", m.handleStop)
	m.Bot.Bot.Handle("/filters", m.handleFilters)

	m.Bot.Bot.Handle(tele.OnText, m.handleText)
}

func (m *FiltersModule) handleFilter(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return c.Send("Usage: /filter <trigger> <response>")
	}

	trigger := strings.ToLower(args[0])
	response := strings.Join(args[1:], " ")

	err := m.Store.AddFilter(c.Chat().ID, trigger, response)
	if err != nil {
		return c.Send("Failed to save filter: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Filter saved!\nTrigger: %s\nResponse: %s", trigger, response))
}

func (m *FiltersModule) handleStop(c tele.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return c.Send("Usage: /stop <trigger>")
	}

	trigger := strings.ToLower(args[0])

	err := m.Store.DeleteFilter(c.Chat().ID, trigger)
	if err != nil {
		return c.Send("Failed to delete filter: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Filter '%s' deleted.", trigger))
}

func (m *FiltersModule) handleFilters(c tele.Context) error {
	filters, err := m.Store.GetFilters(c.Chat().ID)
	if err != nil {
		return c.Send("Failed to fetch filters: " + err.Error())
	}

	if len(filters) == 0 {
		return c.Send("No filters active in this chat.")
	}

	msg := "<b>Active Filters:</b>\n"
	for _, f := range filters {
		msg += fmt.Sprintf("• <code>%s</code>\n", f.Trigger)
	}

	return c.Send(msg, tele.ModeHTML)
}

func (m *FiltersModule) handleText(c tele.Context) error {
	text := c.Text()
	if strings.HasPrefix(text, "/") {
		return nil
	}

	filters, err := m.Store.GetFilters(c.Chat().ID)
	if err != nil {
		return nil
	}

	lowerText := strings.ToLower(text)

	for _, f := range filters {
		if strings.Contains(lowerText, strings.ToLower(f.Trigger)) {
			return c.Send(f.Response)
		}
	}

	return nil
}
