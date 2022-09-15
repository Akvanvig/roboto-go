package commands

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	// Experimental official go package
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

type (
	Response      = discordgo.InteractionResponse
	ResponseData  = discordgo.InteractionResponseData
	CommandOption = discordgo.ApplicationCommandOption

	Event struct {
		Session *discordgo.Session           // Required
		Data    *discordgo.InteractionCreate // Required
	}

	CommandBase = discordgo.ApplicationCommand
	Command     struct {
		State              CommandBase                                         // Required
		Handler            func(cmd *Command, event *Event)                    // Required
		HandlerModalSubmit func(cmd *Command, event *Event, identifier string) // Optional
		Check              func(cmd *Command, event *Event) error              // Optional
		Registered         bool                                                // Not set
	}
)

var allCommands = map[string]*Command{}

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response)

	if err != nil {
		log.Error().Str("message", "Failed to send a response to discord").Err(err).Send()
	}

	return err
}

func (event *Event) RespondMsg(msg string) error {
	return event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: msg,
		},
	})
}

func (event *Event) RespondError(err error) error {
	var fullUserName string
	uuid := uuid.New().String()

	if event.Data.Member != nil {
		fullUserName = event.Data.Interaction.Member.User.Username + "#" + event.Data.Member.User.Discriminator

	} else {
		fullUserName = event.Data.Interaction.User.Username + "#" + event.Data.Interaction.User.Discriminator
	}

	log.Error().Str("message", "Responded with an error to a user interaction").Str("username", fullUserName).Str("uuid", uuid).Err(err).Send()

	return event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: err.Error() + ", Error ID: " + uuid,
		},
	})
}

func (command *Command) GenerateModalID(userData string) string {
	if userData != "" {
		return command.State.Name + "_" + userData
	}

	return command.State.Name
}

func addCommands(commands []*Command) {
	for _, cmd := range commands {
		allCommands[cmd.State.Name] = cmd
	}
}

func addCommandsAdvanced(commands []*Command, permissions int64, check func(cmd *Command, event *Event) error) {
	for _, cmd := range commands {
		// See https://github.com/bwmarrin/discordgo/blob/v0.26.1/structs.go#L1988 for permissions
		cmd.State.DefaultMemberPermissions = &permissions
		cmd.Check = check

		allCommands[cmd.State.Name] = cmd
	}
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
		Session: s,
		Data:    i,
	}

	switch event.Data.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		cmd, ok := allCommands[event.Data.ApplicationCommandData().Name]

		if !ok {
			break
		}

		if cmd.Check != nil {
			err = cmd.Check(cmd, &event)
		}

		if err != nil {
			event.RespondError(err)
		} else {
			cmd.Handler(cmd, &event)
		}

		return
	case discordgo.InteractionModalSubmit:
		modalData := strings.SplitN(event.Data.ModalSubmitData().CustomID, "_", 2)
		cmd, ok := allCommands[modalData[0]]

		if !ok {
			break
		}

		if len(modalData) > 1 {
			cmd.HandlerModalSubmit(cmd, &event, modalData[1])
		} else {
			cmd.HandlerModalSubmit(cmd, &event, "")
		}

		return
	}

	event.RespondError(errors.New("An internal error occured"))
}
