package moderation

import (
	"lappbot/internal/bot"
)

func (m *Module) handleLock(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}

	permissions := map[string]bool{
		"can_send_messages":       false,
		"can_send_media_messages": false,
		"can_send_polls":          false,
		"can_send_other_messages": false,
	}

	err = m.Bot.Raw("setChatPermissions", map[string]any{
		"chat_id":     targetChat.ID,
		"permissions": permissions,
	})
	if err != nil {
		return c.Send("Failed to lock group.")
	}

	m.Logger.Log(targetChat.ID, "admin", "Group locked by "+c.Sender().FirstName)
	return c.Send("Group locked.")
}

func (m *Module) handleUnlock(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	permissions := map[string]bool{
		"can_send_messages":         true,
		"can_send_media_messages":   true,
		"can_send_polls":            true,
		"can_send_other_messages":   true,
		"can_add_web_page_previews": true,
		"can_invite_users":          true,
	}

	err = m.Bot.Raw("setChatPermissions", map[string]any{
		"chat_id":     targetChat.ID,
		"permissions": permissions,
	})
	if err != nil {
		return c.Send("Failed to unlock group.")
	}

	m.Logger.Log(targetChat.ID, "admin", "Group unlocked by "+c.Sender().FirstName)
	return c.Send("Group unlocked.")
}
