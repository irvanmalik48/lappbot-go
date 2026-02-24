package notes

import (
	"strings"

	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"
)

type Module struct {
	Bot    *bot.Bot
	Store  *store.Store
	Logger *logging.Module
}

func New(b *bot.Bot, s *store.Store, logger *logging.Module) *Module {
	return &Module{Bot: b, Store: s, Logger: logger}
}

func (m *Module) Register() {
	m.Bot.Handle("/save", m.handleSave)
	m.Bot.Handle("/get", m.handleGet)
	m.Bot.Handle("/clear", m.handleClear)
	m.Bot.Handle("/notes", m.handleNotes)
	m.Bot.Handle("/saved", m.handleNotes)
	m.Bot.Handle("/clearall", m.handleClearAll)
	m.Bot.Handle("/privatenotes", m.handlePrivateNotes)
	m.Bot.Handle("get_note_pm", m.onGetNotePM)
	m.Bot.Use(m.shortcutMiddleware)
}

func (m *Module) shortcutMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(c *bot.Context) error {
		text := c.Text()
		if len(text) > 1 && strings.HasPrefix(text, "#") {
			name := strings.ToLower(text[1:])
			target, err := m.Bot.GetTargetChat(c)
			if err != nil {
				return next(c)
			}
			note, err := m.Store.GetNote(target.ID, name)
			if err == nil && note != nil {
				return m.sendNoteResponse(c, note)
			}
		}
		return next(c)
	}
}

func (m *Module) handleSave(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /save <name> [content]")
	}
	name := strings.ToLower(args[0])
	content := ""
	if len(args) > 1 {
		content = strings.Join(args[1:], " ")
	}

	noteType := "text"
	fileID := ""

	if c.Message.ReplyTo != nil {
		reply := c.Message.ReplyTo
		if len(reply.Photo) > 0 {
			noteType = "photo"
			fileID = reply.Photo[0].FileID
			if content == "" {
				content = reply.Caption
			}
		} else if reply.Video != nil {
			noteType = "video"
			fileID = reply.Video.FileID
			if content == "" {
				content = reply.Caption
			}
		} else if reply.VideoNote != nil {
			noteType = "videonote"
			fileID = reply.VideoNote.FileID
		} else if reply.Animation != nil {
			noteType = "animation"
			fileID = reply.Animation.FileID
			if content == "" {
				content = reply.Caption
			}
		} else if reply.Sticker != nil {
			noteType = "sticker"
			fileID = reply.Sticker.FileID
		} else if reply.Voice != nil {
			noteType = "voice"
			fileID = reply.Voice.FileID
			if content == "" {
				content = reply.Caption
			}
		} else if reply.Audio != nil {
			noteType = "audio"
			fileID = reply.Audio.FileID
			if content == "" {
				content = reply.Caption
			}
		} else if reply.Document != nil {
			noteType = "document"
			fileID = reply.Document.FileID
			if content == "" {
				content = reply.Caption
			}
		} else {
			if content == "" {
				content = reply.Text
				if content == "" {
					content = reply.Caption
				}
			}
		}
	}

	if content == "" && fileID == "" {
		return c.Send("You need to provide content or reply to a message to save a note.")
	}

	err = m.Store.SaveNote(target.ID, name, content, noteType, fileID, c.Sender().ID)
	if err != nil {
		return c.Send("Failed to save note.")
	}
	m.Logger.Log(target.ID, "other", "Note saved: "+name)
	return c.Send("Note `"+name+"` saved.", "Markdown")
}

func (m *Module) handleGet(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /get <name>")
	}
	name := strings.ToLower(args[0])

	note, err := m.Store.GetNote(target.ID, name)
	if err != nil {
		return c.Send("Error fetching note.")
	}
	if note == nil {
		return c.Send("Note `"+name+"` not found.", "Markdown")
	}

	return m.sendNoteResponse(c, note)
}

