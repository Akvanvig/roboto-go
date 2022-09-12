package bot

import (
	"github.com/Akvanvig/roboto-go/internal/bot/commands"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var session *discordgo.Session

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Info().Msg("Connected as: " + s.State.User.Username + "#" + s.State.User.Discriminator)
}

func onMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Nothing yet
}

func onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands.Process(s, i)
}

func Start(token *string) {
	var err error

	session, err = discordgo.New("Bot " + *token)

	// Note(Fredrico):
	// It's worth mentioning that discordgo does not check if the parameters are valid yet
	if err != nil {
		log.Fatal().Str("message", "Invalid bot parameters").Err(err).Send()
	}

	session.AddHandler(onReady)
	session.AddHandler(onMsg)
	session.AddHandler(onInteraction)

	err = session.Open()

	if err != nil {
		log.Fatal().Str("message", "Cannot open a session").Err(err).Send()
	}

	commands.Create(session)
}

func Stop() {
	log.Info().Msg("Stopping the bot")

	if session != nil {
		commands.Delete(session)

		err := session.Close()

		if err != nil {
			log.Error().Str("message", "Failed to close the session properly").Err(err).Send()
		}

		session = nil
	}
}
