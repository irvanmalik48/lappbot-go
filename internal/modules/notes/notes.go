package notes

import (
	"fmt"
	"strings"

	"lappbot/internal/bot"
	"lappbot/internal/store"

	tele "gopkg.in/telebot.v4"
)

type Module struct {
	Bot   *bot.Bot
	Store *store.Store
}

func New(b *bot.Bot, s *store.Store) *Module {
	return &Module{Bot: b, Store: s}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle("/save", m.handleSave)
	m.Bot.Bot.Handle("/get", m.handleGet)
	m.Bot.Bot.Handle("/clear", m.handleClear)
	m.Bot.Bot.Handle("/notes", m.handleNotes)
	m.Bot.Bot.Handle("/saved", m.handleNotes)
	m.Bot.Bot.Handle("/clearall", m.handleClearAll)
	m.Bot.Bot.Handle("/privatenotes", m.handlePrivateNotes)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "get_note_pm"}, m.onGetNotePM)
	m.Bot.Bot.Use(m.shortcutMiddleware)
}

func (m *Module) shortcutMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
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

func (m *Module) handleSave(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	args := c.Args()
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

	if c.Message().IsReply() {
		reply := c.Message().ReplyTo
		if reply.Photo != nil {
			noteType = "photo"
			fileID = reply.Photo.FileID
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
	return c.Send(fmt.Sprintf("Note `%s` saved.", name), tele.ModeMarkdown)
}

func (m *Module) handleGet(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /get <name>")
	}
	name := strings.ToLower(args[0])

	note, err := m.Store.GetNote(target.ID, name)
	if err != nil {
		return c.Send("Error fetching note.")
	}
	if note == nil {
		return c.Send(fmt.Sprintf("Note `%s` not found.", name), tele.ModeMarkdown)
	}

	return m.sendNoteResponse(c, note)
}

func (m *Module) sendNoteResponse(c tele.Context, note *store.Note) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}

	group, err := m.Store.GetGroup(target.ID)
	if err != nil || group == nil {
		return m.deliverNote(c.Bot(), c.Recipient(), note)
	}

	if group.NotesPrivate {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("Click to get note", "get_note_pm", note.Name)))
		return c.Send(fmt.Sprintf("Click the button below to view note `%s`.", note.Name), markup, tele.ModeMarkdown)
	}

	return m.deliverNote(c.Bot(), c.Recipient(), note)
}

func (m *Module) onGetNotePM(c tele.Context) error {
	name := c.Data()
	// Note: Shortcuts/Buttons usually context-aware, but for PM button, we might need tricky logic.
	// However, GetNote checks chatID. Here c.Chat() might be private if clicked in PM.
	// But the button is usually sent TO group or TO PM.
	// If it's "Click to get note" in group, c.Chat() is group.
	// If it's in PM, c.Chat() is PM.
	// The original code uses c.Chat().ID.
	// If we are in PM and button was sent there, we need to know which group it came from?
	// The original code: note, err := m.Store.GetNote(c.Chat().ID, name)
	// If I am in PM, GetNote with PM ID will fail unless the note is IN PM.
	// Note module as written seems to assume notes are PER CHAT.
	// So if I use /start connection, retrieve a note, it comes to PM.
	// If I click a button attached to that note...
	// Wait, the button "get_note_pm" is added when "NotesPrivate" is true, to move view to PM.
	// If I am ALREADY in PM (via connection), I don't need "get_note_pm".
	// But let's leave this one as is for now, or just update to target?
	// If I am in PM and I click a button, valid context is required.
	// For now, let's update it to respect connection if possible, but buttons are tricky.

	// Actually, let's look at shortcutMiddleware.
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return nil
	}

	note, err := m.Store.GetNote(target.ID, name)
	if err != nil || note == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Note not found."})
	}

	userChat, err := c.Bot().ChatByID(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Please start the bot in private first."})
	}

	err = m.deliverNote(c.Bot(), userChat, note)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Failed to send note."})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Note sent to your PM."})
}

func (m *Module) deliverNote(b tele.API, to tele.Recipient, note *store.Note) error {
	var what interface{}
	opts := &tele.SendOptions{}
	if note.Type == "text" {
		opts.ParseMode = tele.ModeMarkdown
	}

	switch note.Type {
	case "photo":
		what = &tele.Photo{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	case "video":
		what = &tele.Video{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	case "videonote":
		what = &tele.VideoNote{File: tele.File{FileID: note.FileID}}
	case "document":
		what = &tele.Document{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	case "sticker":
		what = &tele.Sticker{File: tele.File{FileID: note.FileID}}
	case "voice":
		what = &tele.Voice{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	case "audio":
		what = &tele.Audio{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	case "animation":
		what = &tele.Animation{File: tele.File{FileID: note.FileID}, Caption: note.Content}
	default:
		what = note.Content
	}

	_, err := b.Send(to, what, opts)
	return err
}

func (m *Module) handleClear(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /clear <name>")
	}
	name := strings.ToLower(args[0])
	err = m.Store.DeleteNote(target.ID, name)
	if err != nil {
		return c.Send("Failed to minimize note.")
	}
	return c.Send(fmt.Sprintf("Note `%s` cleared.", name), tele.ModeMarkdown)
}

func (m *Module) handleNotes(c tele.Context) error {
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
		msg += fmt.Sprintf("- `%s`\n", n.Name)
	}
	return c.Send(msg, tele.ModeMarkdown)
}

func (m *Module) handleClearAll(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
		return nil
	}
	err = m.Store.ClearAllNotes(target.ID)
	if err != nil {
		return c.Send("Failed to clear notes.")
	}
	return c.Send("All notes cleared.")
}

func (m *Module) handlePrivateNotes(c tele.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.IsAdmin(target, c.Sender()) {
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
	return c.Send(fmt.Sprintf("Private notes %s.", status))
}