func (m *Module) sendNoteResponse(c *bot.Context, note *store.Note) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	group, err := m.Store.GetGroup(target.ID)
	if err != nil || group == nil {
		return m.deliverNote(c.Chat().ID, note)
	}

	if group.NotesPrivate {
		markup := &bot.ReplyMarkup{}
		btn := bot.InlineKeyboardButton{
			Text:         "Click to get note",
			CallbackData: "get_note_pm|" + note.Name,
		}
		markup.InlineKeyboard = [][]bot.InlineKeyboardButton{{btn}}
		return c.Send("Click the button below to view note `"+note.Name+"`.", markup, "Markdown")
	}

	return m.deliverNote(c.Chat().ID, note)
}

func (m *Module) onGetNotePM(c *bot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		c.Respond("Invalid data")
		return nil
	}
	name := parts[1]

	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return nil
	}

	note, err := m.Store.GetNote(target.ID, name)
	if err != nil || note == nil {
		c.Respond("Note not found.")
		return nil
	}

	err = m.deliverNote(c.Sender().ID, note)
	if err != nil {
		c.Respond("Failed to send note. Start me in PM first?")
		return nil
	}
	c.Respond("Note sent to your PM.")
	return nil
}

func (m *Module) deliverNote(chatID int64, note *store.Note) error {
	req := map[string]any{
		"chat_id": chatID,
	}

	switch note.Type {
	case "photo":
		req["photo"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendPhoto", req)
	case "video":
		req["video"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendVideo", req)
	case "videonote":
		req["video_note"] = note.FileID
		return m.Bot.Raw("sendVideoNote", req)
	case "document":
		req["document"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendDocument", req)
	case "sticker":
		req["sticker"] = note.FileID
		return m.Bot.Raw("sendSticker", req)
	case "voice":
		req["voice"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendVoice", req)
	case "audio":
		req["audio"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendAudio", req)
	case "animation":
		req["animation"] = note.FileID
		req["caption"] = note.Content
		return m.Bot.Raw("sendAnimation", req)
	default:
		req["text"] = note.Content
		req["parse_mode"] = "Markdown"
		return m.Bot.Raw("sendMessage", req)
	}
}

func (m *Module) handleClear(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender(), "can_change_info") {
		return nil
	}
	if !m.Bot.CheckBotAdmin(c, target, "can_change_info") {
		return nil
	}
	args := c.Args
	if len(args) == 0 {
		return c.Send("Usage: /clear <name>")
	}
	name := strings.ToLower(args[0])
	err = m.Store.DeleteNote(target.ID, name)
	if err != nil {
		return c.Send("Failed to minimize note.")
	}
	m.Logger.Log(target.ID, "other", "Note deleted: "+name)
	return c.Send("Note `"+name+"` cleared.", "Markdown")
}

func (m *Module) handleNotes(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	notes, err := m.Store.GetNotes(target.ID)
	if err != nil {
		return c.Send("Failed to fetch notes.")
	}
	if len(notes) == 0 {
		return c.Send("No notes saved in this chat.")
	}
	msg := "**Notes in this chat:**\n"
	for _, n := range notes {
		msg += "- `" + n.Name + "`\n"
	}
	return c.Send(msg, "Markdown")
}

func (m *Module) handleClearAll(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender(), "can_change_info") {
		return nil
	}
	if !m.Bot.CheckBotAdmin(c, target, "can_change_info") {
		return nil
	}
	err = m.Store.ClearAllNotes(target.ID)
	if err != nil {
		return c.Send("Failed to clear notes.")
	}
	m.Logger.Log(target.ID, "other", "All notes cleared by "+c.Sender().FirstName)
	return c.Send("All notes cleared.")
}

func (m *Module) handlePrivateNotes(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender(), "can_change_info") {
		return nil
	}
	if !m.Bot.CheckBotAdmin(c, target, "can_change_info") {
		return nil
	}
	group, err := m.Store.GetGroup(target.ID)
	if err != nil || group == nil {
		return c.Send("Group not found.")
	}
	newState := !group.NotesPrivate
	err = m.Store.SetNotesPrivate(target.ID, newState)
	if err != nil {
		return c.Send("Failed to update settings.")
	}
	status := "disabled"
	if newState {
		status = "enabled"
	}
	return c.Send("Private notes " + status + ".")
}
