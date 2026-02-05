package connections

import (
	"fmt"
	"strconv"

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
	m.Bot.Bot.Handle("/connect", m.handleConnect)
	m.Bot.Bot.Handle("/disconnect", m.handleDisconnect)
	m.Bot.Bot.Handle("/reconnect", m.handleReconnect)
	m.Bot.Bot.Handle("/connection", m.handleConnection)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "conn_connect"}, m.onConnectCallback)
}

func (m *Module) handleConnect(c tele.Context) error {
	args := c.Args()

	if c.Chat().Type != tele.ChatPrivate {
		if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
			return c.Send("You must be an admin to connect.")
		}
		err := m.Store.SetConnection(c.Sender().ID, c.Chat().ID)
		if err != nil {
			return c.Send("Failed to connect.")
		}
		return c.Send(fmt.Sprintf("Connected to %s.", c.Chat().Title))
	}

	if len(args) > 0 {
		identity := args[0]
		chat, err := m.Bot.ResolveChat(identity)
		if err != nil {
			return c.Send("Chat not found. Make sure the bot is in the chat or use the correct ID.")
		}

		if !m.Bot.IsAdmin(chat, c.Sender()) {
			return c.Send("You must be an admin of that chat to connect.")
		}

		err = m.Store.SetConnection(c.Sender().ID, chat.ID)
		if err != nil {
			return c.Send("Failed to connect.")
		}
		_ = m.Store.AddConnectionHistory(c.Sender().ID, chat.ID, chat.Title)
		return c.Send(fmt.Sprintf("Connected to %s.", chat.Title))
	}

	history, err := m.Store.GetConnectionHistory(c.Sender().ID)
	if err != nil || len(history) == 0 {
		return c.Send("No recent connections found. usage: /connect <username/id>")
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, item := range history {
		rows = append(rows, markup.Row(markup.Data(item.ChatTitle, "conn_connect", fmt.Sprintf("%d", item.ChatID))))
	}
	markup.Inline(rows...)

	return c.Send("Select a chat to connect to:", markup)
}

func (m *Module) onConnectCallback(c tele.Context) error {
	chatIDStr := c.Data()
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Invalid chat ID."})
	}

	chat, err := m.Bot.Bot.ChatByID(chatID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Chat not found."})
	}

	if !m.Bot.IsAdmin(chat, c.Sender()) {
		return c.Respond(&tele.CallbackResponse{Text: "You must be an admin of that chat."})
	}

	err = m.Store.SetConnection(c.Sender().ID, chatID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Failed to connect."})
	}

	_ = m.Bot.Bot.Delete(c.Message())
	return c.Send(fmt.Sprintf("Connected to %s.", chat.Title))
}

func (m *Module) handleDisconnect(c tele.Context) error {
	err := m.Store.Disconnect(c.Sender().ID)
	if err != nil {
		return c.Send("Failed to disconnect.")
	}
	return c.Send("Disconnected.")
}

func (m *Module) handleReconnect(c tele.Context) error {
	history, err := m.Store.GetConnectionHistory(c.Sender().ID)
	if err != nil || len(history) == 0 {
		return c.Send("No recent connections.")
	}

	last := history[0]
	chat, err := m.Bot.Bot.ChatByID(last.ChatID)
	if err != nil {
		return c.Send("Previous chat not found.")
	}

	if !m.Bot.IsAdmin(chat, c.Sender()) {
		return c.Send("You must be an admin of that chat.")
	}

	err = m.Store.SetConnection(c.Sender().ID, last.ChatID)
	if err != nil {
		return c.Send("Failed to connect.")
	}
	return c.Send(fmt.Sprintf("Reconnected to %s.", chat.Title))
}

func (m *Module) handleConnection(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error checking connection.")
	}

	if target.ID == c.Chat().ID {
		return c.Send("Not connected to any remote chat.")
	}

	return c.Send(fmt.Sprintf("Currently connected to: %s (ID: %d)", target.Title, target.ID))
}
