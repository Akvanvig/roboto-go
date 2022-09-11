package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type (
	Session           = discordgo.Session
	InteractionCreate = discordgo.InteractionCreate

	Response     = discordgo.InteractionResponse
	ResponseData = discordgo.InteractionResponseData

	Info    = discordgo.ApplicationCommand
	Command struct {
		State      Info
		Handler    func(s *Session, i *InteractionCreate)
		Registered bool
	}
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

var All = map[string]*Command{}

func (cmd Command) add() {
	All[cmd.State.Name] = &cmd
}

func generateResponseError(msg string, err error) *Response {
	errUUID := uuid.New().String()

	log.Error().Str("message", msg).Str("uuid", errUUID).Err(err).Send()

	return &Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: fmt.Sprintf("An internal error occured, please provide the following UUID to the bot owner: %s", errUUID),
		},
	}
}
