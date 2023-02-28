package commands

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

var allCommandsRaw = []*discordgo.ApplicationCommand{}
var allCommands = map[string]CommandOption{}

type (
	Response           = discordgo.InteractionResponse
	ResponseData       = discordgo.InteractionResponseData
	ResponseDataUpdate = discordgo.WebhookEdit
)

type CommandOption struct {
	Type                     discordgo.ApplicationCommandOptionType
	Name                     string
	NameLocalizations        map[discordgo.Locale]string
	Description              string
	DescriptionLocalizations map[discordgo.Locale]string

	ChannelTypes []discordgo.ChannelType
	Required     bool
	Options      []CommandOption

	// NOTE: mutually exclusive with Choices.
	Autocomplete bool
	Choices      []*discordgo.ApplicationCommandOptionChoice
	// Minimal value of number/integer option.
	MinValue *float64
	// Maximum value of number/integer option.
	MaxValue float64
	// Minimum length of string option.
	MinLength *int
	// Maximum length of string option.
	MaxLength int

	// Event handlers
	Handler *CommandHandler
	// Full command key
	key string
}

type CommandHandler struct {
	// Check handler
	OnRunCheck func(*Event) error
	// Event handler
	OnRun func(*Event)
	// Modal event handler
	OnModalSubmit func(*Event)
}

type Command = CommandOption

type CommandConverter func(cmd *Command)

type Event struct {
	Session    *discordgo.Session                                   // Discord session
	Command    *Command                                             // Command triggering the event
	Data       *discordgo.InteractionCreate                         // Event data
	Options    []*discordgo.ApplicationCommandInteractionDataOption // Extracted options from the event data
	Components []discordgo.MessageComponent                         // Extracted components from the event data
}

// ToDo(Fredrico):
// Rename the function and perhaps try to make it have more sensible parameters
func _parseRawCommandInteractionData(data *discordgo.ApplicationCommandInteractionData) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
	var builder strings.Builder
	builder.WriteString(data.Name)

	options := data.Options
	for {
		if len(options) == 0 {
			break
		}

		option := options[0]
		if option.Type != discordgo.ApplicationCommandOptionSubCommandGroup && option.Type != discordgo.ApplicationCommandOptionSubCommand {
			break
		}

		fmt.Fprintf(&builder, "_%s", options[0].Name)
		options = option.Options
	}

	return builder.String(), options
}

func _convertCommandOptions(parentKey string, options []CommandOption, converters ...CommandConverter) []*discordgo.ApplicationCommandOption {
	optionsLen := len(options)

	if optionsLen == 0 {
		return nil
	}

	optionsConverted := make([]*discordgo.ApplicationCommandOption, optionsLen)
	validNum := 0

	// ToDo(Fredrico):
	// Add more error checking
	// See https://github.com/bwmarrin/discordgo/blob/master/examples/slash_commands/main.go#L162

	for i := 0; i < optionsLen; i++ {
		cmd := &options[i]
		var key string

		if cmd.Type == discordgo.ApplicationCommandOptionSubCommandGroup || cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
			if strings.Contains(cmd.Name, " ") {
				log.Warn().Msg(fmt.Sprintf(
					"Chat subcommand of type '%d' can't have 'Name' with spaces in it. Ignoring command '%s' from file group '%s'",
					cmd.Type, cmd.Name, parentKey))
				continue
			}

			if cmd.Description == "" {
				log.Warn().Msg(fmt.Sprintf(
					"Chat subcommand of type '%d' must have a 'Description'. Ignoring command '%s' from file group '%s'",
					cmd.Type, cmd.Name, parentKey))
				continue
			}

			if cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
				key = fmt.Sprintf("%s_%s", parentKey, cmd.Name)
				cmd.key = key

				for j := 0; j < len(converters); j++ {
					converters[j](cmd)
				}

				allCommands[key] = *cmd
			}
		}

		optionsConverted[validNum] = &discordgo.ApplicationCommandOption{
			Type:                     cmd.Type,
			Name:                     cmd.Name,
			Description:              cmd.Description,
			DescriptionLocalizations: cmd.DescriptionLocalizations,
			ChannelTypes:             cmd.ChannelTypes,
			Required:                 cmd.Required,
			Options:                  _convertCommandOptions(key, cmd.Options),
			Autocomplete:             cmd.Autocomplete,
			Choices:                  cmd.Choices,
			MinValue:                 cmd.MinValue,
			MaxValue:                 cmd.MaxValue,
			MinLength:                cmd.MinLength,
			MaxLength:                cmd.MaxLength,
		}
		validNum += 1
	}

	return optionsConverted[:validNum]
}

