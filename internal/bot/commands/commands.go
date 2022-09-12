package commands

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	// Experimental official go package
	"golang.org/x/exp/maps"
)

type (
	Response     = discordgo.InteractionResponse
	ResponseData = discordgo.InteractionResponseData

	Event struct {
		Source *discordgo.Session
		Data   *discordgo.InteractionCreate
	}

	CommandInfo = discordgo.ApplicationCommand
	Command     struct {
		State        CommandInfo          // Required
		Handler      func(e *Event)       // Required
		HandlerModal func(e *Event)       // Optional
		Check        func(e *Event) error // Optional
		Registered   bool                 // Not set
	}
	CommandMap = map[string]*Command
)

const (
	ResponsePong           = discordgo.InteractionResponsePong
	ResponseMsg            = discordgo.InteractionResponseChannelMessageWithSource
	ResponseMsgLater       = discordgo.InteractionResponseDeferredChannelMessageWithSource
	ResponseMsgUpdate      = discordgo.InteractionResponseUpdateMessage
	ResponseMsgUpdateLater = discordgo.InteractionResponseDeferredMessageUpdate
	ResponseAutoComplete   = discordgo.InteractionApplicationCommandAutocompleteResult
	ResponseModal          = discordgo.InteractionResponseModal
)

var AllCommands = CommandMap{}

func (event *Event) Respond(response *Response) error {
	err := event.Source.InteractionRespond(event.Data.Interaction, response)

	if err != nil {
		log.Error().Str("message", "Failed to send a response to discord").Err(err).Send()
	}

	return err
}

func (event *Event) RespondError(err error) error {
	errStr := err.Error()
	errUUID := uuid.New().String()
	log.Error().Str("message", errStr).Str("uuid", errUUID).Err(err).Send()

	return event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: fmt.Sprintf("%s ID: %s", errStr, errUUID),
		},
	})
}

func addCommands(commands CommandMap) {
	maps.Copy(AllCommands, commands)
}

// TODO(Fredrico):
// This needs to be improved with check addition
func addCommandsAdvanced(commands CommandMap, permissions int64) {
	for _, val := range commands {
		// See https://github.com/bwmarrin/discordgo/blob/v0.26.1/structs.go#L1988 for permissions
		val.State.DefaultMemberPermissions = &permissions
	}

	addCommands(commands)
}

func Create(s *discordgo.Session) {
	log.Info().Msg("Creating commands")

	for cmdName, cmd := range AllCommands {
		updatedState, err := s.ApplicationCommandCreate(s.State.User.ID, "", &cmd.State)

		if err != nil {
			log.Error().Str("message", fmt.Sprintf("Could not create '%v' command: ", cmdName)).Err(err).Send()
		}

		// Update command state
		cmd.State = *updatedState
		cmd.Registered = true
	}
}

func Delete(s *discordgo.Session) {
	log.Info().Msg("Deleting commands")

	for cmdName, cmd := range AllCommands {
		if !cmd.Registered {
			continue
		}

		err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.State.ID)

		if err != nil {
			log.Error().Str("message", fmt.Sprintf("Failed to delete '%v' command: ", cmdName)).Err(err).Send()
		}

		cmd.Registered = false
	}
}

func Process(s *discordgo.Session, i *discordgo.InteractionCreate) {
	event := Event{
		Source: s,
		Data:   i,
	}
	var (
		cmdName string
		err     error
	)

	switch event.Data.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		cmdName = event.Data.ApplicationCommandData().Name
	case discordgo.InteractionModalSubmit:
		// TODO(Fredrico)
		cmdName = "None"
	}

	cmd, ok := AllCommands[cmdName]

	if !ok {
		event.RespondError(errors.New("An internal error occured"))
		return
	}

	switch event.Data.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		if cmd.Check != nil {
			err = cmd.Check(&event)

			if err != nil {
				event.RespondError(errors.New("Check failed, this incident will be reported"))
				return
			}
		}

		cmd.Handler(&event)
	case discordgo.InteractionModalSubmit:
	}
}
