package moderation

import (
	"context"
	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type CachedRegex struct {
	Re   *regexp.Regexp
	Item store.BlacklistItem
}

type BlacklistCache struct {
	sync.RWMutex
	Regexes       map[int64][]CachedRegex
	StickerSets   map[int64]map[string]store.BlacklistItem
	Emojis        map[int64]map[string]store.BlacklistItem
	ApprovedUsers map[int64]map[int64]struct{}
}

type Module struct {
	Bot            *bot.Bot
	Store          *store.Store
	BlacklistCache *BlacklistCache
	Logger         *logging.Module
}

func New(b *bot.Bot, s *store.Store, logger *logging.Module) *Module {
	return &Module{
		Bot:   b,
		Store: s,
		BlacklistCache: &BlacklistCache{
			Regexes:       make(map[int64][]CachedRegex),
			StickerSets:   make(map[int64]map[string]store.BlacklistItem),
			Emojis:        make(map[int64]map[string]store.BlacklistItem),
			ApprovedUsers: make(map[int64]map[int64]struct{}),
		},
		Logger: logger,
	}
}

func (m *Module) Register() {
	m.Bot.Handle("/warn", m.handleWarn)
	m.Bot.Handle("/dwarn", m.handleDWarn)
	m.Bot.Handle("/swarn", m.handleSWarn)
	m.Bot.Handle("/unwarn", m.handleRmWarn)
	m.Bot.Handle("/rmwarn", m.handleRmWarn)
	m.Bot.Handle("/resetwarn", m.handleResetWarns)
	m.Bot.Handle("/resetallwarns", m.handleResetAllWarns)
	m.Bot.Handle("/warns", m.handleMyWarns)
	m.Bot.Handle("/warnings", m.handleWarnings)
	m.Bot.Handle("/warnlimit", m.handleWarnLimit)
	m.Bot.Handle("/warnmode", m.handleWarnMode)
	m.Bot.Handle("/warntime", m.handleWarnTime)
	m.Bot.Handle("btn_remove_warn", m.onRemoveWarnBtn)

	m.Bot.Handle("/kick", m.handleKick)
	m.Bot.Handle("/skick", m.handleSilentKick)

	m.Bot.Handle("/ban", m.handleBan)
	m.Bot.Handle("/unban", m.handleUnban)
	m.Bot.Handle("/sban", m.handleSilentBan)
	m.Bot.Handle("/tban", m.handleTimedBan)
	m.Bot.Handle("/rban", m.handleRealmBan)

	m.Bot.Handle("/mute", m.handleMute)
	m.Bot.Handle("/unmute", m.handleUnmute)
	m.Bot.Handle("/smute", m.handleSilentMute)
	m.Bot.Handle("/tmute", m.handleTimedMute)
	m.Bot.Handle("/rmute", m.handleRealmMute)

	m.Bot.Handle("/pin", m.handlePin)
	m.Bot.Handle("/lock", m.handleLock)
	m.Bot.Handle("/unlock", m.handleUnlock)

	m.Bot.Handle("/bl", m.handleBlacklistAdd)
	m.Bot.Handle("/unbl", m.handleBlacklistRemove)
	m.Bot.Handle("/blacklist", m.handleBlacklistList)

	m.Bot.Handle("/approve", m.handleApprove)
	m.Bot.Handle("/unapprove", m.handleUnapprove)
	m.Bot.Handle("/promote", m.handlePromote)
	m.Bot.Handle("/demote", m.handleDemote)

	m.Bot.Handle("/refreshcache", m.handleRefreshCache)

	m.Bot.Use(m.CheckBlacklist)
}

func (m *Module) handleRefreshCache(c *bot.Context) error {
	if c.Sender().ID != m.Bot.Cfg.BotOwnerID {
		return c.Send("This command is restricted to the bot owner.")
	}

	err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Flushdb().Build()).Error()
	if err != nil {
		return c.Send("Failed to refresh cache: " + err.Error())
	}

	m.BlacklistCache.Lock()
	m.BlacklistCache.Regexes = make(map[int64][]CachedRegex)
	m.BlacklistCache.StickerSets = make(map[int64]map[string]store.BlacklistItem)
	m.BlacklistCache.Emojis = make(map[int64]map[string]store.BlacklistItem)
	m.BlacklistCache.ApprovedUsers = make(map[int64]map[int64]struct{})
	m.BlacklistCache.Unlock()

	return c.Send("Cache refreshed successfully.")
}

func mention(u *bot.User) string {
	name := strings.ReplaceAll(u.FirstName, "]", "\\]")
	name = strings.ReplaceAll(name, "[", "\\[")
	return "[" + name + "](tg://user?id=" + strconv.FormatInt(u.ID, 10) + ")"
}
