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
var allCommands = map[string]Command{}

type (
	Response           = discordgo.InteractionResponse
	ResponseData       = discordgo.InteractionResponseData
	ResponseDataUpdate = discordgo.WebhookEdit
	Command            = CommandOption
)

type Event struct {
	Session    *discordgo.Session                                   // Required
	Data       *discordgo.InteractionCreate                         // Required
	Options    []*discordgo.ApplicationCommandInteractionDataOption // Optional
	Components []discordgo.MessageComponent                         // Optional
	ID         string                                               // Optional
}

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

	// Event handler
	Handler func(cmd *Command, event *Event)
	// Modal event handler
	HandlerModalSubmit func(cmd *Command, event *Event, identifier string)
	// Commmand check handler
	Check func(cmd *Command, event *Event) error

	// Full command key
	key string
}

// Note(Fredrico):
/* Potential future parameters for addCommands
		Version           string                 `json:"version,omitempty"`
    DefaultMemberPermissions *int64 `json:"default_member_permissions,string,omitempty"`
    DMPermission             *bool  `json:"dm_permission,omitempty"`
    NSFW                     *bool  `json:"nsfw,omitempty"`
*/
func createCommands(commands []Command) {
	callerFuncName := util.GetCallingFuncFileName()

	// Define recursive parsing function
	var parseCommands func(parentGroupName string, commands []Command) []*discordgo.ApplicationCommandOption
	parseCommands = func(parentGroupName string, commands []Command) []*discordgo.ApplicationCommandOption {
		commandsLen := len(commands)

		if commandsLen == 0 {
			return nil
		}

		commandsConverted := make([]*discordgo.ApplicationCommandOption, commandsLen)

		for i := 0; i < commandsLen; i++ {
			var options []*discordgo.ApplicationCommandOption
			cmd := &commands[i]

			if cmd.Type == discordgo.ApplicationCommandOptionSubCommandGroup || cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
				var builder strings.Builder
				builder.WriteString(parentGroupName)
				builder.WriteString("_")
				builder.WriteString(cmd.Name)
				key := builder.String()

				cmd.key = key
				allCommands[key] = *cmd
				options = parseCommands(key, cmd.Options)
			}

			commandsConverted[i] = &discordgo.ApplicationCommandOption{
				Type:                     cmd.Type,
				Name:                     cmd.Name,
				Description:              cmd.Description,
				DescriptionLocalizations: cmd.DescriptionLocalizations,
				ChannelTypes:             cmd.ChannelTypes,
				Required:                 cmd.Required,
				Options:                  options,
				Autocomplete:             cmd.Autocomplete,
				Choices:                  cmd.Choices,
				MinValue:                 cmd.MinValue,
				MaxValue:                 cmd.MaxValue,
				MinLength:                cmd.MinLength,
				MaxLength:                cmd.MaxLength,
			}
		}

		return commandsConverted
	}

	// Override topmost type if it's not set to ApplicationCommandOptionSubCommandGroup
	for i := 0; i < len(commands); i++ {
		if commands[i].Type != discordgo.ApplicationCommandOptionSubCommandGroup {
			commands[i].Type = discordgo.ApplicationCommandOptionSubCommand
		}
	}

	name := callerFuncName
	description := fmt.Sprintf("Commands belonging to the %s category", callerFuncName)
	createdCommands := parseCommands(callerFuncName, commands)

	// Append createdCommands to temporary init commands list
	allCommandsRaw = append(allCommandsRaw, &discordgo.ApplicationCommand{
		Name:        name,
		Description: description,
		Options:     createdCommands,
	})
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
			builder.WriteString(fmt.Sprintf("_%s", options[0].Name))
		}

		options = option.Options
	}

	return builder.String(), options
}

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response)

	if err != nil {
		log.Error().Str("message", "Failed to send a response to discord").Err(err).Send()
	}

	return err
}

func (event *Event) RespondLater() error {
	return event.Respond(&Response{
		Type: ResponseMsgLater,
	})
}

func (event *Event) RespondMsg(msg string) error {
	return event.Respond(&Response{
		Type: ResponseMsg,
		Data: &ResponseData{
			Content: msg,
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

	log.Error().Str("username", userNameFull).Str("uuid", uuid).Msg(msg)

	return event.RespondUpdateMsg(msg)
}

func (command *Command) GenerateModalID(userData string) string {
	if userData != "" {
		return command.key + "|" + userData
	} else {
		return command.key
	}
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
		log.Info().Msg("Cleaning up temporary init data")

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		bytesBefore := m.Alloc

		runtime.GC()

		runtime.ReadMemStats(&m)
		bytesAfter := m.Alloc

		log.Info().Uint64("bytes", bytesBefore-bytesAfter).Msg("Finished cleaning up temporary init data")
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
		cmdKey, cmdOptions := parseRawCommandInteractionData(&data)

		cmd, ok := allCommands[cmdKey]
		if !ok {
			break
		}

		event.Options = cmdOptions

		if cmd.Check != nil {
			err = cmd.Check(&cmd, &event)
			if err != nil {
				break
			}
		}

		cmd.Handler(&cmd, &event)

		return
	case discordgo.InteractionModalSubmit:
		data := event.Data.ModalSubmitData()
		tmp := strings.SplitN(data.CustomID, "|", 2)
		cmdKey := tmp[0]

		cmd, ok := allCommands[cmdKey]
		if !ok {
			break
		}

		event.Components = data.Components

		var id string
		if len(tmp) > 1 {
			id = tmp[1]
		}

		cmd.HandlerModalSubmit(&cmd, &event, id)

		return
	}

	event.RespondMsg("An internal error occured")
}
