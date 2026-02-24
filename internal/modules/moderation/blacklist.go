package moderation

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"
)

func (m *Module) handleBlacklistAdd(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, targetChat, c.Sender(), "can_restrict_members") {
		return nil
	}

	args := c.Args
	if len(args) < 2 {
		return c.Send("Usage: /bl <type> <value> [action] [duration]\nTypes: regex, sticker_set, emoji\nActions: delete, soft_warn, hard_warn, kick, mute, ban")
	}

	kind := strings.ToLower(args[0])
	value := args[1]
	action := "delete"
	duration := ""

	if len(args) > 2 {
		action = strings.ToLower(args[2])
	}
	if len(args) > 3 {
		duration = args[3]
	}

	if kind != "regex" && kind != "sticker_set" && kind != "emoji" {
		return c.Send("Invalid type. Use: regex, sticker_set, emoji")
	}
	if action != "delete" && action != "soft_warn" && action != "hard_warn" && action != "kick" && action != "mute" && action != "ban" {
		return c.Send("Invalid action. Use: delete, soft_warn, hard_warn, kick, mute, ban")
	}

	err = m.Store.AddBlacklistItem(targetChat.ID, kind, value, action, duration)
	if err != nil {
		return c.Send("Failed to add blacklist item: " + err.Error())
	}

	m.BlacklistCache.Lock()
	delete(m.BlacklistCache.Regexes, targetChat.ID)
	delete(m.BlacklistCache.StickerSets, targetChat.ID)
	delete(m.BlacklistCache.Emojis, targetChat.ID)
	m.BlacklistCache.Unlock()

	m.Logger.Log(targetChat.ID, "admin", "Blacklisted "+kind+": "+value+" (Action: "+action+") by "+c.Sender().FirstName)
	return c.Send("Blacklisted " + kind + ": " + value + " (Action: " + action + ")")
}

func (m *Module) handleBlacklistRemove(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.CheckAdmin(c, targetChat, c.Sender(), "can_restrict_members") {
		return nil
	}

	args := c.Args
	if len(args) < 2 {
		return c.Send("Usage: /unbl <type> <value>")
	}

	kind := strings.ToLower(args[0])
	value := args[1]

	err = m.Store.RemoveBlacklistItem(targetChat.ID, kind, value)
	if err != nil {
		return c.Send("Failed to remove blacklist item: " + err.Error())
	}

	m.BlacklistCache.Lock()
	delete(m.BlacklistCache.Regexes, targetChat.ID)
	delete(m.BlacklistCache.StickerSets, targetChat.ID)
	delete(m.BlacklistCache.Emojis, targetChat.ID)
	m.BlacklistCache.Unlock()

	m.Logger.Log(targetChat.ID, "admin", "Removed "+kind+" from blacklist: "+value+" by "+c.Sender().FirstName)
	return c.Send("Removed " + kind + " from blacklist: " + value)
}

func (m *Module) handleBlacklistList(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	items, err := m.Store.GetBlacklist(targetChat.ID)
	if err != nil {
		return c.Send("Failed to fetch blacklist: " + err.Error())
	}

	if len(items) == 0 {
		return c.Send("Blacklist is empty.")
	}

	msg := "<b>Blacklist:</b>\n"
	for _, item := range items {
		msg += "• [" + html.EscapeString(item.Type) + "] " + html.EscapeString(item.Value) + " (Action: " + html.EscapeString(item.Action) + ")\n"
	}

	return c.Send(msg, "HTML")
}

func (m *Module) LoadBlacklistCache(groupID int64) error {
	items, err := m.Store.GetBlacklist(groupID)
	if err != nil {
		return err
	}

	m.BlacklistCache.Lock()
	defer m.BlacklistCache.Unlock()

	var regexes []CachedRegex
	stickers := make(map[string]store.BlacklistItem, len(items))
	emojis := make(map[string]store.BlacklistItem, len(items))

	for _, item := range items {
		switch item.Type {
		case "regex":
			re, err := regexp.Compile(item.Value)
			if err == nil {
				regexes = append(regexes, CachedRegex{Re: re, Item: item})
			}
		case "sticker_set":
			stickers[item.Value] = item
		case "emoji":
			emojis[item.Value] = item
		}
	}

	m.BlacklistCache.Regexes[groupID] = regexes
	m.BlacklistCache.StickerSets[groupID] = stickers
	m.BlacklistCache.Emojis[groupID] = emojis

	return nil
}

func (m *Module) LoadApprovedUsers(groupID int64) error {
	users, err := m.Store.GetApprovedUsers(groupID)
	if err != nil {
		return err
	}

	userMap := make(map[int64]struct{}, len(users))
	for _, uid := range users {
		userMap[uid] = struct{}{}
	}

	m.BlacklistCache.Lock()
	m.BlacklistCache.ApprovedUsers[groupID] = userMap
	m.BlacklistCache.Unlock()

	return nil
}

