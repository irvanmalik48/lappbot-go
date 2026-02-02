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
	helpText := `
<b>General</b>
/ping - Check bot latency
/version - Check bot version
/help - Show this message
/report - Call admins (Reply)

<b>Group Settings</b>
/welcome &lt;on|off&gt; [msg] - Configure greetings
/captcha &lt;on|off&gt; - Configure CAPTCHA
/filter &lt;trigger&gt; &lt;response&gt; - Save filter
/stop &lt;trigger&gt; - Delete filter
/filters - List filters

<b>Moderation</b>
/warn [reason] - Warn a user (Reply)
/kick [reason] - Kick a user (Reply)
/ban [reason] - Ban a user (Reply)
/tban &lt;time&gt; [reason] - Timed ban (Reply)
/mute [reason] - Mute a user (Reply)
/tmute &lt;time&gt; [reason] - Timed mute (Reply)
/purge - Purge messages (Reply)
/pin - Pin a message (Reply)

<b>Blacklist</b>
/bl &lt;type&gt; &lt;value&gt; [action] [time] - Add to blacklist
/unbl &lt;type&gt; &lt;value&gt; - Remove from blacklist
/blacklist - List blacklist items

<b>Admin & Approval</b>
/promote [title] - Promote to admin
/demote - Demote to member
/approve - Exempt user from blacklist
/unapprove - Revoke exemption

<b>Silent Actions</b>
/skick, /sban, /smute - Silent variants

<b>Realm Actions</b>
/rban [reason], /rmute [reason] - Global ban/mute
`
	return c.Send(helpText, tele.ModeHTML)
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
