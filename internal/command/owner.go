package command

import (
	"fmt"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

// -- BOOTSTRAP --

func ownerCommands(bot *bot.RobotoBot, r *handler.Mux) discord.ApplicationCommandCreate {
	cmds := discord.SlashCommandCreate{
		Name:        "owner",
		Description: "Owner specific commands",
		Contexts: []discord.InteractionContextType{
			discord.InteractionContextTypeBotDM,
		},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "run",
				Description: "Run an owner command",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "command",
						Description: "The command to run",
						Required:    true,
					},
				},
			},
		},
	}

	h := &OwnerHandler{}
	r.Route("/owner", func(r handler.Router) {
		r.Use(func(next handler.Handler) handler.Handler {
			return func(e *handler.InteractionEvent) error {
				app, err := e.Client().Rest.GetBotApplicationInfo()
				if err != nil {
					return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
						Embeds: new(Embeds("Failed to retrieve bot app info", MessageColorError)),
						Flags:  new(discord.MessageFlagEphemeral),
					})
				}

				user := e.User()
				members := app.Team.Members
				for i := range app.Team.Members {
					member := members[i]
					if user.ID == member.User.ID {
						return next(e)
					}
				}

				return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
					Embeds: new(Embeds("Only bot owners can run this command", MessageColorError)),
					Flags:  new(discord.MessageFlagEphemeral),
				})
			}
		})

		r.SlashCommand("/run", h.onRun)
	})

	return cmds
}

// -- HANDLERS --

type OwnerHandler struct {
}

func (h *OwnerHandler) onRun(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	command := data.String("command")

	switch command {
	case "debug":
		logger := e.Client().Logger.Handler()
		if debugger, ok := logger.(*bot.DiscordDebugHandler); ok {
			mode := debugger.Toggle()
			return e.CreateMessage(discord.MessageCreate{
				Embeds: Embeds(fmt.Sprintf("Toggled debug mode. Now set to: %t", mode), MessageColorDefault),
				Flags:  discord.MessageFlagEphemeral,
			})
		} else {
			return e.CreateMessage(discord.MessageCreate{
				Embeds: Embeds("Failed to toggle debug mode", MessageColorError),
				Flags:  discord.MessageFlagEphemeral,
			})
		}
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds("Tried to run non-existent command", MessageColorError),
		Flags:  discord.MessageFlagEphemeral,
	})
}
