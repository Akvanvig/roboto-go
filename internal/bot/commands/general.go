package commands

import (
	"os"
	"path/filepath"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func init() {
	createCommands([]Command{
		{
			Name:        "catjam",
			Description: "Let's jam!",
			Handler: &CommandHandler{
				OnRun: onCatJam,
			},
		},
		{
			Name:        "game withme",
			Description: "Let's play a game",
			Handler: &CommandHandler{
				OnRun:         onGameWithMe,
				OnModalSubmit: onGameWithMeSubmit,
			},
		},
	}, CommandContextChat)

	createCommands([]Command{
		{
			Name: "OPEEEN UP",
			Handler: &CommandHandler{
				OnRun:         onGameWithMe,
				OnModalSubmit: onGameWithMeSubmit,
			},
		},
	}, CommandContextUser)

	createCommands([]Command{
		{
			Name: "CHECK THIS OUT",
			Handler: &CommandHandler{
				OnRun:         onGameWithMe,
				OnModalSubmit: onGameWithMeSubmit,
			},
		},
	}, CommandContextMessage)
}

func onCatJam(event *Event) {
	file, err := os.Open(filepath.Join(util.RootPath, "assets/img/catjam.gif"))

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
func onGameWithMe(event *Event) {
	event.RespondModal(event.Command, &ResponseData{
		Title: "A Game",
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
	})
}

func onGameWithMeSubmit(event *Event) {
	event.RespondMsg("Thank you for playing!")
}
