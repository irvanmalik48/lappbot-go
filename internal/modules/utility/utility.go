package utility

import (
	"fmt"
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"time"

	tele "gopkg.in/telebot.v3"
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
	m.Bot.Bot.Handle("/report", m.handleReport)
}

func (m *Module) handleStart(c tele.Context) error {
	return c.Send(fmt.Sprintf("Hello! I am %s. Use /help to see what I can do.", m.Cfg.BotName))
}

func (m *Module) handlePing(c tele.Context) error {
	latency := time.Since(c.Message().Time())
	return c.Send(fmt.Sprintf("Pong! Latency: %v", latency))
}

func (m *Module) handleVersion(c tele.Context) error {
	return c.Send(fmt.Sprintf("%s v%s", m.Cfg.BotName, m.Cfg.BotVersion))
}

func (m *Module) handleHelp(c tele.Context) error {
	helpText := `**Available Commands:**

**Moderation:**
/kick - Kick a user (Reply)
/ban [reason] - Ban a user (Reply)
/tban <duration> [reason] - Temporarily ban a user (Reply)
/mute [reason] - Mute a user (Reply)
/tmute <duration> [reason] - Temporarily mute a user (Reply)
/warn [reason] - Warn a user (Reply)
/unwarn - Remove the last warning from a user (Reply)
/resetwarns - Reset all warnings for a user (Reply)
/warns - Check your warnings
/approve - Approve a user (Reply)
/unapprove - Unapprove a user (Reply)
/promote [title] - Promote a user to admin (Reply)
/demote - Demote an admin to member (Reply)
/purge <count> - Delete messages
/pin - Pin a message (Reply)

**Filters:**
/filter <trigger> <response> - Add a filter
/stop <trigger> - Remove a filter
/filters - List all filters

**Configuration:**
/welcome <on|off> [message] - Set welcome message
/goodbye <on|off> [message] - Set goodbye message
/captcha <on|off> - Enable/Disable CAPTCHA

**Placeholders for Welcome/Goodbye:**
{firstname} - User's first name (Link)
{username} - User's username
{userid} - User's ID
`
	return c.Send(helpText, tele.ModeMarkdown)
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
