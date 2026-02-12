package moderation

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"
)

func (m *Module) handleBlacklistAdd(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
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

	err := m.Store.AddBlacklistItem(c.Chat().ID, kind, value, action, duration)
	if err != nil {
		return c.Send("Failed to add blacklist item: " + err.Error())
	}

	m.BlacklistCache.Lock()
	delete(m.BlacklistCache.Regexes, c.Chat().ID)
	delete(m.BlacklistCache.StickerSets, c.Chat().ID)
	delete(m.BlacklistCache.Emojis, c.Chat().ID)
	m.BlacklistCache.Unlock()

	return c.Send(fmt.Sprintf("Blacklisted %s: %s (Action: %s)", kind, value, action))
}

func (m *Module) handleBlacklistRemove(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args
	if len(args) < 2 {
		return c.Send("Usage: /unbl <type> <value>")
	}

	kind := strings.ToLower(args[0])
	value := args[1]

	err := m.Store.RemoveBlacklistItem(c.Chat().ID, kind, value)
	if err != nil {
		return c.Send("Failed to remove blacklist item: " + err.Error())
	}

	m.BlacklistCache.Lock()
	delete(m.BlacklistCache.Regexes, c.Chat().ID)
	delete(m.BlacklistCache.StickerSets, c.Chat().ID)
	delete(m.BlacklistCache.Emojis, c.Chat().ID)
	m.BlacklistCache.Unlock()

	return c.Send(fmt.Sprintf("Removed %s from blacklist: %s", kind, value))
}

func (m *Module) handleBlacklistList(c *bot.Context) error {
	items, err := m.Store.GetBlacklist(c.Chat().ID)
	if err != nil {
		return c.Send("Failed to fetch blacklist: " + err.Error())
	}

	if len(items) == 0 {
		return c.Send("Blacklist is empty.")
	}

	msg := "<b>Blacklist:</b>\n"
	for _, item := range items {
		msg += fmt.Sprintf("• [%s] %s (Action: %s)\n", html.EscapeString(item.Type), html.EscapeString(item.Value), html.EscapeString(item.Action))
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

	var regexes []*regexp.Regexp
	stickers := make(map[string]store.BlacklistItem, len(items))
	emojis := make(map[string]store.BlacklistItem, len(items))

	for _, item := range items {
		switch item.Type {
		case "regex":
			re, err := regexp.Compile(item.Value)
			if err == nil {
				regexes = append(regexes, re)
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
			if re.MatchString(text) || (caption != "" && re.MatchString(caption)) {
				items, _ := m.Store.GetBlacklist(c.Chat().ID)
				for _, item := range items {
					if item.Type == "regex" && item.Value == re.String() {
						return m.executeBlacklistAction(c, item)
					}
				}
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
			"text":       fmt.Sprintf("%s, that is not allowed here.", mention(c.Sender())),
			"parse_mode": "Markdown",
		})
		return nil
	case "hard_warn":
		count, err := m.Store.AddWarn(c.Sender().ID, c.Chat().ID, "Blacklist violation: "+item.Type, m.Bot.Client.Timeout.Microseconds())
		if err != nil {
			return nil
		}
		msg := fmt.Sprintf("%s has been warned (Blacklist).\nTotal Warns: %d/3", mention(c.Sender()), count)
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
		c.Send(fmt.Sprintf("%s kicked for blacklist violation.", mention(c.Sender())))
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
		c.Send(fmt.Sprintf("%s muted for %v (Blacklist violation).", mention(c.Sender()), duration))
		return nil
	case "ban":
		m.Bot.Raw("banChatMember", map[string]any{"chat_id": c.Chat().ID, "user_id": c.Sender().ID})
		c.Send(fmt.Sprintf("%s banned for blacklist violation.", mention(c.Sender())))
		return nil
	}

	return nil
}
