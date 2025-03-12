package command

import (
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
	}

	_ = &OwnerHandler{}
	r.Route("/owner", func(r handler.Router) {
		r.Use(func(next handler.Handler) handler.Handler {
			return func(e *handler.InteractionEvent) error {
				app, err := e.Client().Rest().GetBotApplicationInfo()
				if err != nil {
					return e.Respond(discord.InteractionResponseTypeCreateMessage, *message(&discord.MessageUpdate{}, "Failed to retrieve bot app info", MessageTypeError, 0))
				}

				user := e.User()
				members := app.Team.Members
				for i := range app.Team.Members {
					member := members[i]

					if user.ID == member.User.ID {
						next(e)
					}
				}

				return e.Respond(discord.InteractionResponseTypeCreateMessage, *message(&discord.MessageUpdate{}, "Only bot owners can run this command", MessageTypeError, 0))
			}
		})
	})

	return cmds
}

// -- HANDLERS --

type OwnerHandler struct {
}
