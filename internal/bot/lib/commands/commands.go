package commands

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const ResponsePong = discordgo.InteractionResponsePong
const ResponseMsg = discordgo.InteractionResponseChannelMessageWithSource
const ResponseMsgLater = discordgo.InteractionResponseDeferredChannelMessageWithSource
const ResponseMsgUpdate = discordgo.InteractionResponseUpdateMessage
const ResponseMsgUpdateLater = discordgo.InteractionResponseDeferredMessageUpdate
const ResponseAutoComplete = discordgo.InteractionApplicationCommandAutocompleteResult
const ResponseModal = discordgo.InteractionResponseModal

var allCommandsRaw = []*discordgo.ApplicationCommand{}
var allCommands = map[string]CommandOption{}

type Response = discordgo.InteractionResponse
type ResponseData = discordgo.InteractionResponseData
type ResponseDataUpdate = discordgo.WebhookEdit

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

type CommandGroupSettings struct {
	DefaultMemberPermissions *int64
	DMPermission             *bool
	NSFW                     *bool
}

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
	var username string
	uuid := uuid.New().String()

	if event.Data.Member != nil {
		username = fmt.Sprintf("%s#%s", event.Data.Interaction.Member.User.Username, event.Data.Interaction.Member.User.Discriminator)

	} else {
		username = fmt.Sprintf("%s#%s", event.Data.Interaction.User.Username, event.Data.Interaction.User.Discriminator)
	}

	log.Info().Str("username", username).Str("uuid", uuid).Msg(msg)

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
				event.RespondMsg(err.Error(), discordgo.MessageFlagsEphemeral)
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

	event.RespondMsg("An internal error occured", discordgo.MessageFlagsEphemeral)
}
