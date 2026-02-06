package moderation

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"lappbot/internal/store"

	tele "gopkg.in/telebot.v4"
)

func (m *Module) handleBlacklistAdd(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args()
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

	return c.Send(fmt.Sprintf("Blacklisted %s: %s (Action: %s)", kind, value, action))
}

func (m *Module) handleBlacklistRemove(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	args := c.Args()
	if len(args) < 2 {
		return c.Send("Usage: /unbl <type> <value>")
	}

	kind := strings.ToLower(args[0])
	value := args[1]

	err := m.Store.RemoveBlacklistItem(c.Chat().ID, kind, value)
	if err != nil {
		return c.Send("Failed to remove blacklist item: " + err.Error())
	}

	return c.Send(fmt.Sprintf("Removed %s from blacklist: %s", kind, value))
}

func (m *Module) handleBlacklistList(c tele.Context) error {
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

	return c.Send(msg, tele.ModeHTML)
}

func (m *Module) CheckBlacklist(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if m.Bot.IsAdmin(c.Chat(), c.Sender()) {
			return next(c)
		}

		isApproved, err := m.Store.IsApprovedUser(c.Sender().ID, c.Chat().ID)
		if err == nil && isApproved {
			return next(c)
		}

		items, err := m.Store.GetBlacklist(c.Chat().ID)
		if err != nil {
			return next(c)
		}

		for _, item := range items {
			matched := false

			switch item.Type {
			case "regex":
				var re *regexp.Regexp
				if val, ok := m.RegexCache.Load(item.Value); ok {
					re = val.(*regexp.Regexp)
				} else {
					var err error
					re, err = regexp.Compile(item.Value)
					if err == nil {
						m.RegexCache.Store(item.Value, re)
					}
				}

				if re != nil {
					matched = re.MatchString(c.Text())
					if !matched && c.Message().Caption != "" {
						matched = re.MatchString(c.Message().Caption)
					}
				}
			case "sticker_set":
				if c.Message().Sticker != nil {
					if c.Message().Sticker.SetName == item.Value {
						matched = true
					}
				}
			case "emoji":
				if c.Message().Entities != nil {
					for _, entity := range c.Message().Entities {
						if entity.Type == tele.EntityCustomEmoji {
							if entity.CustomEmojiID == item.Value {
								matched = true
								break
							}
						}
					}
				}
			}

			if matched {
				return m.executeBlacklistAction(c, item)
			}
		}

		return next(c)
	}
}

func (m *Module) executeBlacklistAction(c tele.Context, item store.BlacklistItem) error {
	c.Delete()

	switch item.Action {
	case "delete":
		return nil
	case "soft_warn":
		tempMsg, _ := c.Bot().Send(c.Chat(), fmt.Sprintf("%s, that is not allowed here.", mention(c.Sender())))
		go func() {
			time.Sleep(5 * time.Second)
			c.Bot().Delete(tempMsg)
		}()
		return nil
	case "hard_warn":
		count, err := m.Store.AddWarn(c.Sender().ID, c.Chat().ID, "Blacklist violation: "+item.Type, m.Bot.Bot.Me.ID)
		if err != nil {
			return nil
		}
		msg := fmt.Sprintf("%s has been warned (Blacklist).\nTotal Warns: %d/3", mention(c.Sender()), count)
		if count >= 3 {
			m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: c.Sender()})
			m.Bot.Bot.Unban(c.Chat(), c.Sender())
			m.Store.ResetWarns(c.Sender().ID, c.Chat().ID)
			msg += "\nUser kicked (limit reached)."
		}
		c.Send(msg)
		return nil
	case "kick":
		m.Bot.Bot.Unban(c.Chat(), c.Sender())
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
		until := time.Now().Add(duration)
		rights := tele.Rights{
			CanSendMessages: false,
			CanSendMedia:    false,
			CanSendPolls:    false,
			CanSendOther:    false,
		}
		m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: rights, RestrictedUntil: until.Unix()})
		c.Send(fmt.Sprintf("%s muted for %v (Blacklist violation).", mention(c.Sender()), duration))
		return nil
	case "ban":
		m.Bot.Bot.Ban(c.Chat(), &tele.ChatMember{User: c.Sender()})
		c.Send(fmt.Sprintf("%s banned for blacklist violation.", mention(c.Sender())))
		return nil
	}

	return nil
}
