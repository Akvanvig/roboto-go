package command

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
	"github.com/mroctopus/bottie-bot/internal/bot"
)

// -- COMMANDS --

type CommandBootstrapper func(*bot.RobotoBot, *handler.Mux) discord.ApplicationCommandCreate

// Add more bootstappers here
var bootstrappers = [...]CommandBootstrapper{
	adminCommands,
	musicCommands,
}

func New(bot *bot.RobotoBot) ([]discord.ApplicationCommandCreate, *handler.Mux) {
	r := handler.New()
	r.Use(middleware.Go)

	cmds := make([]discord.ApplicationCommandCreate, 0, len(bootstrappers))
	for _, bootstrap := range bootstrappers {
		cmd := bootstrap(bot, r)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return cmds, r
}

// -- COMMON --

type MessageType int

const (
	MessageTypeDefault MessageType = iota
	MessageTypeError
)

func message[T *discord.MessageCreate | *discord.MessageUpdate](dst T, txt string, t MessageType, flags discord.MessageFlags) T {
	var color int
	switch t {
	case MessageTypeError:
		color = 0
		txt = fmt.Sprintf("Error: %s", txt)
	case MessageTypeDefault:
		fallthrough
	default:
		color = 0
	}

	embeds := []discord.Embed{
		{
			Description: txt,
			Color:       color,
		},
	}

	// NOTE:
	// For the love of god, please let this proposal go through:
	// https://github.com/golang/go/issues/45380
	switch v := any(dst).(type) {
	case *discord.MessageCreate:
		v.Embeds = embeds
		v.Flags = flags
	case *discord.MessageUpdate:
		v.Embeds = &embeds
		if flags > 0 {
			v.Flags = &flags
		}
		v.Components = nil
	}

	return dst
}
