package moderation

import (
	"lappbot/internal/bot"
)

func (m *Module) handleLock(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	permissions := map[string]bool{
		"can_send_messages":       false,
		"can_send_media_messages": false,
		"can_send_polls":          false,
		"can_send_other_messages": false,
	}

	err := m.Bot.Raw("setChatPermissions", map[string]any{
		"chat_id":     c.Chat().ID,
		"permissions": permissions,
	})
	if err != nil {
		return c.Send("Failed to lock group: " + err.Error())
	}

	return c.Send("Group locked. Members cannot send messages.")
}

func (m *Module) handleUnlock(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}

	permissions := map[string]bool{
		"can_send_messages":         true,
		"can_send_media_messages":   true,
		"can_send_polls":            true,
		"can_send_other_messages":   true,
		"can_add_web_page_previews": true,
		"can_invite_users":          true,
	}

	err := m.Bot.Raw("setChatPermissions", map[string]any{
		"chat_id":     c.Chat().ID,
		"permissions": permissions,
	})
	if err != nil {
		return c.Send("Failed to unlock group: " + err.Error())
	}

	return c.Send("Group unlocked. Members can send messages.")
}
