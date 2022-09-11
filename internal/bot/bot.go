package bot

import (
	"fmt"

	"github.com/Akvanvig/roboto-go/internal/bot/commands"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var session *discordgo.Session

func onReady(session *discordgo.Session, r *discordgo.Ready) {
	log.Info().Msg(fmt.Sprintf("Connected as: %v#%v", session.State.User.Username, session.State.User.Discriminator))
}

func onInteraction(session *discordgo.Session, i *discordgo.InteractionCreate) {
	if cmd, ok := commands.All[i.ApplicationCommandData().Name]; ok {
		cmd.Handler(session, i)
	}
}

func initHandlers() {
	session.AddHandler(onInteraction)
	session.AddHandler(onReady)
}

func initCommands() {
	log.Info().Msg("Initializing commands")

	for name, cmd := range commands.All {
		updatedState, err := session.ApplicationCommandCreate(session.State.User.ID, "", &cmd.State)

		if err != nil {
			log.Error().Str("message", fmt.Sprintf("Could not create '%v' command: ", name)).Err(err).Send()
		}

		// Update command state
		cmd.State = *updatedState
		cmd.Registered = true
	}
}

func delCommands() {
	log.Info().Msg("Deleting commands")

	for name, cmd := range commands.All {
		if !cmd.Registered {
			continue
		}

		err := session.ApplicationCommandDelete(session.State.User.ID, "", cmd.State.ID)

		if err != nil {
			log.Error().Str("message", fmt.Sprintf("Failed to delete '%v' command: ", name)).Err(err).Send()
		}

		cmd.Registered = false
	}
}

func Start(token *string) {
	var err error

	session, err = discordgo.New("Bot " + *token)

	// Note(Fredrico):
	// It's worth mentioning that discordgo does not check if the parameters are valid yet
	if err != nil {
		log.Fatal().Str("message", "Invalid bot parameters: ").Err(err).Send()
	}

	initHandlers()

	err = session.Open()

	if err != nil {
		log.Fatal().Str("message", "Cannot open a session: ").Err(err).Send()
	}

	initCommands()
}

func Stop() {
	log.Info().Msg("Stopping the bot")

	if session != nil {
		delCommands()

		err := session.Close()

		if err != nil {
			log.Error().Str("message", "Failed to close the session properly").Err(err).Send()
		}

		session = nil
	}
}
