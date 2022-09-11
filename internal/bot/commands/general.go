package commands

import (
	"os"
	"path"

	. "github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/bwmarrin/discordgo"
)

func onCatJam(s *Session, i *InteractionCreate) {
	file, err := os.Open(path.Join(RootPath, "assets/img/catjam.gif"))

	if err != nil {
		s.InteractionRespond(i.Interaction, generateResponseError("Failed to open catjam asset", err))
		return
	}

	defer file.Close()

	s.InteractionRespond(i.Interaction, &Response{
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
// We probably want to make a more efficient command adding API.
// We also need a way to easily limit commands to a group of people (e.g admins)
func init() {
	Command{
		State: Info{
			Name:        "catjam",
			Description: "Let's jam!",
		},
		Handler: onCatJam,
	}.add()
}
