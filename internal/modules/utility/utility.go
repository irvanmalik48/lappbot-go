package utility

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Module struct {
	Bot *bot.Bot
	Cfg *config.Config
}

func New(b *bot.Bot, cfg *config.Config) *Module {
	return &Module{Bot: b, Cfg: cfg}
}

func (m *Module) Register() {
	m.Bot.Handle("/ping", m.handlePing)
	m.Bot.Handle("/start", m.handleStart)
	m.Bot.Handle("/version", m.handleVersion)
	m.Bot.Handle("/help", m.handleHelp)
	m.Bot.Handle("help_main", m.onHelpCallback)
	m.Bot.Handle("help_mod", m.onHelpCallback)
	m.Bot.Handle("help_settings", m.onHelpCallback)
	m.Bot.Handle("help_filters", m.onHelpCallback)
	m.Bot.Handle("help_admin", m.onHelpCallback)
	m.Bot.Handle("help_antispam", m.onHelpCallback)
	m.Bot.Handle("help_warns", m.onHelpCallback)
	m.Bot.Handle("help_realm", m.onHelpCallback)
	m.Bot.Handle("help_purges", m.onHelpCallback)
	m.Bot.Handle("help_notes", m.onHelpCallback)
	m.Bot.Handle("help_conn", m.onHelpCallback)
	m.Bot.Handle("help_topics", m.onHelpCallback)
	m.Bot.Handle("help_cursed", m.onHelpCallback)
	m.Bot.Handle("/report", m.handleReport)
}

func (m *Module) handleStart(c *bot.Context) error {
	return c.Send(fmt.Sprintf("Hello! I am %s. Use /help to see what I can do.", m.Cfg.BotName))
}

func (m *Module) handlePing(c *bot.Context) error {
	latency := time.Since(time.Unix(c.Message.Date, 0)).Round(time.Millisecond)

	storePings, err := m.Bot.Store.Ping()
	if err != nil {
		return c.Send(fmt.Sprintf("Pong! Latency: %v\nError checking store: %v", latency, err))
	}

	uptime := time.Since(m.Bot.StartTime).Round(time.Second)
	uptimeStr := strings.ReplaceAll(uptime.String(), "h", "h ")
	uptimeStr = strings.ReplaceAll(uptimeStr, "m", "m ")

	msg := fmt.Sprintf("**PONG!**\n\nBot: `%v`\nDatabase: `%v`\nValkey: `%v`\n\nUptime: `%v`",
		latency, storePings["database"].Round(time.Millisecond), storePings["valkey"].Round(time.Millisecond), uptimeStr)

	return c.Send(msg, "Markdown")
}

func (m *Module) handleVersion(c *bot.Context) error {
	return c.Send(fmt.Sprintf("**%s**\nVersion: v%s\nGo: %s\nOS: %s/%s", m.Cfg.BotName, m.Cfg.BotVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH), "Markdown")
}

func (m *Module) handleHelp(c *bot.Context) error {
	text, markup := m.getHelpMenu("main")
	return c.Send(text, markup, "Markdown")
}

func (m *Module) onHelpCallback(c *bot.Context) error {
	c.Respond()
	data := strings.TrimPrefix(c.Data(), "help_")
	text, markup := m.getHelpMenu(data)
	return c.Edit(text, markup, "Markdown")
}

