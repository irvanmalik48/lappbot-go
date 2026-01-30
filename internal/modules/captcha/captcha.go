package captcha

import (
	"fmt"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v3"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle(tele.OnUserJoined, m.onUserJoined)

	m.Bot.Bot.Handle(&tele.Btn{Unique: "captcha_btn"}, m.onCaptchaButton)
	m.Bot.Bot.Handle("/captcha", m.handleCaptchaCommand)
}

func (m *Module) onUserJoined(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	if group == nil || !group.CaptchaEnabled {
		return nil
	}

	rights := tele.Rights{
		CanSendMessages: false,
		CanSendMedia:    false,
		CanSendPolls:    false,
		CanSendOther:    false,
		CanAddPreviews:  false,
	}

	err = m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: rights})
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to mute user %s for CAPTCHA verification. Am I admin?", c.Sender().FirstName))
	}

	markup := &tele.ReplyMarkup{}

	btn := markup.Data("I'm Human", "captcha_btn", fmt.Sprintf("%d", c.Sender().ID))

	markup.Inline(markup.Row(btn))

	msg, err := c.Bot().Send(c.Chat(), fmt.Sprintf("Welcome [%s](tg://user?id=%d)! Please verify you are human.", c.Sender().FirstName, c.Sender().ID), markup)
	if err != nil {
		return err
	}

	_ = msg
	return nil
}

func (m *Module) onCaptchaButton(c tele.Context) error {
	targetID := c.Data()
	if fmt.Sprintf("%d", c.Sender().ID) != targetID {
		return c.Respond(&tele.CallbackResponse{Text: "This button is not for you!"})
	}

	rights := tele.Rights{
		CanSendMessages: true,
		CanSendMedia:    true,
		CanSendPolls:    true,
		CanSendOther:    true,
		CanAddPreviews:  true,
	}

	err := m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: rights})
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Failed to unmute."})
	}

	c.Respond(&tele.CallbackResponse{Text: "Verified!"})
	return c.Delete()
}

func (m *Module) handleCaptchaCommand(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /captcha <on|off>")
	}

	switch args[0] {
	case "on":
		err := m.Store.UpdateGroupCaptcha(c.Chat().ID, true)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}
		return c.Send("CAPTCHA enabled.")
	case "off":
		err := m.Store.UpdateGroupCaptcha(c.Chat().ID, false)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}
		return c.Send("CAPTCHA disabled.")
	default:
		return c.Send("Usage: /captcha <on|off>")
	}
}
