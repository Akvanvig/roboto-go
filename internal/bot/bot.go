package bot

import (
	"fmt"

	"github.com/Akvanvig/roboto-go/internal/bot/lib/commands"
	_ "github.com/Akvanvig/roboto-go/internal/bot/modules"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var session *discordgo.Session

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Info().Msg(fmt.Sprintf("Connected as: %s#%s", s.State.User.Username, s.State.User.Discriminator))
}

func onMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Nothing yet
}

// Note(Fredrico):
// Currently discordgo has not added support for auditlog entry creations yet...
// See https://github.com/bwmarrin/discordgo/pull/1314
/*
func onAuditlog(s *discordgo.Session, l *discordgo.AuditLogEntryCreate) {
	switch *l.ActionType {
	case discordgo.AuditLogActionMemberKick:
		fallthrough
	case discordgo.AuditLogActionMemberDisconnect:
		if l.TargetID == s.State.User.ID {
			log.Info().Msg("Detected event!")
		}
	}
}
*/

/*
func onVoiceStateUpdate(s *discordgo.Session, st *discordgo.VoiceStateUpdate) {
	// We only care about ourselves
	if st.UserID != s.State.User.ID {
		return
	}

	st.VoiceState.
		log.Info().Msg(fmt.Sprintf("Test %+v", st))
}
*/

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
	session.AddHandler(commands.Process)
	//session.AddHandler(onAuditlog)

	err = session.Open()

	if err != nil {
		log.Fatal().Err(err).Msg("Cannot open a session")
	}

	err = commands.Sync(session)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed synchronization step")
	}

	log.Info().Msg("Bot is ready")
}

func Stop() {
	log.Info().Msg("Stopping the bot")

	if session != nil {
		err := session.Close()

		if err != nil {
			log.Error().Err(err).Msg("Failed to close the session properly")
		}

		session = nil
	}
}
