package bot

type Context struct {
	Bot      *Bot
	Update   *Update
	Message  *Message
	Callback *CallbackQuery
	Args     []string
}

func (c *Context) Reset(b *Bot, u *Update) {
	c.Bot = b
	c.Update = u
	c.Message = nil
	c.Callback = nil
	c.Args = nil
}

func (c *Context) Send(text string, opts ...any) error {
	req := map[string]any{
		"chat_id": c.Chat().ID,
		"text":    text,
	}

	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			req["parse_mode"] = v
		case *ReplyMarkup:
			req["reply_markup"] = v
		}
	}

	return c.Bot.Raw("sendMessage", req)
}

func (c *Context) Reply(text string, opts ...any) error {
	req := map[string]any{
		"chat_id":             c.Chat().ID,
		"text":                text,
		"reply_to_message_id": c.Message.ID,
	}
	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			req["parse_mode"] = v
		case *ReplyMarkup:
			req["reply_markup"] = v
		}
	}
	return c.Bot.Raw("sendMessage", req)
}

func (c *Context) Delete() error {
	msgID := c.Message.ID
	chatID := c.Chat().ID
	return c.Bot.Raw("deleteMessage", map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
	})
}

func (c *Context) Edit(text string, opts ...any) error {
	if c.Callback != nil && c.Callback.Message != nil {
		req := map[string]any{
			"chat_id":    c.Callback.Message.Chat.ID,
			"message_id": c.Callback.Message.ID,
			"text":       text,
		}
		for _, opt := range opts {
			switch v := opt.(type) {
			case string:
				req["parse_mode"] = v
			case *ReplyMarkup:
				req["reply_markup"] = v
			}
		}
		return c.Bot.Raw("editMessageText", req)
	}
	return nil
}

func (c *Context) Respond(opts ...any) error {
	if c.Callback == nil {
		return nil
	}
	req := map[string]any{
		"callback_query_id": c.Callback.ID,
	}
	for _, opt := range opts {
		if s, ok := opt.(string); ok {
			req["text"] = s
		}
	}
	return c.Bot.Raw("answerCallbackQuery", req)
}

func (c *Context) Chat() *Chat {
	if c.Message != nil {
		return c.Message.Chat
	}
	if c.Callback != nil && c.Callback.Message != nil {
		return c.Callback.Message.Chat
	}
	return &Chat{}
}

func (c *Context) Sender() *User {
	if c.Callback != nil {
		return c.Callback.From
	}
	if c.Message != nil {
		return c.Message.From
	}
	return &User{}
}

func (c *Context) Text() string {
	if c.Message != nil {
		return c.Message.Text
	}
	return ""
}

func (c *Context) Data() string {
	if c.Callback != nil {
		return c.Callback.Data
	}
	return ""
}
