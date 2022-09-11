package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	// Experimental official go package
	"golang.org/x/exp/maps"
)

type (
	Session           = discordgo.Session
	InteractionCreate = discordgo.InteractionCreate

	Response     = discordgo.InteractionResponse
	ResponseData = discordgo.InteractionResponseData

	CommandInfo = discordgo.ApplicationCommand
	Command     struct {
		State      CommandInfo                                   // Required
		Handler    func(i *InteractionCreate) (*Response, error) // Required
		Check      func(s *Session, i *InteractionCreate) error  // Optional
		Registered bool                                          // Not set
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

var All = CommandMap{}

func addCommands(commands CommandMap) {
	maps.Copy(All, commands)
}

// Note(Fredrico):
// See https://github.com/bwmarrin/discordgo/blob/v0.26.1/structs.go#L1988 for permissions
// TODO(Fredrico):
// This needs to be improved with check addition
func addCommandsAdvanced(commands CommandMap, permissions int64) {
	for _, val := range commands {
		val.State.DefaultMemberPermissions = &permissions
	}

	addCommands(commands)
}

func SendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, response *Response) {
	s.InteractionRespond(i.Interaction, response)
}

func SendErrorCheckFailed(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	errUUID := uuid.New().String()
	log.Warn().Str("message", err.Error()).Str("uuid", errUUID).Err(err).Send()

	SendResponse(s, i, &Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: "Check failed, this incident will be reported",
		},
	})
}

func SendErrorInternal(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	errUUID := uuid.New().String()
	log.Error().Str("message", err.Error()).Str("uuid", errUUID).Err(err).Send()

	SendResponse(s, i, &Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: fmt.Sprintf("An internal error occured, please provide the following UUID to the bot owner: %s", errUUID),
		},
	})
}
