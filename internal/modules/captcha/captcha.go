package captcha

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	"github.com/steambap/captcha"
	tele "gopkg.in/telebot.v3"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

const CaptchaDuration = 5 * time.Minute

func (m *Module) Register() {
	m.Bot.Bot.Use(m.CheckCaptcha)
	m.Bot.Bot.Handle("/captcha", m.handleCaptchaCommand)
}

func (m *Module) CheckCaptcha(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Text() == "" {
			return next(c)
		}

		key := fmt.Sprintf("captcha:%d:%d", c.Chat().ID, c.Sender().ID)
		val, err := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Get().Key(key).Build()).ToString()

		if err != nil || val == "" {
			return next(c)
		}

		text := strings.TrimSpace(c.Text())
		if strings.EqualFold(text, val) {
			rights := tele.Rights{
				CanSendMessages: true,
				CanSendMedia:    true,
				CanSendPolls:    true,
				CanSendOther:    true,
				CanAddPreviews:  true,
			}
			m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: rights})

			m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())

			msgKey := fmt.Sprintf("captcha_msg:%d:%d", c.Chat().ID, c.Sender().ID)
			msgIDStr, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Get().Key(msgKey).Build()).ToString()
			if msgIDStr != "" {
				msgID := 0
				fmt.Sscanf(msgIDStr, "%d", &msgID)
				c.Bot().Delete(&tele.Message{ID: msgID, Chat: c.Chat()})
				m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(msgKey).Build())
			}

			c.Delete()

			return c.Reply("Verification successful! You can now chat.")
		} else {
			c.Delete()
			return nil
		}
	}
}

func (m *Module) OnUserJoined(c tele.Context) error {
	group, err := m.Store.GetGroup(c.Chat().ID)
	if err != nil {
		return err
	}

	if group == nil || !group.CaptchaEnabled {
		return nil
	}

	rights := tele.Rights{
		CanSendMessages: true,
		CanSendMedia:    false,
		CanSendPolls:    false,
		CanSendOther:    false,
		CanAddPreviews:  false,
	}

	err = m.Bot.Bot.Restrict(c.Chat(), &tele.ChatMember{User: c.Sender(), Rights: rights})
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to restrict user %s for CAPTCHA verification. Am I admin?", c.Sender().FirstName))
	}

	img, err := captcha.New(150, 50)
	if err != nil {
		return c.Send("Internal error generating captcha.")
	}

	key := fmt.Sprintf("captcha:%d:%d", c.Chat().ID, c.Sender().ID)
	err = m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(key).Value(img.Text).Ex(CaptchaDuration).Build()).Error()
	if err != nil {
		return c.Send("Internal error generating captcha.")
	}

	buf := new(bytes.Buffer)
	if err := img.WriteImage(buf); err != nil {
		return err
	}

	photo := &tele.Photo{File: tele.FromReader(buf)}
	photo.Caption = "Please type the code in the image to verify you are human."

	msg, err := c.Bot().Send(c.Chat(), photo, tele.ModeMarkdown)
	if err != nil {
		return err
	}

	msgKey := fmt.Sprintf("captcha_msg:%d:%d", c.Chat().ID, c.Sender().ID)
	m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(msgKey).Value(fmt.Sprintf("%d", msg.ID)).Ex(CaptchaDuration).Build())

	return nil
}

func (m *Module) handleCaptchaCommand(c tele.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

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
