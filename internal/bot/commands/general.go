package commands

import (
	"os"
	"path/filepath"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func init() {
	createChatCommands([]Command{
		{
			Name:        "catjam",
			Description: "Let's jam!",
			Handler:     onCatJam,
		},
		{
			Name:               "gamewithme",
			Description:        "Let's play a game",
			Handler:            onGameWithMe,
			HandlerModalSubmit: onGameWithMeSubmit,
		},
	})

	createUserContextCommands([]Command{
		{
			Name: "Play a game",
			// We can reuse handlers!!
			Handler:            onGameWithMe,
			HandlerModalSubmit: onGameWithMeSubmit,
		},
	})
}

func onCatJam(cmd *Command, event *Event) {
	file, err := os.Open(filepath.Join(globals.RootPath, "assets/img/catjam.gif"))

	if err != nil {
		log.Error().Err(err).Send()
		event.RespondMsg("Failed to open catjam asset")
		return
	}

	defer file.Close()

	event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Files: []*discordgo.File{
				{
					ContentType: "image/gif",
					Name:        "catjam.gif",
					Reader:      file,
				},
			},
		},
	})
}

// TODO(Fredrico):
// This is unfinished
func onGameWithMe(cmd *Command, event *Event) {
	event.Respond(&Response{
		Type: ResponseModal,
		Data: &ResponseData{
			// Note(Fredrico):
			// The parameter to GenerateModalID is optional.
			// ToDo(Fredrico):
			// Contemplate auto generating this ID somewhere else based on runtime caller data maybe?
			// As it is, this isn't intuitive anyhow. Gosh the API for this is shit.
			CustomID: cmd.GenerateModalID(event.Data.Interaction.Member.User.ID),
			Title:    "A Game",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "opinion",
							Label:       "Why do you want to play the game?",
							Style:       discordgo.TextInputShort,
							Placeholder: "Don't be shy, tell me",
							Required:    true,
							MaxLength:   300,
							MinLength:   10,
						},
					},
				},
			},
		},
	})
}

func onGameWithMeSubmit(cmd *Command, event *Event, identifier string) {
	event.RespondMsg("Thank you for playing! Here's your doxed user ID: " + identifier)
}
