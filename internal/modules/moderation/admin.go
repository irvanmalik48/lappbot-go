package moderation

import (
	"lappbot/internal/bot"
	"strconv"
	"strings"
)

func (m *Module) handleApprove(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to approve them.")
	}
	target := c.Message.ReplyTo.From

	err = m.Store.AddApprovedUser(target.ID, targetChat.ID, c.Sender().ID)
	if err != nil {
		return c.Send("Failed to approve user: " + err.Error())
	}

	m.BlacklistCache.Lock()
	if m.BlacklistCache.ApprovedUsers[targetChat.ID] == nil {
		m.BlacklistCache.ApprovedUsers[targetChat.ID] = make(map[int64]struct{})
	}
	m.BlacklistCache.ApprovedUsers[targetChat.ID][target.ID] = struct{}{}
	m.BlacklistCache.Unlock()

	return c.Send(mention(target)+" is now approved.", "Markdown")
}

func (m *Module) handleUnapprove(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to unapprove them.")
	}
	target := c.Message.ReplyTo.From

	err = m.Store.RemoveApprovedUser(target.ID, targetChat.ID)
	if err != nil {
		return c.Send("Failed to unapprove user: " + err.Error())
	}

	m.Logger.Log(targetChat.ID, "admin", "Unapproved "+target.FirstName+" (ID: "+strconv.FormatInt(target.ID, 10)+") by "+c.Sender().FirstName)
	return c.Send("Unapproved " + mention(target) + ".")
}

func (m *Module) handlePromote(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to promote them.")
	}
	target := c.Message.ReplyTo.From

	title := "Admin"
	args := c.Args
	if len(args) > 0 {
		title = strings.Join(args, " ")
	}

	params := map[string]any{
		"chat_id":                targetChat.ID,
		"user_id":                target.ID,
		"is_anonymous":           false,
		"can_manage_chat":        true,
		"can_post_messages":      true,
		"can_edit_messages":      true,
		"can_delete_messages":    true,
		"can_manage_video_chats": true,
		"can_restrict_members":   true,
		"can_promote_members":    false,
		"can_change_info":        true,
		"can_invite_users":       true,
		"can_pin_messages":       true,
	}

	err = m.Bot.Raw("promoteChatMember", params)
	if err != nil {
		return c.Send("Failed to promote user: " + err.Error())
	}

	m.Bot.Raw("setChatAdministratorCustomTitle", map[string]any{
		"chat_id":      targetChat.ID,
		"user_id":      target.ID,
		"custom_title": title,
	})

	m.Bot.InvalidateAdminCache(targetChat.ID, target.ID)
	m.Logger.Log(targetChat.ID, "admin", "Promoted "+target.FirstName+" to admin ("+title+") by "+c.Sender().FirstName)
	return c.Send(mention(target)+" promoted to admin with title '"+title+"'.", "Markdown")
}

func (m *Module) handleDemote(c *bot.Context) error {
	targetChat, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	if !m.Bot.IsAdmin(targetChat, c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to demote them.")
	}
	target := c.Message.ReplyTo.From

	params := map[string]any{
		"chat_id":                targetChat.ID,
		"user_id":                target.ID,
		"is_anonymous":           false,
		"can_manage_chat":        false,
		"can_post_messages":      false,
		"can_edit_messages":      false,
		"can_delete_messages":    false,
		"can_manage_video_chats": false,
		"can_restrict_members":   false,
		"can_promote_members":    false,
		"can_change_info":        false,
		"can_invite_users":       false,
		"can_pin_messages":       false,
	}

	err = m.Bot.Raw("promoteChatMember", params)
	if err != nil {
		return c.Send("Failed to demote user: " + err.Error())
	}

	m.Bot.InvalidateAdminCache(targetChat.ID, target.ID)
	return c.Send(mention(target)+" demoted to member.", "Markdown")
}
