package moderation

import (
	"fmt"
	"lappbot/internal/bot"
	"strings"
)

func (m *Module) handleApprove(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to approve them.")
	}
	target := c.Message.ReplyTo.From

	err := m.Store.AddApprovedUser(target.ID, c.Chat().ID, c.Sender().ID)
	if err != nil {
		return c.Send("Failed to approve user: " + err.Error())
	}

	m.BlacklistCache.Lock()
	if m.BlacklistCache.ApprovedUsers[c.Chat().ID] == nil {
		m.BlacklistCache.ApprovedUsers[c.Chat().ID] = make(map[int64]struct{})
	}
	m.BlacklistCache.ApprovedUsers[c.Chat().ID][target.ID] = struct{}{}
	m.BlacklistCache.Unlock()

	return c.Send(fmt.Sprintf("%s is now approved.", mention(target)), "Markdown")
}

func (m *Module) handleUnapprove(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to unapprove them.")
	}
	target := c.Message.ReplyTo.From

	err := m.Store.RemoveApprovedUser(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Failed to unapprove user: " + err.Error())
	}

	m.BlacklistCache.Lock()
	if m.BlacklistCache.ApprovedUsers[c.Chat().ID] != nil {
		delete(m.BlacklistCache.ApprovedUsers[c.Chat().ID], target.ID)
	}
	m.BlacklistCache.Unlock()

	return c.Send(fmt.Sprintf("%s is no longer approved.", mention(target)), "Markdown")
}

func (m *Module) handlePromote(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
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

	params := map[string]interface{}{
		"chat_id":                c.Chat().ID,
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

	err := m.Bot.Raw("promoteChatMember", params)
	if err != nil {
		return c.Send("Failed to promote user: " + err.Error())
	}

	m.Bot.Raw("setChatAdministratorCustomTitle", map[string]interface{}{
		"chat_id":      c.Chat().ID,
		"user_id":      target.ID,
		"custom_title": title,
	})

	m.Bot.InvalidateAdminCache(c.Chat().ID, target.ID)
	return c.Send(fmt.Sprintf("%s promoted to admin with title '%s'.", mention(target), title), "Markdown")
}

func (m *Module) handleDemote(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user to demote them.")
	}
	target := c.Message.ReplyTo.From

	params := map[string]interface{}{
		"chat_id":                c.Chat().ID,
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

	err := m.Bot.Raw("promoteChatMember", params)
	if err != nil {
		return c.Send("Failed to demote user: " + err.Error())
	}

	m.Bot.InvalidateAdminCache(c.Chat().ID, target.ID)
	return c.Send(fmt.Sprintf("%s demoted to member.", mention(target)), "Markdown")
}
