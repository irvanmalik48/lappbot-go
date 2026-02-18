package cursed

import (
	"lappbot/internal/bot"
	"lappbot/internal/config"
	"lappbot/internal/modules/logging"
	"math/rand"
	"strings"
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
	m.Bot.Handle("/zalgo", m.handleZalgo)
	m.Bot.Handle("/uwuify", m.handleUwuify)
	m.Bot.Handle("/emojify", m.handleEmojify)
	m.Bot.Handle("/leetify", m.handleLeetify)
}

func (m *Module) handleZalgo(c *bot.Context) error {
	input := getInput(c)
	if input == "" {
		return c.Send("Usage: /zalgo <text> (or reply to a message)")
	}
	return c.Send(zalgo(input))
}

func (m *Module) handleUwuify(c *bot.Context) error {
	input := getInput(c)
	if input == "" {
		return c.Send("Usage: /uwuify <text> (or reply to a message)")
	}
	return c.Send(uwuify(input))
}

func (m *Module) handleEmojify(c *bot.Context) error {
	input := getInput(c)
	if input == "" {
		return c.Send("Usage: /emojify <text> (or reply to a message)")
	}
	return c.Send(emojify(input))
}

func (m *Module) handleLeetify(c *bot.Context) error {
	input := getInput(c)
	if input == "" {
		return c.Send("Usage: /leetify <text> (or reply to a message)")
	}
	return c.Send(leetify(input))
}

func getInput(c *bot.Context) string {
	if len(c.Args) > 0 {
		return strings.Join(c.Args, " ")
	}

	if c.Message.ReplyTo != nil {
		if c.Message.ReplyTo.Text != "" {
			return c.Message.ReplyTo.Text
		}
		if c.Message.ReplyTo.Caption != "" {
			return c.Message.ReplyTo.Caption
		}
	}

	return ""
}

func zalgo(text string) string {
	var result strings.Builder
	for _, char := range text {
		result.WriteRune(char)
		numUp := rand.Intn(8)
		numMid := rand.Intn(2)
		numDown := rand.Intn(8)

		for i := 0; i < numUp; i++ {
			result.WriteRune(rune(0x0300 + rand.Intn(0x036F-0x0300)))
		}
		for i := 0; i < numMid; i++ {
			result.WriteRune(rune(0xFE20 + rand.Intn(0xFE2F-0xFE20)))
		}
		for i := 0; i < numDown; i++ {
			result.WriteRune(rune(0x0300 + rand.Intn(0x036F-0x0300)))
		}
	}
	return result.String()
}

func uwuify(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "r", "w")
	text = strings.ReplaceAll(text, "l", "w")
	text = strings.ReplaceAll(text, "m", "mw")
	text = strings.ReplaceAll(text, "ove", "uv")

	faces := []string{"(・`ω´・)", ";;w;;", "OwO", "UwU", ">w<", "^w^", "(˶>⩊<˶)", "(,,>﹏<,,)👉👈"}

	var result strings.Builder
	for _, char := range text {
		result.WriteRune(char)
		if char == ' ' && rand.Intn(10) == 0 {
			result.WriteString(" " + faces[rand.Intn(len(faces))] + " ")
		}
	}

	if rand.Intn(2) == 0 {
		result.WriteString(" " + faces[rand.Intn(len(faces))])
	}

	return result.String()
}

func emojify(text string) string {
	var result strings.Builder
	for _, char := range strings.ToLower(text) {
		if char >= 'a' && char <= 'z' {
			result.WriteString(string(rune(0x1F1E6 + int(char-'a'))))
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

func leetify(text string) string {
	replacements := map[rune]string{
		'a': "4", 'e': "3", 'g': "6", 'i': "1",
		'o': "0", 's': "5", 't': "7",
	}

	var result strings.Builder
	for _, char := range strings.ToLower(text) {
		if val, Ok := replacements[char]; Ok {
			result.WriteString(val)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}
