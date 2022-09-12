package commands

import (
	"errors"
	"os"
	"path"

	. "github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/bwmarrin/discordgo"
)

func onCatJam(cmd *Command, event *Event) {
	file, err := os.Open(path.Join(RootPath, "assets/img/catjam.gif"))

	if err != nil {
		event.RespondError(errors.New("Failed to open catjam asset"))
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
func onLetsPlay(cmd *Command, event *Event) {
	event.Respond(&Response{
		Type: ResponseModal,
		Data: &ResponseData{
			// Note(Fredrico):
			// The parameter to GenerateModalID is optional
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

func onLetsPlaySubmit(cmd *Command, event *Event, identifier string) {
	event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: "Thank you for playing! Here's your doxed user ID: " + identifier,
		},
	})
}

func init() {
	generalCommands := CommandMap{
		"catjam": &Command{
			State: CommandInfo{
				Name:        "catjam",
				Description: "Let's jam!",
			},
			Handler: onCatJam,
		},
		"letsplay": &Command{
			State: CommandInfo{
				Name:        "letsplay",
				Description: "Let's play a game",
			},
			Handler:      onLetsPlay,
			HandlerModal: onLetsPlaySubmit,
		},
	}

	addCommands(generalCommands)
}