func _createContextCommands(commands []Command, callerName string, contextType discordgo.ApplicationCommandType, converters ...CommandConverter) {
	for i := 0; i < len(commands); i++ {
		cmd := &commands[i]

		if cmd.Name == "" {
			log.Warn().Msg(fmt.Sprintf(
				"A command is missing a 'Name' field. Ignoring command at the index '%d' from file '%s'",
				i, callerName))
			continue
		}

		if cmd.Type != 0 {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't have a set subtype. Ignoring set value on command '%s' in the '%s' category",
				contextType, cmd.Name, callerName))
			cmd.Type = 0
		}

		if cmd.Description != "" || cmd.DescriptionLocalizations != nil {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't have a description. Ignoring set value on command '%s' from file '%s'",
				cmd.Type, cmd.Name, callerName))
			cmd.Description = ""
			cmd.DescriptionLocalizations = nil
		}

		if cmd.Options != nil {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't contain options array. Ignoring set value on command '%s' from file '%s'",
				cmd.Type, cmd.Name, callerName))
			cmd.Options = nil
		}

		key := fmt.Sprintf("clickcontext_%s", cmd.Name)
		cmd.key = key

		for j := 0; j < len(converters); j++ {
			converters[j](cmd)
		}

		allCommands[key] = *cmd
		// ToDo(Fredrico):
		// This should probably set more options
		allCommandsRaw = append(allCommandsRaw, &discordgo.ApplicationCommand{
			Name: cmd.Name,
			Type: contextType,
		})
	}
}

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response)

	if err != nil {
		log.Error().Err(err).Msg("Discord event response failed")
	}

	return err
}

func (event *Event) RespondLater() error {
	return event.Respond(&Response{
		Type: ResponseMsgLater,
	})
}

func (event *Event) RespondModal(cmd *Command, responseData *ResponseData) error {
	// Automatically set modal ID to cmd key to enable the handler to work
	responseData.CustomID = cmd.key
	return event.Respond(&Response{
		Type: ResponseModal,
		Data: responseData,
	})
}

func (event *Event) RespondMsg(msg string, flags ...discordgo.MessageFlags) error {
	var tmpFlags discordgo.MessageFlags

	switch len(flags) {
	case 0:
		tmpFlags = 0
	case 1:
		tmpFlags = flags[0]
	default:
		log.Fatal().Msg("Function can only take up to 1 flags parameter")
	}

	return event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: msg,
			Flags:   tmpFlags,
		},
	})
}

func (event *Event) RespondUpdate(responseDataUpdate *ResponseDataUpdate) error {
	_, err := event.Session.InteractionResponseEdit(event.Data.Interaction, responseDataUpdate)
	return err
}

func (event *Event) RespondUpdateMsg(msg string) error {
	return event.RespondUpdate(&ResponseDataUpdate{
		Content: &msg,
	})
}

func (event *Event) RespondUpdateMsgLog(msg string) error {
	var userNameFull string
	uuid := uuid.New().String()

	if event.Data.Member != nil {
		userNameFull = event.Data.Interaction.Member.User.Username + "#" + event.Data.Member.User.Discriminator

	} else {
		userNameFull = event.Data.Interaction.User.Username + "#" + event.Data.Interaction.User.Discriminator
	}

	log.Info().Str("username", userNameFull).Str("uuid", uuid).Msg(msg)

	return event.RespondUpdateMsg(msg)
}

