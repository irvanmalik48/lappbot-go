package filters

import (
	"html"
	"strings"
	"sync"

	"lappbot/internal/bot"
	"lappbot/internal/modules/logging"
	"lappbot/internal/store"
)

type FiltersCache struct {
	sync.RWMutex
	Filters map[int64][]store.Filter
}

type FiltersModule struct {
	Bot    *bot.Bot
	Store  *store.Store
	Cache  *FiltersCache
	Logger *logging.Module
}

func New(b *bot.Bot, s *store.Store, l *logging.Module) *FiltersModule {
	return &FiltersModule{
		Bot:   b,
		Store: s,
		Cache: &FiltersCache{
			Filters: make(map[int64][]store.Filter),
		},
		Logger: l,
	}
}

func (m *FiltersModule) Register() {
	m.Bot.Handle("/filter", m.handleFilter)
	m.Bot.Handle("/stop", m.handleStop)
	m.Bot.Handle("/filters", m.handleFilters)

	m.Bot.Handle("on_text", m.handleText)
}

func (m *FiltersModule) handleFilter(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender(), "can_change_info") {
		return c.Send("You must be an admin to use this command.")
	}
	if !m.Bot.CheckBotAdmin(c, target, "can_change_info") {
		return nil
	}
	args := c.Args
	if len(args) < 1 {
		return c.Send("Usage: /filter <trigger> <response> (or reply to a message)")
	}

	trigger := strings.ToLower(args[0])
	var response, kind string

	if c.Message.ReplyTo != nil {
		msg := c.Message.ReplyTo
		if msg.Sticker != nil {
			kind = "sticker"
			response = msg.Sticker.FileID
		} else if len(msg.Photo) > 0 {
			kind = "photo"
			response = msg.Photo[len(msg.Photo)-1].FileID
		} else if msg.Video != nil {
			kind = "video"
			response = msg.Video.FileID
		} else if msg.Voice != nil {
			kind = "voice"
			response = msg.Voice.FileID
		} else if msg.Audio != nil {
			kind = "audio"
			response = msg.Audio.FileID
		} else if msg.Document != nil {
			kind = "document"
			response = msg.Document.FileID
		} else if msg.VideoNote != nil {
			kind = "video_note"
			response = msg.VideoNote.FileID
		} else if msg.Animation != nil {
			kind = "animation"
			response = msg.Animation.FileID
		} else {
			kind = "text"
			response = msg.Text
			if response == "" {
				response = msg.Caption
			}
		}
	} else if len(args) >= 2 {
		kind = "text"
		response = strings.Join(args[1:], " ")
	}

	if response == "" {
		return c.Send("Please provide a response text or reply to a message.")
	}

	err = m.Store.AddFilter(target.ID, trigger, response, kind)
	if err != nil {
		return c.Send("Failed to save filter: " + err.Error())
	}

	m.Cache.Lock()
	delete(m.Cache.Filters, target.ID)
	m.Cache.Unlock()

	m.Logger.Log(target.ID, "other", "Filter added by "+c.Sender().FirstName+"\nTrigger: "+trigger+"\nType: "+kind)

	return c.Send("Filter saved!\nTrigger: " + trigger + "\nType: " + kind)
}

func (m *FiltersModule) handleStop(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	if !m.Bot.CheckAdmin(c, target, c.Sender(), "can_change_info") {
		return c.Send("You must be an admin to use this command.")
	}
	if !m.Bot.CheckBotAdmin(c, target, "can_change_info") {
		return nil
	}
	args := c.Args
	if len(args) < 1 {
		return c.Send("Usage: /stop <trigger>")
	}

	trigger := strings.ToLower(args[0])

	err = m.Store.DeleteFilter(target.ID, trigger)
	if err != nil {
		return c.Send("Failed to delete filter: " + err.Error())
	}

	m.Cache.Lock()
	delete(m.Cache.Filters, target.ID)
	m.Cache.Unlock()

	m.Logger.Log(target.ID, "other", "Filter deleted by "+c.Sender().FirstName+"\nTrigger: "+trigger)

	return c.Send("Filter '" + trigger + "' deleted.")
}

func (m *FiltersModule) handleFilters(c *bot.Context) error {
	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return c.Send("Error resolving chat.")
	}
	filters, err := m.Store.GetFilters(target.ID)
	if err != nil {
		return c.Send("Failed to fetch filters: " + err.Error())
	}

	if len(filters) == 0 {
		return c.Send("No filters active in this chat.")
	}

	msg := "<b>Active Filters:</b>\n"
	for _, f := range filters {
		msg += "• <code>" + html.EscapeString(f.Trigger) + "</code>\n"
	}

	return c.Send(msg, "HTML")
}

func (m *FiltersModule) handleText(c *bot.Context) error {
	text := c.Text()
	if strings.HasPrefix(text, "/") {
		return nil
	}

	target, err := m.Bot.GetTargetChat(c)
	if err != nil {
		return nil
	}

	m.Cache.RLock()
	filters, ok := m.Cache.Filters[target.ID]
	m.Cache.RUnlock()

	if !ok {
		var err error
		filters, err = m.Store.GetFilters(target.ID)
		if err != nil {
			return nil
		}
		m.Cache.Lock()
		m.Cache.Filters[target.ID] = filters
		m.Cache.Unlock()
	}

	lowerText := strings.ToLower(text)

	for _, f := range filters {
		if strings.Contains(lowerText, strings.ToLower(f.Trigger)) {
			switch f.Type {
			case "sticker":
				return m.Bot.Raw("sendSticker", map[string]any{"chat_id": c.Chat().ID, "sticker": f.Response})
			case "photo":
				return m.Bot.Raw("sendPhoto", map[string]any{"chat_id": c.Chat().ID, "photo": f.Response})
			case "video":
				return m.Bot.Raw("sendVideo", map[string]any{"chat_id": c.Chat().ID, "video": f.Response})
			case "voice":
				return m.Bot.Raw("sendVoice", map[string]any{"chat_id": c.Chat().ID, "voice": f.Response})
			case "audio":
				return m.Bot.Raw("sendAudio", map[string]any{"chat_id": c.Chat().ID, "audio": f.Response})
			case "document":
				return m.Bot.Raw("sendDocument", map[string]any{"chat_id": c.Chat().ID, "document": f.Response})
			case "video_note":
				return m.Bot.Raw("sendVideoNote", map[string]any{"chat_id": c.Chat().ID, "video_note": f.Response})
			case "animation":
				return m.Bot.Raw("sendAnimation", map[string]any{"chat_id": c.Chat().ID, "animation": f.Response})
			default:
				return c.Send(f.Response, "Markdown")
			}
		}
	}

	return nil
}
