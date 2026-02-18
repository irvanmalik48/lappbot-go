package connections

import (
	"strconv"
	"strings"

	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"
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
	m.Bot.Handle("/connect", m.handleConnect)
	m.Bot.Handle("/disconnect", m.handleDisconnect)
	m.Bot.Handle("/reconnect", m.handleReconnect)
	m.Bot.Handle("/connection", m.handleConnection)
	m.Bot.Handle("conn_connect", m.onConnectCallback)
}

func (m *Module) handleConnect(c *bot.Context) error {
	args := c.Args

	if c.Chat().Type != "private" {
		if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
			return c.Send("You must be an admin to connect.")
		}
		err := m.Store.SetConnection(c.Sender().ID, c.Chat().ID)
		if err != nil {
			return c.Send("Failed to connect.")
		}
		m.Logger.Log(c.Chat().ID, "other", "User connected via command in group (ID: "+strconv.FormatInt(c.Chat().ID, 10)+")")
		return c.Send("Connected to " + c.Chat().Title + ".")
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
		m.Logger.Log(chat.ID, "other", "User connected remotely via PM (ID: "+strconv.FormatInt(c.Sender().ID, 10)+")")
		return c.Send("Connected to " + chat.Title + ".")
	}

	history, err := m.Store.GetConnectionHistory(c.Sender().ID)
	if err != nil || len(history) == 0 {
		return c.Send("No recent connections found. usage: /connect <username/id>")
	}

	markup := &bot.ReplyMarkup{}
	msg := "Select a chat to connect to:"

	var rows [][]bot.InlineKeyboardButton
	for _, item := range history {
		data := "conn_connect|" + strconv.FormatInt(item.ChatID, 10)
		btn := bot.InlineKeyboardButton{Text: item.ChatTitle, CallbackData: data}
		rows = append(rows, []bot.InlineKeyboardButton{btn})
	}
	markup.InlineKeyboard = rows

	return c.Send(msg, markup)
}

func (m *Module) onConnectCallback(c *bot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		c.Respond("Invalid data")
		return nil
	}

	chatIDStr := parts[1]
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		c.Respond("Invalid chat ID.")
		return nil
	}

	chat, err := m.Bot.ResolveChat(chatIDStr)
	if err != nil {
		c.Respond("Chat not found.")
		return nil
	}

	if !m.Bot.IsAdmin(chat, c.Sender()) {
		c.Respond("You must be an admin of that chat.")
		return nil
	}

	err = m.Store.SetConnection(c.Sender().ID, chatID)
	if err != nil {
		c.Respond("Failed to connect.")
		return nil
	}

	m.Logger.Log(chat.ID, "other", "User connected via history button (ID: "+strconv.FormatInt(c.Sender().ID, 10)+")")

	c.Delete()
	return c.Send("Connected to " + chat.Title + ".")
}

func (m *Module) handleDisconnect(c *bot.Context) error {
	err := m.Store.Disconnect(c.Sender().ID)
	if err != nil {
		return c.Send("Failed to disconnect.")
	}
	return c.Send("Disconnected.")
}

func (m *Module) handleReconnect(c *bot.Context) error {
	history, err := m.Store.GetConnectionHistory(c.Sender().ID)
	if err != nil || len(history) == 0 {
		return c.Send("No recent connections.")
	}

	last := history[0]
	chatStr := strconv.FormatInt(last.ChatID, 10)
	chat, err := m.Bot.ResolveChat(chatStr)
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
	m.Logger.Log(chat.ID, "other", "User reconnected via command (ID: "+strconv.FormatInt(c.Sender().ID, 10)+")")
	return c.Send("Reconnected to " + chat.Title + ".")
}

func (m *Module) handleConnection(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error checking connection.")
	}

	if target.ID == c.Chat().ID {
		return c.Send("Not connected to any remote chat.")
	}

	return c.Send("Currently connected to: " + target.Title + " (ID: " + strconv.FormatInt(target.ID, 10) + ")")
}
