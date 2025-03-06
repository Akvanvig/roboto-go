package command

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
	"github.com/disgoorg/json"
	"github.com/mroctopus/bottie-bot/internal/bot"
)

// -- COMMANDS --

type CommandBootstrapper func(*bot.RobotoBot, *handler.Mux) discord.ApplicationCommandCreate

// Add more bootstappers here
var bootstrappers = [...]CommandBootstrapper{
	music,
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

/*
const (
	MessageColorPrimary =
)
*/

func message(txt string) discord.MessageCreate {
	return discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				Description: txt,
				Color:       0,
			},
		},
	}
}

func messageUpdate(txt string) discord.MessageUpdate {
	return discord.MessageUpdate{
		Embeds: &[]discord.Embed{
			{
				Description: txt,
				Color:       0,
			},
		},
	}
}

func errorMessage(err error) discord.MessageCreate {
	return discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				Description: fmt.Sprintf("Error: %s", err.Error()),
				Color:       0,
			},
		},
		Flags: discord.MessageFlagEphemeral,
	}
}

func errorMessageUpdate(err error) discord.MessageUpdate {
	return discord.MessageUpdate{
		Embeds: &[]discord.Embed{
			{
				Description: fmt.Sprintf("Error: %s", err.Error()),
				Color:       0,
			},
		},
		Flags: json.Ptr(discord.MessageFlagEphemeral),
	}
}