func (m *Module) getHelpMenu(section string) (string, *bot.ReplyMarkup) {
	markup := &bot.ReplyMarkup{}

	var backBtn bot.InlineKeyboardButton
	backBtn.Text = "« Back"
	backBtn.CallbackData = "help_main"

	var text string
	switch section {
	case "main":
		markup.InlineKeyboard = [][]bot.InlineKeyboardButton{
			{{Text: "Moderation", CallbackData: "help_mod"}, {Text: "Settings", CallbackData: "help_settings"}},
			{{Text: "Filters", CallbackData: "help_filters"}, {Text: "Warnings", CallbackData: "help_warns"}, {Text: "Admin", CallbackData: "help_admin"}},
			{{Text: "Realm", CallbackData: "help_realm"}, {Text: "Anti-Spam", CallbackData: "help_antispam"}, {Text: "Purges", CallbackData: "help_purges"}},
			{{Text: "Notes", CallbackData: "help_notes"}, {Text: "Connection", CallbackData: "help_conn"}},
			{{Text: "Topics", CallbackData: "help_topics"}, {Text: "Cursed", CallbackData: "help_cursed"}},
		}
		return "Welcome to Lappbot Help.\nSelect a category:", markup

	case "cursed":
		text = `**Cursed Commands:**
/zalgo <text> - Zalgo text
/uwuify <text> - UwU text
/emojify <text> - Emojify text
/leetify <text> - Leetify text`

	case "topics":
		text = `**Topic Commands:**
/actiontopic - Get action topic
/setactiontopic - Set action topic
/newtopic <name> - Create topic
/renametopic <name> - Rename topic
/closetopic - Close topic
/reopentopic - Reopen topic
/deletetopic - Delete topic`

	case "notes":
		text = `**Notes Commands:**
/get <notename> - Get note
#<notename> - Get note
/save <notename> <content> - Save note
/clear <notename> - Delete note
/notes - List notes
/clearall - Delete all notes
/privatenotes - Toggle private mode`

	case "conn":
		text = `**Connection Commands:**
/connect <chat> - Connect to Chat
/disconnect - Disconnect
/reconnect - Reconnect
/connection - Check Connection`

	case "mod":
		text = `**Moderation Commands:**
/kick - Kick (Reply)
/ban [reason] - Ban (Reply)
/tban <duration> [reason] - Timed Ban (Reply)
/mute [reason] - Mute (Reply)
/tmute <duration> [reason] - Timed Mute (Reply)
/skick - Silent Kick (Reply)
/sban - Silent Ban (Reply)
/smute - Silent Mute (Reply)
/unban - Unban (Reply)
/unmute - Unmute (Reply)
/pin - Pin (Reply)
/lock - Lock Group
/unlock - Unlock Group`

	case "purges":
		text = `**Purge Commands:**
/purge [count] - Purge messages
/spurge [count] - Silent purge
/del - Delete message
/purgefrom - Mark start
/purgeto - Purge range`

	case "warns":
		text = `**Warning Commands:**
/warn [reason] - Warn (Reply)
/dwarn [reason] - Warn & Delete
/swarn [reason] - Silent Warn
/rmwarn - Remove Last Warn (Reply)
/resetwarn - Reset Warns (Reply)
/resetallwarns - Reset Chat Warns
/warns - Check Warns
/warnings - Check Settings
/warnlimit <number> - Set Limit
/warnmode <action> - Set Action
/warntime <duration> - Set Duration`

	case "antispam":
		text = `**Anti-Raid:**
/antiraid <time/off> - Toggle
/raidtime <time> - Set duration
/raidactiontime <time> - Ban duration
/autoantiraid <count> - Auto-enable

**Anti-Flood:**
/flood - Settings
/setflood <count> - Consecutive limit
/setfloodtimer <count> <time> - Timed limit
/floodmode <action> [time] - Action
/clearflood <yes/no> - Delete flood`
	case "settings":
		text = `**Group Settings:**
/welcome <on|off|text> [msg] - Welcome Msg
/goodbye <on|off|text> [msg] - Goodbye Msg
/captcha <on|off> - CAPTCHA

**Placeholders:**
{firstname}, {username}, {userid}`
	case "filters":
		text = `**Filter Commands:**
/filter <trigger> <response> - Add filter (reply)
/stop <trigger> - Remove filter
/filters - List filters`
	case "admin":
		text = `**Admin Commands:**
/promote [title] - Promote
/demote - Demote
/approve - Exempt User
/unapprove - Revoke Exemption
/bl <type> <value> [action] - Blacklist
/unbl <type> <value> - Unblacklist
/blacklist - List Rules`
	case "realm":
		text = `**Realm Commands:**
/rban [reason] - Realm Ban (Reply)
/rmute [reason] - Realm Mute (Reply)
(Bot Owner Only)`
	}

	markup.InlineKeyboard = [][]bot.InlineKeyboardButton{
		{backBtn},
	}
	return text, markup
}

func (m *Module) handleReport(c *bot.Context) error {
	if c.Message.ReplyTo == nil {
		return c.Send("Reply to a user's message to report it.")
	}

	reason := c.Args
	reasonStr := "No reason"
	if len(reason) > 0 {
		reasonStr = ""
		for _, s := range reason {
			reasonStr += s + " "
		}
	}

	reportedUser := c.Message.ReplyTo.From
	reporter := c.Sender()

	var sb strings.Builder
	sb.WriteString("REPORT!\n")
	sb.WriteString("Group: ")
	sb.WriteString(c.Chat().Title)
	sb.WriteString("\n")
	sb.WriteString("Reporter: [")
	sb.WriteString(reporter.FirstName)
	sb.WriteString("](tg://user?id=")
	sb.WriteString(strconv.FormatInt(reporter.ID, 10))
	sb.WriteString(")\n")
	sb.WriteString("Reported: [")
	sb.WriteString(reportedUser.FirstName)
	sb.WriteString("](tg://user?id=")
	sb.WriteString(strconv.FormatInt(reportedUser.ID, 10))
	sb.WriteString(")\n")
	sb.WriteString("Reason: ")
	sb.WriteString(reasonStr)

	reportMsg := sb.String()

	targetID := m.Cfg.ReportChannelID
	if targetID == 0 {
		targetID = m.Cfg.BotOwnerID
	}

	if targetID != 0 {
		payload := map[string]any{
			"chat_id":    targetID,
			"text":       reportMsg,
			"parse_mode": "Markdown",
		}
		m.Bot.Raw("sendMessage", payload)
	}
	return c.Send("Report sent to admins.")
}

func ReplacePlaceholders(msg string, user *bot.User) string {
	userIDStr := strconv.FormatInt(user.ID, 10)
	firstNameLink := "[" + user.FirstName + "](tg://user?id=" + userIDStr + ")"

	r := strings.NewReplacer(
		"{firstname}", firstNameLink,
		"{username}", user.Username,
		"{userid}", userIDStr,
	)
	return r.Replace(msg)
}
