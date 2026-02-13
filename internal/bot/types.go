package bot

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	ID             int64           `json:"message_id"`
	ThreadID       int64           `json:"message_thread_id,omitempty"`
	Date           int64           `json:"date"`
	From           *User           `json:"from,omitempty"`
	Chat           *Chat           `json:"chat"`
	ReplyTo        *Message        `json:"reply_to_message,omitempty"`
	Sticker        *Sticker        `json:"sticker,omitempty"`
	ReplyMarkup    *ReplyMarkup    `json:"reply_markup,omitempty"`
	Video          *Video          `json:"video,omitempty"`
	Audio          *Audio          `json:"audio,omitempty"`
	Document       *Document       `json:"document,omitempty"`
	Voice          *Voice          `json:"voice,omitempty"`
	Animation      *Animation      `json:"animation,omitempty"`
	VideoNote      *VideoNote      `json:"video_note,omitempty"`
	LeftChatMember *User           `json:"left_chat_member,omitempty"`
	Text           string          `json:"text,omitempty"`
	Caption        string          `json:"caption,omitempty"`
	Entities       []MessageEntity `json:"entities,omitempty"`
	NewChatMembers []User          `json:"new_chat_members,omitempty"`
	Photo          []PhotoSize     `json:"photo,omitempty"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size,omitempty"`
}

type Video struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
}

type Audio struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
}

type Document struct {
	FileID string `json:"file_id"`
}

type Voice struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
}

type Animation struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
}

type VideoNote struct {
	FileID   string `json:"file_id"`
	Length   int    `json:"length"`
	Duration int    `json:"duration"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type MessageEntity struct {
	Type          string `json:"type"`
	Url           string `json:"url,omitempty"`
	User          *User  `json:"user,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
}

type Sticker struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	SetName      string `json:"set_name,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	IsAnimated   bool   `json:"is_animated"`
	IsVideo      bool   `json:"is_video"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type ReplyMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard,omitempty"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	Url          string `json:"url,omitempty"`
}

type ChatMember struct {
	User   *User  `json:"user"`
	Status string `json:"status"`
	Role   string `json:"custom_title,omitempty"`
}

type SendMessageReq struct {
	ChatID           int64        `json:"chat_id"`
	Text             string       `json:"text"`
	ParseMode        string       `json:"parse_mode,omitempty"`
	ReplyToMessageID int64        `json:"reply_to_message_id,omitempty"`
	ReplyMarkup      *ReplyMarkup `json:"reply_markup,omitempty"`
}

type EditMessageTextReq struct {
	ChatID      int64        `json:"chat_id"`
	MessageID   int64        `json:"message_id"`
	Text        string       `json:"text"`
	ParseMode   string       `json:"parse_mode,omitempty"`
	ReplyMarkup *ReplyMarkup `json:"reply_markup,omitempty"`
}
