package modules

import (
	"os"
	"path/filepath"

	. "github.com/Akvanvig/roboto-go/internal/bot/api"
	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/rs/zerolog/log"
)

func init() {
	InitChatCommands(nil, []CommandOption{
		{
			Name:        "catjam",
			Description: "Let's jam!",
			Handler: &CommandHandler{
				OnRun: onCatJam,
			},
		},
		{
			Name:        "gamewithme",
			Description: "Let's play a game",
			Handler: &CommandHandler{
				OnRun: onGameWithMe,
			},
		},
	})
	/*
		InitChatCommands(nil, []Command{
			{
				Name: "OPEEEN UP",
				Handler: &CommandHandler{
					OnRun:         onGameWithMe,
				},
			},
		})

		InitChatCommands(nil, []Command{
			{
				Name: "CHECK THIS OUT",
				Handler: &CommandHandler{
					OnRun:         onGameWithMe,
				},
			},
		})
	*/
}

func onCatJam(event *CommandEvent) {
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
			Files: []*File{
				{
					ContentType: "image/gif",
					Name:        "catjam.gif",
					Reader:      file,
				},
			},
			Actions: []ActionsRow{
				{
					Components: []MessageComponent{
						Button{
							Label: "Bonk the cat!",
							Style: DangerButton,
						},
					},
				},
			},
			Handler: &ResponseHandler{
				OnComponentSubmit: onCatBonk,
			},
		},
	})
}

func onCatBonk(event *ComponentEvent) {
	event.RespondUpdateDirect(&ResponseData{
		Content: "OUCH, You bonked me!",
		Actions: []ActionsRow{
			{
				Components: []MessageComponent{
					Button{
						Label: "Bonk again?",
						Style: DangerButton,
					},
					Button{
						Label: "Click me for more funny cats",
						Style: LinkButton,
						URL:   "https://www.youtube.com/watch?v=YSHDBB6id4A",
					},
				},
			},
		},
		Handler: &ResponseHandler{
			OnComponentSubmit: onCatBonk,
		},
	})
}

// TODO(Fredrico):
// This is unfinished
func onGameWithMe(event *CommandEvent) {
	event.RespondModal(&ResponseData{
		Title: "A Game",
		Actions: []ActionsRow{
			{
				Components: []MessageComponent{
					TextInput{
						Label:       "Why do you want to play the game?",
						Style:       TextInputShort,
						Placeholder: "Don't be shy, tell me",
						Required:    true,
						MaxLength:   300,
						MinLength:   10,
					},
				},
			},
		},
		Handler: &ResponseHandler{
			OnModalSubmit: onGameWithMeSubmit,
		},
	})
}

func onGameWithMeSubmit(event *ModalEvent) {
	event.RespondMsg("Thank you for playing!")
}
