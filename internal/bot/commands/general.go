package commands

import (
	"errors"
	"os"
	"path"

	. "github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/bwmarrin/discordgo"
)

func onCatJam(i *InteractionCreate) (*Response, error) {
	file, err := os.Open(path.Join(RootPath, "assets/img/catjam.gif"))

	if err != nil {
		return nil, errors.New("Failed to open catjam asset")
	}

	defer file.Close()

	return &Response{
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
	}, nil
}

func init() {
	generalCommands := &CommandMap{
		"catjam": &Command{
			State: CommandInfo{
				Name:        "catjam",
				Description: "Let's jam!",
			},
			Handler: onCatJam,
		},
	}

	addCommands(generalCommands)
}
