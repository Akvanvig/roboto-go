package modules

import (
	"errors"
	"os"
	"strings"

	. "github.com/Akvanvig/roboto-go/internal/bot/lib/commands"
	"github.com/bwmarrin/discordgo"
)

func init() {
	ownerCheck := func(event *Event) error {
		app, _ := event.Session.Application("@me")
		user := event.Data.User
		if user == nil {
			user = event.Data.Member.User
		}

		// Check if calling user is a owner
		for _, member := range app.Team.Members {
			if user.ID == member.User.ID {
				return nil
			}
		}

		return errors.New("You are not an owner...")
	}

	InitChatCommands(nil, []Command{
		{
			Name:        "run",
			Description: "?",
			Options: []CommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "command",
					Description: "?",
					Required:    true,
				},
			},
			Handler: &CommandHandler{
				OnRun:      onOwnerRunCommand,
				OnRunCheck: ownerCheck,
			},
		},
	})
}

func onOwnerRunCommand(event *Event) {
	commandToRun := event.Options[0].StringValue()
	commandToRun = strings.Trim(commandToRun, " ")
	commandToRun = strings.ToLower(commandToRun)

	switch commandToRun {
	case "info":
		app, _ := event.Session.Application("@me")
		team := app.Team
		members := make([]string, len(team.Members))
		for i := 0; i < len(members); i++ {
			members[i] = team.Members[i].User.Mention()
		}

		event.Respond(&Response{
			Type: ResponseMsg,
			Data: &ResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Application Information",
						Description: app.Description,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Team Members",
								Value: strings.Join(members, "\n"),
							},
						},
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

	case "shutdown":
		event.RespondMsg("Shutting down", discordgo.MessageFlagsEphemeral)
		os.Exit(0)
	case "help":
		fallthrough
	default:
		var builder strings.Builder

		builder.WriteString("- **info**: Display information about the application\n")
		builder.WriteString("- **shutdown**: Shutdown the application\n")
		builder.WriteString("- **help**: Display the help menu")

		event.Respond(&Response{
			Type: ResponseMsg,
			Data: &ResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Valid Commands",
						Description: builder.String(),
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