func CreateUserCommands(commands []Command, converters ...CommandConverter) {
	callerName := util.GetCallingFuncFileName()
	_createContextCommands(commands, callerName, discordgo.UserApplicationCommand, converters...)
}

func CreateMessageCommands(commands []Command, converters ...CommandConverter) {
	callerName := util.GetCallingFuncFileName()
	_createContextCommands(commands, callerName, discordgo.MessageApplicationCommand, converters...)
}

func CreateChatCommands(commands []Command, converters ...CommandConverter) {
	callerName := util.GetCallingFuncFileName()

	for i := 0; i < len(commands); i++ {
		cmd := &commands[i]

		if cmd.Type != discordgo.ApplicationCommandOptionSubCommandGroup && cmd.Type != discordgo.ApplicationCommandOptionSubCommand {
			if cmd.Type != 0 {
				log.Warn().Msg(fmt.Sprintf(
					"Chat command type always has to be set to 'ApplicationCommandOptionSubCommandGroup' or 'ApplicationCommandOptionSubCommand' at the top level. Forcefully correcting type on command '%s' in the '%s' category",
					cmd.Name, callerName))
			}

			cmd.Type = discordgo.ApplicationCommandOptionSubCommand
		}
	}

	allCommandsRaw = append(allCommandsRaw, &discordgo.ApplicationCommand{
		Name:        callerName,
		Type:        discordgo.ChatApplicationCommand,
		Description: fmt.Sprintf("Commands belonging to the %s category", callerName),
		Options:     _convertCommandOptions(callerName, commands, converters...),
	})
}

func Sync(s *discordgo.Session) error {
	log.Info().Msg("Synchronizing commands")

	{
		// Fetch existing commands
		commandsExisting, err := s.ApplicationCommands(s.State.User.ID, "")

		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch existing commands")
			return err
		}

		// Delete commands out of sync
		for _, cmd := range commandsExisting {
			deleteCommand := true

			for _, cmdTmp := range allCommandsRaw {
				if cmd.Name == cmdTmp.Name {
					deleteCommand = false
					break
				}
			}

			if !deleteCommand {
				continue
			}

			err = s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)

			if err != nil {
				log.Error().Msg("Failed to delete an out of sync command")
				return err
			}
		}
	}

	{
		// Bulk creation of commands
		_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", allCommandsRaw)

		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create commands")
			return err
		}

		allCommandsRaw = nil
	}

	{
		// Cleanup of init alloc data
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		bytesBefore := m.Alloc

		runtime.GC()

		runtime.ReadMemStats(&m)
		bytesAfter := m.Alloc

		log.Info().Uint64("bytes", bytesBefore-bytesAfter).Msg("Cleaned up temporary init data")
	}

	log.Info().Msg("Finished synchronizing commands")

	return nil
}

func Process(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error
	event := Event{
		Session: s,
		Data:    i,
	}

	switch event.Data.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		data := event.Data.ApplicationCommandData()
		key, options := _parseRawCommandInteractionData(&data)

		cmd, ok := allCommands[key]
		if !ok {
			// If command was not found, check if it's a clickcontext command instead
			cmd, ok = allCommands[fmt.Sprintf("clickcontext_%s", key)]

			if !ok {
				break
			}
		}

		event.Command = &cmd
		event.Options = options

		if cmd.Handler.OnRunCheck != nil {
			// If check fails, respond with error
			err = cmd.Handler.OnRunCheck(&event)
			if err != nil {
				event.RespondMsg(err.Error())
				return
			}
		}

		cmd.Handler.OnRun(&event)

		return
	case discordgo.InteractionMessageComponent:
		log.Warn().Msg("Received unsupported interaction")
	case discordgo.InteractionModalSubmit:
		data := event.Data.ModalSubmitData()
		key := data.CustomID

		cmd, ok := allCommands[key]
		if !ok {
			break
		}

		event.Command = &cmd
		event.Components = data.Components

		cmd.Handler.OnModalSubmit(&event)

		return
	}

	event.RespondMsg("An internal error occured")
}
