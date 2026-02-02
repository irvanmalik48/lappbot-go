package moderation

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (m *Module) handleApprove(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to approve them.")
	}
	target := c.Message().ReplyTo.Sender

	err := m.Store.AddApprovedUser(target.ID, c.Chat().ID, c.Sender().ID)
	if err != nil {
		return c.Send("Failed to approve user: " + err.Error())
	}

	return c.Send(fmt.Sprintf("%s is now approved.", mention(target)), tele.ModeMarkdown)
}

func (m *Module) handleUnapprove(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to unapprove them.")
	}
	target := c.Message().ReplyTo.Sender

	err := m.Store.RemoveApprovedUser(target.ID, c.Chat().ID)
	if err != nil {
		return c.Send("Failed to unapprove user: " + err.Error())
	}

	return c.Send(fmt.Sprintf("%s is no longer approved.", mention(target)), tele.ModeMarkdown)
}

func (m *Module) handlePromote(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to promote them.")
	}
	target := c.Message().ReplyTo.Sender

	title := "Admin"
	args := c.Args()
	if len(args) > 0 {
		title = strings.Join(args, " ")
	}

	rights := tele.Rights{
		CanBeEdited:         true,
		CanChangeInfo:       true,
		CanPostMessages:     true,
		CanEditMessages:     true,
		CanDeleteMessages:   true,
		CanInviteUsers:      true,
		CanRestrictMembers:  true,
		CanPinMessages:      true,
		CanPromoteMembers:   false,
		CanManageVideoChats: true,
		CanManageChat:       true,
	}

	err := m.Bot.Bot.Promote(c.Chat(), &tele.ChatMember{
		User:   target,
		Rights: rights,
		Title:  title,
	})
	if err != nil {
		return c.Send("Failed to promote user: " + err.Error())
	}

	m.Bot.InvalidateAdminCache(c.Chat().ID, target.ID)
	return c.Send(fmt.Sprintf("%s promoted to admin with title '%s'.", mention(target), title), tele.ModeMarkdown)
}

func (m *Module) handleDemote(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return nil
	}
	if !c.Message().IsReply() {
		return c.Send("Reply to a user to demote them.")
	}
	target := c.Message().ReplyTo.Sender

	rights := tele.Rights{
		CanBeEdited:         false,
		CanChangeInfo:       false,
		CanPostMessages:     false,
		CanEditMessages:     false,
		CanDeleteMessages:   false,
		CanInviteUsers:      false,
		CanRestrictMembers:  false,
		CanPinMessages:      false,
		CanPromoteMembers:   false,
		CanManageVideoChats: false,
		CanManageChat:       false,
	}

	err := m.Bot.Bot.Promote(c.Chat(), &tele.ChatMember{
		User:   target,
		Rights: rights,
	})
	if err != nil {
		return c.Send("Failed to demote user: " + err.Error())
	}

	m.Bot.InvalidateAdminCache(c.Chat().ID, target.ID)
	return c.Send(fmt.Sprintf("%s demoted to member.", mention(target)), tele.ModeMarkdown)
}
