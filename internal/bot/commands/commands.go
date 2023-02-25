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

	ContextTypeUser    = discordgo.UserApplicationCommand
	ContextTypeMessage = discordgo.MessageApplicationCommand
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
	OnPassingCheck func(*Event) error
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

func createChatCommands(commands []Command, converters ...CommandConverter) {
	fileNameCaller := util.GetCallingFuncFileName()
	var builder strings.Builder

	// Define options generator function
	var generateOptions func(string, []CommandOption) []*discordgo.ApplicationCommandOption
	generateOptions = func(groupName string, options []CommandOption) []*discordgo.ApplicationCommandOption {
		optionsLen := len(options)

		if optionsLen == 0 {
			return nil
		}

		optionsGenerated := make([]*discordgo.ApplicationCommandOption, optionsLen)

		for i := 0; i < optionsLen; i++ {
			cmd := &options[i]
			optionsGenerated[i] = &discordgo.ApplicationCommandOption{
				Type:                     cmd.Type,
				Name:                     cmd.Name,
				Description:              cmd.Description,
				DescriptionLocalizations: cmd.DescriptionLocalizations,
				ChannelTypes:             cmd.ChannelTypes,
				Required:                 cmd.Required,
				Autocomplete:             cmd.Autocomplete,
				Choices:                  cmd.Choices,
				MinValue:                 cmd.MinValue,
				MaxValue:                 cmd.MaxValue,
				MinLength:                cmd.MinLength,
				MaxLength:                cmd.MaxLength,
			}

			if cmd.Type == discordgo.ApplicationCommandOptionSubCommandGroup || cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
				fmt.Fprintf(&builder, "%s_%s", groupName, cmd.Name)
				key := builder.String()
				builder.Reset()

				cmd.key = key
				allCommands[key] = *cmd
				optionsGenerated[i].Options = generateOptions(key, cmd.Options)
			}
		}

		return optionsGenerated
	}

	for i := 0; i < len(commands); i++ {
		cmd := &commands[i]

		// Run user defined converters
		for _, convert := range converters {
			convert(cmd)
		}

		// Override topmost type if it's not set to ApplicationCommandOptionSubCommandGroup
		switch cmd.Type {
		case discordgo.ApplicationCommandOptionSubCommandGroup:
			continue
		case discordgo.ApplicationCommandOptionSubCommand:
			continue
		default:
			fmt.Fprintf(&builder,
				"Chat command type always has to be 'ApplicationCommandOptionSubCommandGroup' or 'ApplicationCommandOptionSubCommand' at the top level. Forcefully correcting type on command '%s' in the '%s' category",
				cmd.Name, fileNameCaller)

			log.Warn().Msg(builder.String())
			builder.Reset()
			fallthrough
		case 0:
			cmd.Type = discordgo.ApplicationCommandOptionSubCommand
		}
	}

	// Build description
	fmt.Fprintf(&builder, "Commands belonging to the %s category", fileNameCaller)
	description := builder.String()
	builder.Reset()

	// Append createdCommands to temporary init commands list
	allCommandsRaw = append(allCommandsRaw, &discordgo.ApplicationCommand{
		Name:        fileNameCaller,
		Type:        discordgo.ChatApplicationCommand,
		Description: description,
		Options:     generateOptions(fileNameCaller, commands),
	})
}

func createContextCommands(commands []Command, contextType discordgo.ApplicationCommandType, converters ...CommandConverter) {
	fileNameCaller := util.GetCallingFuncFileName()

	for i := 0; i < len(commands); i++ {
		var builder strings.Builder
		cmd := &commands[i]

		if commands[i].Type != 0 {
			fmt.Fprintf(&builder,
				"Context command can't have a set type. Ignoring set value on command '%s' in the '%s' category",
				cmd.Name, fileNameCaller)

			log.Warn().Msg(builder.String())
			builder.Reset()
			cmd.Type = 0
		}

		if commands[i].Description != "" {
			fmt.Fprintf(&builder,
				"Context command can't have a description. Ignoring set value on command '%s' in the '%s' category",
				cmd.Name, fileNameCaller)

			log.Warn().Msg(builder.String())
			builder.Reset()
			cmd.Description = ""
		}

		if cmd.Options != nil {
			fmt.Fprintf(&builder,
				"Context command can't contain options array. Ignoring set value on command '%s' in the '%s' category",
				cmd.Name, fileNameCaller)

			log.Warn().Msg(builder.String())
			builder.Reset()
			cmd.Options = nil
		}

		fmt.Fprintf(&builder, "context_%s", commands[i].Name)
		key := builder.String()
		builder.Reset()

		allCommands[key] = *cmd
		allCommandsRaw = append(allCommandsRaw, &discordgo.ApplicationCommand{
			Name: commands[i].Name,
			Type: contextType,
		})
	}
}

// ToDo(Fredrico):
// Rename the function and perhaps try to make it have more sensible parameters
func parseRawCommandInteractionData(data *discordgo.ApplicationCommandInteractionData) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
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
		} else {
			fmt.Fprintf(&builder, "_%s", options[0].Name)
		}

		options = option.Options
	}

	return builder.String(), options
}

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response)

	if err != nil {
		log.Error().Err(err).Msg("Failed to send a response to discord")
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
		key, options := parseRawCommandInteractionData(&data)

		cmd, ok := allCommands[key]
		if !ok {
			break
		}

		event.Command = &cmd
		event.Options = options

		if cmd.Handler.OnPassingCheck != nil {
			err = cmd.Handler.OnPassingCheck(&event)
			if err != nil {
				break
			}
		}

		cmd.Handler.OnRun(&event)

		return
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
