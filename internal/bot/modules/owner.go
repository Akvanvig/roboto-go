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
		user := event.Data.User
		if user == nil {
			user = event.Data.Member.User
		}

		// Check if calling user is a owner
		for _, member := range fetchAppTeam(event.Session).Members {
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

func fetchAppTeam(session *discordgo.Session) *discordgo.Team {
	app, _ := session.Application("@me")
	return app.Team
}

func onOwnerRunCommand(event *Event) {
	commandToRun := event.Options[0].StringValue()
	commandToRun = strings.ToLower(commandToRun)

	switch commandToRun {
	case "team":
		var builder strings.Builder
		team := fetchAppTeam(event.Session)

		builder.WriteString("Team Info:\n```Name: ")
		builder.WriteString(team.Name)
		if team.Description != "" {
			builder.WriteString("\nDescription: ")
			builder.WriteString(team.Description)
		}
		builder.WriteString("\nMembers: ")
		for i := 0; i < len(team.Members); i++ {
			user := team.Members[i].User
			builder.WriteString("\n- ")
			builder.WriteString(user.Username)
			builder.WriteString("#")
			builder.WriteString(user.Discriminator)
		}
		builder.WriteString("```")

		event.RespondMsg(builder.String(), discordgo.MessageFlagsEphemeral)

	case "shutdown":
		event.RespondMsg("Shutting down", discordgo.MessageFlagsEphemeral)
		os.Exit(0)
	case "help":
		fallthrough
	default:
		var builder strings.Builder

		builder.WriteString("Valid owner commands:\n```")
		builder.WriteString("\n- team: Display the application's team info")
		builder.WriteString("\n- shutdown: Shutdown the application")
		builder.WriteString("\n- help: Display the help menu")
		builder.WriteString("```")

		event.RespondMsg(builder.String(), discordgo.MessageFlagsEphemeral)
	}
}
