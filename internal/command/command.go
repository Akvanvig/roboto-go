package command

import (
	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
)

// -- COMMANDS --

type CommandBootstrapper func(*bot.RobotoBot, *handler.Mux) discord.ApplicationCommandCreate

// Add more bootstappers here
var bootstrappers = [...]CommandBootstrapper{
	ownerCommands,
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

const (
	MessageColorDefault = 0x00A8FC
	MessageColorError   = 0xD43535
)

func Embeds(text string, color int) []discord.Embed {
	return []discord.Embed{
		{
			Description: text,
			Color:       color,
		},
	}
}
