package moderation

import (
	"context"
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/store"
	"time"

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
	m.Bot.Bot.Handle("/warn", m.handleWarn)
	m.Bot.Bot.Handle("/resetwarns", m.handleResetWarns)
	m.Bot.Bot.Handle("/warns", m.handleMyWarns)

	m.Bot.Bot.Handle("/kick", m.handleKick)
	m.Bot.Bot.Handle("/skick", m.handleSilentKick)

	m.Bot.Bot.Handle("/ban", m.handleBan)
	m.Bot.Bot.Handle("/sban", m.handleSilentBan)
	m.Bot.Bot.Handle("/tban", m.handleTimedBan)
	m.Bot.Bot.Handle("/rban", m.handleRealmBan)

	m.Bot.Bot.Handle("/mute", m.handleMute)
	m.Bot.Bot.Handle("/smute", m.handleSilentMute)
	m.Bot.Bot.Handle("/tmute", m.handleTimedMute)
	m.Bot.Bot.Handle("/rmute", m.handleRealmMute)

	m.Bot.Bot.Handle("/purge", m.handlePurge)

	m.Bot.Bot.Handle("/pin", m.handlePin)

	m.Bot.Bot.Handle("/bl", m.handleBlacklistAdd)
	m.Bot.Bot.Handle("/unbl", m.handleBlacklistRemove)
	m.Bot.Bot.Handle("/blacklist", m.handleBlacklistList)

	m.Bot.Bot.Handle("/approve", m.handleApprove)
	m.Bot.Bot.Handle("/unapprove", m.handleUnapprove)
	m.Bot.Bot.Handle("/promote", m.handlePromote)
	m.Bot.Bot.Handle("/demote", m.handleDemote)

	m.Bot.Bot.Handle("/refreshcache", m.handleRefreshCache)

	m.Bot.Bot.Use(m.CheckBlacklist)
}

func (m *Module) IsAdmin(chat *tele.Chat, user *tele.User) bool {
	key := fmt.Sprintf("admin:%d:%d", chat.ID, user.ID)
	val, err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Get().Key(key).Build()).ToString()
	if err == nil {
		return val == "1"
	}

	member, err := m.Bot.Bot.ChatMemberOf(chat, user)
	if err != nil {
		return false
	}

	isAdmin := member.Role == tele.Administrator || member.Role == tele.Creator

	v := "0"
	if isAdmin {
		v = "1"
	}

	m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(key).Value(v).Ex(2*time.Minute).Build())

	return isAdmin
}

func (m *Module) handleRefreshCache(c tele.Context) error {
	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Flushdb().Build()).Error()
	if err != nil {
		return c.Send("Failed to refresh cache: " + err.Error())
	}

	return c.Send("Cache refreshed successfully.")
}

func mention(u *tele.User) string {
	return fmt.Sprintf("[%s](tg://user?id=%d)", u.FirstName, u.ID)
}
