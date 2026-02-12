package captcha

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"lappbot/internal/bot"
	"lappbot/internal/modules/utility"
	"lappbot/internal/store"

	"github.com/steambap/captcha"
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
	m.Bot.Use(m.CheckCaptcha)
	m.Bot.Handle("/captcha", m.handleCaptchaCommand)
	m.Bot.Handle("new_chat_members", m.OnUserJoined)
}

func (m *Module) CheckCaptcha(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
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
			permissions := map[string]bool{
				"can_send_messages":         true,
				"can_send_media_messages":   true,
				"can_send_polls":            true,
				"can_send_other_messages":   true,
				"can_add_web_page_previews": true,
			}
			m.Bot.Raw("restrictChatMember", map[string]any{
				"chat_id":     c.Chat().ID,
				"user_id":     c.Sender().ID,
				"permissions": permissions,
			})

			m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(key).Build())

			msgKey := fmt.Sprintf("captcha_msg:%d:%d", c.Chat().ID, c.Sender().ID)
			msgIDStr, _ := m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Get().Key(msgKey).Build()).ToString()
			if msgIDStr != "" {
				var msgID int
				fmt.Sscanf(msgIDStr, "%d", &msgID)
				m.Bot.Raw("deleteMessage", map[string]any{
					"chat_id":    c.Chat().ID,
					"message_id": msgID,
				})
				m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Del().Key(msgKey).Build())
			}

			c.Delete()

			return c.Send("Verification successful! You can now chat.")
		} else {
			c.Delete()
			return nil
		}
	}
}

func (m *Module) OnUserJoined(c *bot.Context) error {
	if len(c.Update.Message.NewChatMembers) == 0 {
		return nil
	}

	for _, u := range c.Update.Message.NewChatMembers {
		if u.IsBot {
			continue
		}

		group, err := m.Store.GetGroup(c.Chat().ID)
		if err != nil {
			continue
		}

		if group == nil || !group.CaptchaEnabled {
			continue
		}

		permissions := map[string]bool{
			"can_send_messages":         true,
			"can_send_media_messages":   false,
			"can_send_polls":            false,
			"can_send_other_messages":   false,
			"can_add_web_page_previews": false,
		}
		m.Bot.Raw("restrictChatMember", map[string]any{
			"chat_id":     c.Chat().ID,
			"user_id":     u.ID,
			"permissions": permissions,
		})

		img, err := captcha.New(150, 50)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("captcha:%d:%d", c.Chat().ID, u.ID)
		err = m.Store.Valkey.Do(context.Background(), m.Store.Valkey.B().Set().Key(key).Value(img.Text).Ex(CaptchaDuration).Build()).Error()
		if err != nil {
			continue
		}

		buf := new(bytes.Buffer)
		if err := img.WriteImage(buf); err != nil {
			continue
		}

		code := img.Text

		caption := "Please type the code below to verify you are human."
		if group.GreetingEnabled && group.GreetingMessage != "" {
			userPtr := &u
			caption = utility.ReplacePlaceholders(group.GreetingMessage, userPtr)
			caption += "\n\nVerification Code: " + code
		} else {
			caption = fmt.Sprintf("Welcome! Please type this code to verify: %s", code)
		}

		m.Bot.Raw("sendMessage", map[string]any{
			"chat_id": c.Chat().ID,
			"text":    caption,
		})
	}

	return nil
}

func (m *Module) handleCaptchaCommand(c *bot.Context) error {
	if !m.Bot.IsAdmin(c.Chat(), c.Sender()) {
		return c.Send("You must be an admin to use this command.")
	}

	args := c.Args
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
