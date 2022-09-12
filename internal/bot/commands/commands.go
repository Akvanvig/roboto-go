package commands

import (
	"errors"
	"strings"

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
		Source *discordgo.Session           // Required
		Data   *discordgo.InteractionCreate // Required
	}

	CommandInfo = discordgo.ApplicationCommand
	Command     struct {
		State        CommandInfo                                         // Required
		Handler      func(cmd *Command, event *Event)                    // Optional
		HandlerModal func(cmd *Command, event *Event, identifier string) // Optional
		Check        func(cmd *Command, event *Event) error              // Optional
		Registered   bool                                                // Not set
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

var allCommands = CommandMap{}

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
			Content: errStr + " ID: " + errUUID,
		},
	})
}

func (command *Command) GenerateModalID(userData string) string {
	if userData != "" {
		return command.State.Name + "_" + userData
	}

	return command.State.Name
}

func addCommands(commands CommandMap) {
	maps.Copy(allCommands, commands)
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

	for cmdName, cmd := range allCommands {
		updatedState, err := s.ApplicationCommandCreate(s.State.User.ID, "", &cmd.State)

		if err != nil {
			log.Error().Str("message", "Could not create '"+cmdName+"' command").Err(err).Send()
		}

		// Update command state
		cmd.State = *updatedState
		cmd.Registered = true
	}
}

func Delete(s *discordgo.Session) {
	log.Info().Msg("Deleting commands")

	for cmdName, cmd := range allCommands {
		if !cmd.Registered {
			continue
		}

		err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.State.ID)

		if err != nil {
			log.Error().Str("message", "Failed to delete '"+cmdName+"' command: ").Err(err).Send()
		}

		cmd.Registered = false
	}
}

func Process(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error
	event := Event{
		Source: s,
		Data:   i,
	}

	switch event.Data.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		cmd, ok := allCommands[event.Data.ApplicationCommandData().Name]

		if !ok {
			break
		}

		if cmd.Check != nil {
			err = cmd.Check(cmd, &event)

			if err != nil {
				event.RespondError(errors.New("Check failed, this incident will be reported"))
				break
			}
		}

		cmd.Handler(cmd, &event)
		return
	case discordgo.InteractionModalSubmit:
		modalData := strings.SplitN(event.Data.ModalSubmitData().CustomID, "_", 2)
		cmd, ok := allCommands[modalData[0]]

		if !ok {
			break
		}

		if len(modalData) > 1 {
			cmd.HandlerModal(cmd, &event, modalData[1])
		} else {
			cmd.HandlerModal(cmd, &event, "")
		}

		return
	default:
		return
	}

	event.RespondError(errors.New("An internal error occured"))
}
