package utility

import (
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"lappbot/internal/modules/logging"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Module struct {
	Bot    *bot.Bot
	Cfg    *config.Config
	Logger *logging.Module
}

func New(b *bot.Bot, cfg *config.Config, l *logging.Module) *Module {
	return &Module{Bot: b, Cfg: cfg, Logger: l}
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
	m.Bot.Handle("help_clean", m.onHelpCallback)
	m.Bot.Handle("help_logging", m.onHelpCallback)
	m.Bot.Handle("btn_refresh_ping", m.handlePingRefresh)
	m.Bot.Handle("/report", m.handleReport)
}

func (m *Module) handleStart(c *bot.Context) error {
	return c.Send("Hello! I am " + m.Cfg.BotName + ". Use /help to see what I can do.")
}

func (m *Module) handlePing(c *bot.Context) error {
	msgStr, markup := m.buildPingMessage()
	return c.Send(msgStr, markup, "Markdown")
}

func (m *Module) handlePingRefresh(c *bot.Context) error {
	msgStr, markup := m.buildPingMessage()
	c.Respond("Refreshed")
	return c.Edit(msgStr, markup, "Markdown")
}

func (m *Module) buildPingMessage() (string, *bot.ReplyMarkup) {
	start := time.Now()
	_, err := m.Bot.GetMe()
	if err != nil {
		return "Ping failed: " + err.Error(), nil
	}
	rtt := time.Since(start)

	msg := "Ping: `" + strconv.FormatInt(rtt.Milliseconds(), 10) + "ms`"

	markup := &bot.ReplyMarkup{}
	markup.InlineKeyboard = [][]bot.InlineKeyboardButton{
		{{Text: "Refresh", CallbackData: "btn_refresh_ping"}},
	}

	return msg, markup
}

func (m *Module) handleVersion(c *bot.Context) error {
	msg := "**" + m.Cfg.BotName + "**\nVersion: v" + m.Cfg.BotVersion + "\nGo: " + runtime.Version() + "\nOS: " + runtime.GOOS + "/" + runtime.GOARCH
	return c.Send(msg, "Markdown")
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

var helpCache = map[string]struct {
	Text   string
	Markup *bot.ReplyMarkup
}{
	"main": {
		Text: "Welcome to Lappbot Help.\nSelect a category:",
		Markup: &bot.ReplyMarkup{
			InlineKeyboard: [][]bot.InlineKeyboardButton{
				{{Text: "Moderation", CallbackData: "help_mod"}, {Text: "Settings", CallbackData: "help_settings"}},
				{{Text: "Filters", CallbackData: "help_filters"}, {Text: "Warnings", CallbackData: "help_warns"}, {Text: "Admin", CallbackData: "help_admin"}},
				{{Text: "Realm", CallbackData: "help_realm"}, {Text: "Anti-Spam", CallbackData: "help_antispam"}, {Text: "Purges", CallbackData: "help_purges"}},
				{{Text: "Notes", CallbackData: "help_notes"}, {Text: "Connection", CallbackData: "help_conn"}, {Text: "Logging", CallbackData: "help_logging"}},
				{{Text: "Topics", CallbackData: "help_topics"}, {Text: "Cursed", CallbackData: "help_cursed"}, {Text: "Clean", CallbackData: "help_clean"}},
			},
		},
	},
	"logging": {
		Text: `**Logging Commands:**
/loggroup - View Log Group
/setlog <group_id> - Set Log Group
/unsetlog - Unset Log Group
/log <category> - Enable Log Category
/nolog <category> - Disable Log Category
/logcategories - List Log Categories

Categories: settings, admin, user, automated, reports, other`,
	},
	"cursed": {
		Text: `**Cursed Commands:**
/zalgo <text> - Zalgo text
/uwuify <text> - UwU text
/emojify <text> - Emojify text
/leetify <text> - Leetify text`,
	},
	"topics": {
		Text: `**Topic Commands:**
/actiontopic - Get action topic
/setactiontopic - Set action topic
/newtopic <name> - Create topic
/renametopic <name> - Rename topic
/closetopic - Close topic
/reopentopic - Reopen topic
/deletetopic - Delete topic`,
	},
	"notes": {
		Text: `**Notes Commands:**
/get <notename> - Get note
#<notename> - Get note
/save <notename> <content> - Save note
/clear <notename> - Delete note
/notes - List notes
/clearall - Delete all notes
/privatenotes - Toggle private mode`,
	},
	"clean": {
		Text: `**Clean Commands:**
/cleancommand <type> - Add type to clean list
/keepcommand <type> - Remove type from clean list
/cleancommandtypes - List available types

Types: settings, admin, user, automated, reports, other, all`,
	},
	"conn": {
		Text: `**Connection Commands:**
/connect <chat> - Connect to Chat
/disconnect - Disconnect
/reconnect - Reconnect
/connection - Check Connection`,
	},
	"mod": {
		Text: `**Moderation Commands:**
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
/unlock - Unlock Group`,
	},
	"purges": {
		Text: `**Purge Commands:**
/purge [count] - Purge messages
/spurge [count] - Silent purge
/del - Delete message
/purgefrom - Mark start
/purgeto - Purge range`,
	},
	"warns": {
		Text: `**Warning Commands:**
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
/warntime <duration> - Set Duration`,
	},
	"antispam": {
		Text: `**Anti-Raid:**
/antiraid <time/off> - Toggle
/raidtime <time> - Set duration
/raidactiontime <time> - Ban duration
/autoantiraid <count> - Auto-enable

**Anti-Flood:**
/flood - Settings
/setflood <count> - Consecutive limit
/setfloodtimer <count> <time> - Timed limit
/floodmode <action> [time] - Action
/clearflood <yes/no> - Delete flood`,
	},
	"settings": {
		Text: `**Group Settings:**
/welcome <on|off|text> [msg] - Welcome Msg
/goodbye <on|off|text> [msg] - Goodbye Msg
/captcha <on|off> - CAPTCHA

**Placeholders:**
{firstname}, {username}, {userid}`,
	},
	"filters": {
		Text: `**Filter Commands:**
/filter <trigger> <response> - Add filter (reply)
/stop <trigger> - Remove filter
/filters - List filters`,
	},
	"admin": {
		Text: `**Admin Commands:**
/promote [title] - Promote
/demote - Demote
/approve - Exempt User
/unapprove - Revoke Exemption
/bl <type> <value> [action] - Blacklist
/unbl <type> <value> - Unblacklist
/blacklist - List Rules`,
	},
	"realm": {
		Text: `**Realm Commands:**
/rban [reason] - Realm Ban (Reply)
/rmute [reason] - Realm Mute (Reply)
(Bot Owner Only)`,
	},
}

var backMarkup = &bot.ReplyMarkup{
	InlineKeyboard: [][]bot.InlineKeyboardButton{
		{{Text: "« Back", CallbackData: "help_main"}},
	},
}

func (m *Module) getHelpMenu(section string) (string, *bot.ReplyMarkup) {
	if data, ok := helpCache[section]; ok {
		if data.Markup != nil {
			return data.Text, data.Markup
		}
		return data.Text, backMarkup
	}
	return "Help section not found.", backMarkup
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

	m.Logger.Log(c.Chat().ID, "reports", "Report filed by "+reporter.FirstName+"\nTriggering user: "+reportedUser.FirstName+"\nReason: "+reasonStr)

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
