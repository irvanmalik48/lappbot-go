package moderation

import (
	"context"
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/store"
	"strings"
	"sync"

	tele "gopkg.in/telebot.v3"
)

type Module struct {
	Bot        *bot.Bot
	Store      *store.Store
	RegexCache *sync.Map
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{
		Bot:        b,
		Store:      s,
		RegexCache: &sync.Map{},
	}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle(&tele.Btn{Unique: "btn_remove_warn"}, m.onRemoveWarn)
	m.Bot.Bot.Handle("/warn", m.handleWarn)
	m.Bot.Bot.Handle("/unwarn", m.handleUnwarn)
	m.Bot.Bot.Handle("/resetwarns", m.handleResetWarns)
	m.Bot.Bot.Handle("/warns", m.handleMyWarns)

	m.Bot.Bot.Handle("/kick", m.handleKick)
	m.Bot.Bot.Handle("/skick", m.handleSilentKick)

	m.Bot.Bot.Handle("/ban", m.handleBan)
	m.Bot.Bot.Handle("/unban", m.handleUnban)
	m.Bot.Bot.Handle("/sban", m.handleSilentBan)
	m.Bot.Bot.Handle("/tban", m.handleTimedBan)
	m.Bot.Bot.Handle("/rban", m.handleRealmBan)

	m.Bot.Bot.Handle("/mute", m.handleMute)
	m.Bot.Bot.Handle("/unmute", m.handleUnmute)
	m.Bot.Bot.Handle("/smute", m.handleSilentMute)
	m.Bot.Bot.Handle("/tmute", m.handleTimedMute)
	m.Bot.Bot.Handle("/rmute", m.handleRealmMute)

	m.Bot.Bot.Handle("/purge", m.handlePurge)

	m.Bot.Bot.Handle("/pin", m.handlePin)
	m.Bot.Bot.Handle("/lock", m.handleLock)
	m.Bot.Bot.Handle("/unlock", m.handleUnlock)

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

func (m *Module) onRemoveWarn(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Respond(&tele.CallbackResponse{Text: "Only admins can use this button.", ShowAlert: true})
	}

	targetIDStr := c.Data()
	var targetID int64
	fmt.Sscanf(targetIDStr, "%d", &targetID)

	err := m.Store.RemoveLastWarn(targetID, c.Chat().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Failed to remove warn."})
	}

	count, _ := m.Store.GetWarnCount(targetID, c.Chat().ID)

	c.Edit(fmt.Sprintf("Warn removed by %s.\nTotal Warns: %d/3", mention(c.Sender()), count), tele.ModeMarkdown)
	return c.Respond(&tele.CallbackResponse{Text: "Warn removed!"})
}

func mention(u *tele.User) string {
	name := strings.ReplaceAll(u.FirstName, "]", "\\]")
	name = strings.ReplaceAll(name, "[", "\\[")
	return fmt.Sprintf("[%s](tg://user?id=%d)", name, u.ID)
}
