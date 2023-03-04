package modules

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/Akvanvig/roboto-go/internal/bot/lib/commands"
	"github.com/Akvanvig/roboto-go/internal/util"
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
	input := strings.SplitN(event.Options[0].StringValue(), " ", 2)
	arg := ""

	commandToRun := input[0]
	commandToRun = strings.Trim(commandToRun, " ")
	commandToRun = strings.ToLower(commandToRun)

	if len(input) > 1 {
		arg = input[1]
	}

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
	case "announce":
		if arg == "" {
			event.RespondMsg("Can't send an empty announcement", discordgo.MessageFlagsEphemeral)
			break
		}

		sender := event.Data.User
		if sender == nil {
			sender = event.Data.Member.User
		}

		app, _ := event.Session.Application("@me")
		members := app.Team.Members

		for i := 0; i < len(members); i++ {
			user := members[i].User
			dm, err := event.Session.UserChannelCreate(user.ID)

			if err == nil {
				event.Session.ChannelMessageSend(dm.ID, fmt.Sprintf("%s - %s", arg, sender.Mention()))
			}
		}

		event.RespondMsg("Announcement sent", discordgo.MessageFlagsEphemeral)
	case "shutdown":
		event.RespondMsg("Shutting down", discordgo.MessageFlagsEphemeral)
		util.SendOSInterruptSignal()
	case "help":
		fallthrough
	default:
		var builder strings.Builder

		builder.WriteString("- **info**: Display information about the application\n")
		builder.WriteString("- **announce**: Announce a message to all team members\n")
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
