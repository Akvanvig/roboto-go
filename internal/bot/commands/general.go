package commands

import (
	"errors"
	"os"
	"path"

	. "github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/bwmarrin/discordgo"
)

func onCatJam(event *Event) {
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
func onLetsPlay(event *Event) {
	event.Respond(&Response{
		Type: ResponseModal,
		Data: &ResponseData{
			CustomID: "modals_survey_" + event.Data.Interaction.Member.User.ID,
			Title:    "Modals survey",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "opinion",
							Label:       "What is your opinion on them?",
							Style:       discordgo.TextInputShort,
							Placeholder: "Don't be shy, share your opinion with us",
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
			Handler: onLetsPlay,
		},
	}

	addCommands(generalCommands)
}
