package utility

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"runtime"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

type Module struct {
	Bot *bot.Bot
	Cfg *config.Config
}

func New(b *bot.Bot, cfg *config.Config) *Module {
	return &Module{Bot: b, Cfg: cfg}
}

func (m *Module) Register() {
	m.Bot.Bot.Handle("/ping", m.handlePing)
	m.Bot.Bot.Handle("/start", m.handleStart)
	m.Bot.Bot.Handle("/version", m.handleVersion)
	m.Bot.Bot.Handle("/help", m.handleHelp)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_main"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_mod"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_settings"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_filters"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_admin"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_antispam"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_warns"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_realm"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_purges"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_notes"}, m.onHelpCallback)
	m.Bot.Bot.Handle(&tele.Btn{Unique: "help_conn"}, m.onHelpCallback)
	m.Bot.Bot.Handle("/report", m.handleReport)
}

func (m *Module) handleStart(c tele.Context) error {
	return c.Send(fmt.Sprintf("Hello! I am %s. Use /help to see what I can do.", m.Cfg.BotName))
}

func (m *Module) handlePing(c tele.Context) error {
	latency := time.Since(c.Message().Time()).Round(time.Millisecond)
	storePings, err := m.Bot.Store.Ping()
	if err != nil {
		return c.Send(fmt.Sprintf("Pong! Latency: %v\nError checking store: %v", latency, err))
	}

	uptime := time.Since(m.Bot.StartTime).Round(time.Second)

	msg := fmt.Sprintf("**PONG!**\n\nBot: `%v`\nDatabase: `%v`\nValkey: `%v`\n\nUptime: `%v`",
		latency, storePings["database"].Round(time.Millisecond), storePings["valkey"].Round(time.Millisecond), uptime)

	return c.Send(msg, tele.ModeMarkdown)
}

func (m *Module) handleVersion(c tele.Context) error {
	return c.Send(fmt.Sprintf("**%s**\nVersion: v%s\nGo: %s\nOS: %s/%s", m.Cfg.BotName, m.Cfg.BotVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH), tele.ModeMarkdown)
}

func (m *Module) handleHelp(c tele.Context) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Moderation", "help_mod", "mod"),
			markup.Data("Settings", "help_settings", "settings"),
		),
		markup.Row(
			markup.Data("Filters", "help_filters", "filters"),
			markup.Data("Warnings", "help_warns", "warns"),
			markup.Data("Admin", "help_admin", "admin"),
		),
		markup.Row(
			markup.Data("Realm", "help_realm", "realm"),
			markup.Data("Anti-Spam", "help_antispam", "antispam"),
		),
	)

	return c.Send("Welcome to Lappbot Help.\nSelect a category:", markup)
}

func (m *Module) onHelpCallback(c tele.Context) error {
	defer c.Respond()
	section := c.Data()
	markup := &tele.ReplyMarkup{}
	backBtn := markup.Data("« Back", "help_main", "main")

	var text string

	switch section {
	case "main":
		markup.Inline(
			markup.Row(
				markup.Data("Moderation", "help_mod", "mod"),
				markup.Data("Settings", "help_settings", "settings"),
			),
			markup.Row(
				markup.Data("Filters", "help_filters", "filters"),
				markup.Data("Warnings", "help_warns", "warns"),
				markup.Data("Admin", "help_admin", "admin"),
			),
			markup.Row(
				markup.Data("Realm", "help_realm", "realm"),
				markup.Data("Anti-Spam", "help_antispam", "antispam"),
				markup.Data("Purges", "help_purges", "purges"),
			),
			markup.Row(
				markup.Data("Notes", "help_notes", "notes"),
				markup.Data("Connection", "help_conn", "conn"),
			),
		)
		return c.Edit("Welcome to Lappbot Help.\nSelect a category:", markup)

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

	markup.Inline(markup.Row(backBtn))
	return c.Edit(text, markup, tele.ModeMarkdown)
}

func (m *Module) handleReport(c tele.Context) error {
	if !c.Message().IsReply() {
		return c.Send("Reply to a user's message to report it.")
	}

	reason := c.Args()
	reasonStr := "No reason"
	if len(reason) > 0 {
		reasonStr = ""
		for _, s := range reason {
			reasonStr += s + " "
		}
	}

	reportedUser := c.Message().ReplyTo.Sender
	reporter := c.Sender()

	reportMsg := fmt.Sprintf("REPORT!\nGroup: %s\nReporter: [%s](tg://user?id=%d)\nReported: [%s](tg://user?id=%d)\nReason: %s",
		c.Chat().Title, reporter.FirstName, reporter.ID, reportedUser.FirstName, reportedUser.ID, reasonStr)

	admins, err := c.Bot().AdminsOf(c.Chat())
	if err != nil {
		return c.Send("Failed to fetch admins.")
	}

	sentCount := 0
	for _, admin := range admins {
		if !admin.User.IsBot {
			_, err := c.Bot().Send(admin.User, reportMsg)
			if err == nil {
				sentCount++
			}
		}
	}

	if sentCount == 0 {
		return c.Send("Report failed (Admins haven't started bot).")
	}
	return c.Send("Report sent to admins.")
}

func ReplacePlaceholders(msg string, user *tele.User) string {
	msg = strings.ReplaceAll(msg, "{firstname}", fmt.Sprintf("[%s](tg://user?id=%d)", user.FirstName, user.ID))
	msg = strings.ReplaceAll(msg, "{username}", user.Username)
	msg = strings.ReplaceAll(msg, "{userid}", fmt.Sprintf("%d", user.ID))
	return msg
}
