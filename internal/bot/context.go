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
	req := SendMessageReq{
		ChatID: c.Chat().ID,
		Text:   text,
	}

	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			req.ParseMode = v
		case *ReplyMarkup:
			req.ReplyMarkup = v
		}
	}

	return c.Bot.Raw("sendMessage", req)
}

func (c *Context) Reply(text string, opts ...any) error {
	var replyTo int64
	if c.Message != nil {
		replyTo = c.Message.ID
	} else if c.Callback != nil && c.Callback.Message != nil {
		replyTo = c.Callback.Message.ID
	}

	req := SendMessageReq{
		ChatID:           c.Chat().ID,
		Text:             text,
		ReplyToMessageID: replyTo,
	}
	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			req.ParseMode = v
		case *ReplyMarkup:
			req.ReplyMarkup = v
		}
	}
	return c.Bot.Raw("sendMessage", req)
}

func (c *Context) Delete() error {
	var msgID int64
	var chatID int64

	if c.Message != nil && c.Message.Chat != nil {
		msgID = c.Message.ID
		chatID = c.Message.Chat.ID
	} else if c.Callback != nil && c.Callback.Message != nil && c.Callback.Message.Chat != nil {
		msgID = c.Callback.Message.ID
		chatID = c.Callback.Message.Chat.ID
	} else {
		return nil
	}

	return c.Bot.Raw("deleteMessage", map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
	})
}

func (c *Context) Edit(text string, opts ...any) error {
	if c.Callback != nil && c.Callback.Message != nil && c.Callback.Message.Chat != nil {
		req := EditMessageTextReq{
			ChatID:    c.Callback.Message.Chat.ID,
			MessageID: c.Callback.Message.ID,
			Text:      text,
		}
		for _, opt := range opts {
			switch v := opt.(type) {
			case string:
				req.ParseMode = v
			case *ReplyMarkup:
				req.ReplyMarkup = v
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
	if c.Message != nil && c.Message.Chat != nil {
		return c.Message.Chat
	}
	if c.Callback != nil && c.Callback.Message != nil && c.Callback.Message.Chat != nil {
		return c.Callback.Message.Chat
	}
	return &Chat{}
}

func (c *Context) Sender() *User {
	if c.Callback != nil && c.Callback.From != nil {
		return c.Callback.From
	}
	if c.Message != nil && c.Message.From != nil {
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