func (m *Module) CheckBlacklist(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
		if c.Message == nil {
			return next(c)
		}

		if m.Bot.IsAdmin(c.Chat(), c.Sender()) {
			return next(c)
		}

		m.BlacklistCache.RLock()
		approvedMap, ok := m.BlacklistCache.ApprovedUsers[c.Chat().ID]
		if !ok {
			m.BlacklistCache.RUnlock()
			if err := m.LoadApprovedUsers(c.Chat().ID); err != nil {
			}
			m.BlacklistCache.RLock()
			approvedMap = m.BlacklistCache.ApprovedUsers[c.Chat().ID]
		}

		if approvedMap != nil {
			if _, exists := approvedMap[c.Sender().ID]; exists {
				m.BlacklistCache.RUnlock()
				return next(c)
			}
		}
		m.BlacklistCache.RUnlock()

		if approvedMap == nil {
			isApproved, err := m.Store.IsApprovedUser(c.Sender().ID, c.Chat().ID)
			if err == nil && isApproved {
				return next(c)
			}
		}

		m.BlacklistCache.RLock()
		regexes, ok := m.BlacklistCache.Regexes[c.Chat().ID]
		if !ok {
			m.BlacklistCache.RUnlock()
			if err := m.LoadBlacklistCache(c.Chat().ID); err != nil {
				return next(c)
			}
			m.BlacklistCache.RLock()
			regexes = m.BlacklistCache.Regexes[c.Chat().ID]
		}
		stickers := m.BlacklistCache.StickerSets[c.Chat().ID]
		emojis := m.BlacklistCache.Emojis[c.Chat().ID]
		m.BlacklistCache.RUnlock()

		text := c.Message.Text
		caption := c.Message.Caption

		for _, re := range regexes {
			if re.Re.MatchString(text) || (caption != "" && re.Re.MatchString(caption)) {
				return m.executeBlacklistAction(c, re.Item)
			}
		}

		if c.Message.Sticker != nil {
			if item, exists := stickers[c.Message.Sticker.SetName]; exists {
				return m.executeBlacklistAction(c, item)
			}
		}

		if c.Message.Entities != nil {
			for _, entity := range c.Message.Entities {
				if entity.Type == "custom_emoji" {
					if item, exists := emojis[entity.CustomEmojiID]; exists {
						return m.executeBlacklistAction(c, item)
					}
				}
			}
		}

		return next(c)
	}
}

func (m *Module) executeBlacklistAction(c *bot.Context, item store.BlacklistItem) error {
	c.Delete()

	switch item.Action {
	case "delete":
		return nil
	case "soft_warn":
		m.Bot.Raw("sendMessage", map[string]any{
			"chat_id":    c.Chat().ID,
			"text":       mention(c.Sender()) + ", that is not allowed here.",
			"parse_mode": "Markdown",
		})
		return nil
	case "hard_warn":
		count, err := m.Store.AddWarn(c.Sender().ID, c.Chat().ID, "Blacklist violation: "+item.Type, 0)
		if err != nil {
			return nil
		}
		msg := mention(c.Sender()) + " has been warned (Blacklist).\nTotal Warns: " + strconv.Itoa(count) + "/3"
		if count >= 3 {
			m.Bot.Raw("banChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": c.Sender().ID})
			m.Bot.Raw("unbanChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": c.Sender().ID, "only_if_banned": true})
			m.Store.ResetWarns(c.Sender().ID, c.Chat().ID)
			msg += "\nUser kicked (limit reached)."
		}
		c.Send(msg)
		return nil
	case "kick":
		m.Bot.Raw("unbanChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": c.Sender().ID})
		c.Send(mention(c.Sender()) + " kicked for blacklist violation.")
		return nil
	case "mute":
		duration := 30 * time.Minute
		if item.ActionDuration != "" {
			d, err := time.ParseDuration(item.ActionDuration)
			if err == nil {
				duration = d
			}
		}
		until := time.Now().Add(duration).Unix()
		permissions := map[string]bool{
			"can_send_messages":       false,
			"can_send_media_messages": false,
			"can_send_polls":          false,
			"can_send_other_messages": false,
		}
		m.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     c.Chat().ID,
			"user_id":     c.Sender().ID,
			"permissions": permissions,
			"until_date":  until,
		})
		c.Send(mention(c.Sender()) + " muted for " + duration.String() + " (Blacklist violation).")
		return nil
	case "ban":
		m.Bot.Raw("banChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": c.Sender().ID})
		c.Send(mention(c.Sender()) + " banned for blacklist violation.")
		return nil
	}

	return nil
}
