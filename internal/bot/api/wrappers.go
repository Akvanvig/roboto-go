package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/cnf/structhash"
	"github.com/rs/zerolog/log"
)

const ResponsePong = discordgo.InteractionResponsePong
const ResponseMsg = discordgo.InteractionResponseChannelMessageWithSource
const ResponseMsgLater = discordgo.InteractionResponseDeferredChannelMessageWithSource
const ResponseMsgUpdate = discordgo.InteractionResponseUpdateMessage
const ResponseMsgUpdateLater = discordgo.InteractionResponseDeferredMessageUpdate
const ResponseAutoComplete = discordgo.InteractionApplicationCommandAutocompleteResult
const ResponseModal = discordgo.InteractionResponseModal
const TextInputShort = discordgo.TextInputShort
const TextInputParagraph = discordgo.TextInputParagraph
const PrimaryButton = discordgo.PrimaryButton
const SecondaryButton = discordgo.SecondaryButton
const SuccessButton = discordgo.SuccessButton
const DangerButton = discordgo.DangerButton
const LinkButton = discordgo.LinkButton
const MessageFlagsEphemeral = discordgo.MessageFlagsEphemeral
const MessageFlagsCrossPosted = discordgo.MessageFlagsCrossPosted
const MessageFlagsFailedToMentionSomeRolesInThread = discordgo.MessageFlagsFailedToMentionSomeRolesInThread
const MessageFlagsHasThread = discordgo.MessageFlagsHasThread
const MessageFlagsIsCrossPosted = discordgo.MessageFlagsIsCrossPosted
const MessageFlagsLoading = discordgo.MessageFlagsLoading
const MessageFlagsSourceMessageDeleted = discordgo.MessageFlagsSourceMessageDeleted
const MessageFlagsSuppressEmbeds = discordgo.MessageFlagsSuppressEmbeds
const MessageFlagsSupressEmbeds = discordgo.MessageFlagsSupressEmbeds
const MessageFlagsUrgent = discordgo.MessageFlagsUrgent
const CommandOptionSubCommand = discordgo.ApplicationCommandOptionSubCommand
const CommandOptionSubCommandGroup = discordgo.ApplicationCommandOptionSubCommandGroup
const CommandOptionString = discordgo.ApplicationCommandOptionString
const CommandOptionInteger = discordgo.ApplicationCommandOptionInteger
const CommandOptionBoolean = discordgo.ApplicationCommandOptionBoolean
const CommandOptionUser = discordgo.ApplicationCommandOptionUser
const CommandOptionChannel = discordgo.ApplicationCommandOptionChannel
const CommandOptionRole = discordgo.ApplicationCommandOptionRole
const CommandOptionMentionable = discordgo.ApplicationCommandOptionMentionable
const CommandOptionNumber = discordgo.ApplicationCommandOptionNumber
const CommandOptionAttachment = discordgo.ApplicationCommandOptionAttachment
const ChannelTypeGuildText = discordgo.ChannelTypeGuildText
const ChannelTypeDM = discordgo.ChannelTypeDM
const ChannelTypeGuildVoice = discordgo.ChannelTypeGuildVoice
const ChannelTypeGroupDM = discordgo.ChannelTypeGroupDM
const ChannelTypeGuildCategory = discordgo.ChannelTypeGuildCategory
const ChannelTypeGuildNews = discordgo.ChannelTypeGuildNews
const ChannelTypeGuildStore = discordgo.ChannelTypeGuildStore
const ChannelTypeGuildNewsThread = discordgo.ChannelTypeGuildNewsThread
const ChannelTypeGuildPublicThread = discordgo.ChannelTypeGuildPublicThread
const ChannelTypeGuildPrivateThread = discordgo.ChannelTypeGuildPrivateThread
const ChannelTypeGuildStageVoice = discordgo.ChannelTypeGuildStageVoice
const ChannelTypeGuildForum = discordgo.ChannelTypeGuildForum

type File = discordgo.File
type Locale = discordgo.Locale
type ChannelType = discordgo.ChannelType
type CommandOptionType = discordgo.ApplicationCommandOptionType
type CommandOptionChoice = discordgo.ApplicationCommandOptionChoice
type MessageEmbed = discordgo.MessageEmbed
type MessageEmbedField = discordgo.MessageEmbedField
type MessageAllowedMentions = discordgo.MessageAllowedMentions
type ButtonStyle = discordgo.ButtonStyle
type ComponentEmoji = discordgo.ComponentEmoji
type SelectMenuType = discordgo.SelectMenuType
type TextInputStyle = discordgo.TextInputStyle
type MessageFlags = discordgo.MessageFlags
type SelectMenuOption = discordgo.SelectMenuOption
type MessageComponenResolved = discordgo.MessageComponentInteractionDataResolved
type MessageAttachment = discordgo.MessageAttachment
type MessageReference = discordgo.MessageReference

type MessageSend struct {
	Content         string
	Embeds          []*MessageEmbed
	TTS             bool
	Actions         []ActionsRow
	Files           []*File
	AllowedMentions *MessageAllowedMentions
	Reference       *MessageReference
	// Event handlers
	Handler *MessageHandler
}

func (message *MessageSend) ConvertToOriginal() *discordgo.MessageSend {
	var components []discordgo.MessageComponent

	if message.Actions != nil {
		id, err := structhash.Hash(message.Actions, 1)
		if err != nil {
			log.Fatal().Msg("Failed to cache a response data struct")
		}

		components = make([]discordgo.MessageComponent, len(message.Actions))
		for i := 0; i < len(components); i++ {
			components[i] = message.Actions[i].ConvertToOriginal(i, id)
		}

		mutexCache.RLock()
		_, ok := allCachedResponseData[id]
		mutexCache.RUnlock()
		if !ok {
			log.Debug().Msg(fmt.Sprintf("Cached response data for '%+v' using the ID '%s'", message, id))
			mutexCache.Lock()
			allCachedResponseData[id] = ResponseData{
				Actions: message.Actions,
				Handler: &ResponseHandler{
					OnComponentSubmit: message.Handler.OnComponentSubmit,
				},
			}
			mutexCache.Unlock()
		}
	}

	return &discordgo.MessageSend{
		Content:         message.Content,
		Embeds:          message.Embeds,
		TTS:             message.TTS,
		Components:      components,
		Files:           message.Files,
		AllowedMentions: message.AllowedMentions,
		Reference:       message.Reference,
	}
}

type MessageEdit struct {
	Content         *string
	Actions         []ActionsRow
	Embeds          []*MessageEmbed
	AllowedMentions *MessageAllowedMentions
	Flags           MessageFlags
	// Files to append to the message
	Files []*File
	// Overwrite existing attachments
	Attachments *[]*MessageAttachment

	ID      string
	Channel string
	// Event handlers
	Handler *MessageHandler
}

func (message *MessageEdit) ConvertToOriginal() *discordgo.MessageEdit {
	var components []discordgo.MessageComponent

	if message.Actions != nil {
		id, err := structhash.Hash(message.Actions, 1)
		if err != nil {
			log.Fatal().Msg("Failed to cache a response data struct")
		}

		components = make([]discordgo.MessageComponent, len(message.Actions))
		for i := 0; i < len(components); i++ {
			components[i] = message.Actions[i].ConvertToOriginal(i, id)
		}

		mutexCache.RLock()
		_, ok := allCachedResponseData[id]
		mutexCache.RUnlock()
		if !ok {
			log.Debug().Msg(fmt.Sprintf("Cached response data for '%+v' using the ID '%s'", message, id))
			mutexCache.Lock()
			allCachedResponseData[id] = ResponseData{
				Actions: message.Actions,
				Handler: &ResponseHandler{
					OnComponentSubmit: message.Handler.OnComponentSubmit,
				},
			}
			mutexCache.Unlock()
		}
	}

	return &discordgo.MessageEdit{
		Content:         message.Content,
		Components:      components,
		Embeds:          message.Embeds,
		AllowedMentions: message.AllowedMentions,
		Flags:           message.Flags,
		Files:           message.Files,
		Attachments:     message.Attachments,
		ID:              message.ID,
		Channel:         message.Channel,
	}
}

type MessageHandler struct {
	OnComponentSubmit func(*ComponentEvent)
}

type MessageComponent interface {
	Type() discordgo.ComponentType
}

type ActionsRow struct {
	Components []MessageComponent
}

func (row *ActionsRow) ConvertToOriginal(index int, id string) discordgo.ActionsRow {
	components := make([]discordgo.MessageComponent, len(row.Components))

	for i := 0; i < len(components); i++ {
		component := row.Components[i]

		switch component.Type() {
		case discordgo.ButtonComponent:
			buttonComponent := component.(Button)
			var customId string

			// Custom ID is mutually exclusive with URLs.
			if buttonComponent.URL == "" {
				customId = fmt.Sprintf("%s_%d_%d", id, index, i)
			} else {
				buttonComponent.Style = LinkButton
			}

			components[i] = discordgo.Button{
				Label:    buttonComponent.Label,
				Style:    buttonComponent.Style,
				Disabled: buttonComponent.Disabled,
				Emoji:    buttonComponent.Emoji,
				URL:      buttonComponent.URL,
				CustomID: customId,
			}
		case discordgo.SelectMenuComponent:
			selectComponent := component.(SelectMenu)
			components[i] = discordgo.SelectMenu{
				MenuType:     selectComponent.MenuType,
				CustomID:     fmt.Sprintf("%s_%d_%d", id, index, i),
				Placeholder:  selectComponent.Placeholder,
				MinValues:    selectComponent.MinValues,
				MaxValues:    selectComponent.MaxValues,
				Options:      selectComponent.Options,
				Disabled:     selectComponent.Disabled,
				ChannelTypes: selectComponent.ChannelTypes,
			}
		case discordgo.TextInputComponent:
			textComponent := component.(TextInput)
			components[i] = discordgo.TextInput{
				CustomID:    fmt.Sprintf("%s_%d_%d", id, index, i),
				Label:       textComponent.Label,
				Style:       textComponent.Style,
				Placeholder: textComponent.Placeholder,
				Value:       textComponent.Value,
				Required:    textComponent.Required,
				MinLength:   textComponent.MinLength,
				MaxLength:   textComponent.MaxLength,
			}

		}
	}

	return discordgo.ActionsRow{
		Components: components,
	}
}

type Button struct {
	Label    string
	Style    ButtonStyle
	Disabled bool
	Emoji    ComponentEmoji
	URL      string
}

func (button Button) Type() discordgo.ComponentType {
	return discordgo.ButtonComponent
}

type SelectMenu struct {
	// Type of the select menu.
	MenuType SelectMenuType
	// The text which will be shown in the menu if there's no default options or all options was deselected and component was closed.
	Placeholder string
	// This value determines the minimal amount of selected items in the menu.
	MinValues *int
	// This value determines the maximal amount of selected items in the menu.
	// If MaxValues or MinValues are greater than one then the user can select multiple items in the component.
	MaxValues int
	Options   []SelectMenuOption
	Disabled  bool

	// NOTE: Can only be used in SelectMenu with Channel menu type.
	ChannelTypes []ChannelType
}

func (menu SelectMenu) Type() discordgo.ComponentType {
	return discordgo.SelectMenuComponent
}

type TextInput struct {
	Label       string
	Style       TextInputStyle
	Placeholder string
	Value       string
	Required    bool
	MinLength   int
	MaxLength   int
}

func (input TextInput) Type() discordgo.ComponentType {
	return discordgo.TextInputComponent
}

type Response struct {
	Type discordgo.InteractionResponseType
	Data *ResponseData
}

func (response *Response) ConvertToOriginal() *discordgo.InteractionResponse {
	var data *discordgo.InteractionResponseData

	if response.Data != nil {
		data = response.Data.ConvertToOriginal(response.Type)
	}

	return &discordgo.InteractionResponse{
		Type: response.Type,
		Data: data,
	}
}

type ResponseData struct {
	TTS             bool
	Content         string
	Actions         []ActionsRow
	Embeds          []*MessageEmbed
	AllowedMentions *MessageAllowedMentions
	Files           []*File

	// NOTE: only MessageFlagsSuppressEmbeds and MessageFlagsEphemeral can be set.
	Flags MessageFlags

	// NOTE: autocomplete interaction only.
	Choices []*CommandOptionChoice

	// NOTE: modal interaction only.
	Title string
	// Event handlers
	Handler *ResponseHandler
}

func (data *ResponseData) ConvertToOriginal(type_ discordgo.InteractionResponseType) *discordgo.InteractionResponseData {
	var components []discordgo.MessageComponent
	var customId string

	if data.Actions != nil && (type_ == ResponseModal || type_ == ResponseMsg || type_ == ResponseMsgUpdate) {
		id, err := structhash.Hash(data.Actions, 1)
		if err != nil {
			log.Fatal().Msg("Failed to cache a response data struct")
		}

		components = make([]discordgo.MessageComponent, len(data.Actions))
		for i := 0; i < len(components); i++ {
			components[i] = data.Actions[i].ConvertToOriginal(i, id)
		}

		mutexCache.RLock()
		_, ok := allCachedResponseData[id]
		mutexCache.RUnlock()
		if !ok {
			log.Debug().Msg(fmt.Sprintf("Cached response data for '%+v' using the ID '%s'", data, id))
			mutexCache.Lock()
			allCachedResponseData[id] = *data
			mutexCache.Unlock()
		}

		if type_ == ResponseModal {
			customId = id
		}
	}

	return &discordgo.InteractionResponseData{
		TTS:             data.TTS,
		Content:         data.Content,
		Components:      components,
		Embeds:          data.Embeds,
		AllowedMentions: data.AllowedMentions,
		Files:           data.Files,
		Flags:           data.Flags,
		CustomID:        customId,
		Title:           data.Title,
	}
}

type ResponseDataUpdate struct {
	Content         *string
	Actions         *[]ActionsRow
	Embeds          *[]*MessageEmbed
	Files           []*File
	AllowedMentions *MessageAllowedMentions
	// Event handlers
	Handler *ResponseHandler
}

func (data *ResponseDataUpdate) ConvertToOriginal() *discordgo.WebhookEdit {
	var components []discordgo.MessageComponent

	if data.Actions != nil {
		id, err := structhash.Hash(data.Actions, 1)
		if err != nil {
			log.Fatal().Msg("Failed to cache a response data struct")
		}

		components = make([]discordgo.MessageComponent, len(*data.Actions))
		for i := 0; i < len(components); i++ {
			components[i] = (*data.Actions)[i].ConvertToOriginal(i, id)
		}

		mutexCache.RLock()
		_, ok := allCachedResponseData[id]
		mutexCache.RUnlock()
		if !ok {
			log.Debug().Msg(fmt.Sprintf("Cached response data for '%+v' using the ID '%s'", data, id))
			mutexCache.Lock()
			allCachedResponseData[id] = ResponseData{
				Actions: *data.Actions,
				Handler: data.Handler,
			}
			mutexCache.Unlock()
		}
	}

	return &discordgo.WebhookEdit{
		Content:         data.Content,
		Components:      &components,
		Embeds:          data.Embeds,
		Files:           data.Files,
		AllowedMentions: data.AllowedMentions,
	}
}

type ResponseHandler struct {
	OnComponentSubmit func(*ComponentEvent)
	OnModalSubmit     func(*ModalEvent)
}

type CommandGroupSettings struct {
	DefaultMemberPermissions *int64
	DMPermission             *bool
	NSFW                     *bool
}

type CommandOption struct {
	Type                     CommandOptionType
	Name                     string
	NameLocalizations        map[Locale]string
	Description              string
	DescriptionLocalizations map[Locale]string

	ChannelTypes []ChannelType
	Required     bool
	Options      []CommandOption

	// NOTE: mutually exclusive with Choices.
	Autocomplete bool
	Choices      []*CommandOptionChoice
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
}

type CommandHandler struct {
	OnRunCheck func(*CommandEvent) error
	OnRun      func(*CommandEvent)
}

type Event struct {
	Session *discordgo.Session           // Discord session
	Data    *discordgo.InteractionCreate // Event data
}

func (event *Event) ParseCommandData() (func(*CommandEvent), *CommandEvent) {
	if event.Data.Interaction.Type != discordgo.InteractionApplicationCommand {
		return nil, nil
	}

	data := event.Data.Interaction.ApplicationCommandData()
	options := data.Options
	var builder strings.Builder
	builder.WriteString(data.Name)

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

	key := builder.String()
	cmd, ok := allCommands[key]
	if !ok {
		cmd, ok = allCommands[fmt.Sprintf("clickcontext_%s", key)]
		if !ok {
			return nil, nil
		}
	}

	tmpEvent := &CommandEvent{
		Event:   *event,
		Command: &cmd,
		Options: options,
	}

	if cmd.Handler.OnRunCheck != nil {
		err := cmd.Handler.OnRunCheck(tmpEvent)
		if err != nil {
			event.RespondMsg(err.Error(), discordgo.MessageFlagsEphemeral)
			return nil, nil
		}
	}

	return cmd.Handler.OnRun, tmpEvent
}

func (event *Event) ParseModalData() (func(*ModalEvent), *ModalEvent) {
	if event.Data.Interaction.Type != discordgo.InteractionModalSubmit {
		return nil, nil
	}

	data := event.Data.Interaction.ModalSubmitData()
	key := data.CustomID

	mutexCache.RLock()
	responseData, ok := allCachedResponseData[key]
	mutexCache.RUnlock()
	if !ok {
		return nil, nil
	}
	if responseData.Handler == nil {
		return nil, nil
	}
	if responseData.Handler.OnModalSubmit == nil {
		return nil, nil
	}

	return responseData.Handler.OnModalSubmit, &ModalEvent{
		Event:   *event,
		Actions: responseData.Actions,
	}
}

func (event *Event) ParseComponentData() (func(*ComponentEvent), *ComponentEvent) {
	if event.Data.Interaction.Type != discordgo.InteractionMessageComponent {
		return nil, nil
	}

	data := event.Data.Interaction.MessageComponentData()
	args := strings.Split(data.CustomID, "_")
	key := fmt.Sprintf("%s_%s", args[0], args[1])

	mutexCache.RLock()
	responseData, ok := allCachedResponseData[key]
	mutexCache.RUnlock()
	if !ok {
		return nil, nil
	}
	if responseData.Handler == nil {
		return nil, nil
	}
	if responseData.Handler.OnComponentSubmit == nil {
		return nil, nil
	}

	rowIndex, _ := strconv.Atoi(args[2])
	componentIndex, _ := strconv.Atoi(args[3])
	component := responseData.Actions[rowIndex].Components[componentIndex]

	return responseData.Handler.OnComponentSubmit, &ComponentEvent{
		Event:             *event,
		Component:         &component,
		SelectionResolved: &data.Resolved,
		SelectionValues:   data.Values,
	}
}

func (event *Event) SendChannelMessage(channelID string, data *MessageSend) error {
	if data == nil {
		return errors.New("Tried to use SendChannelMessage with a nil data value")
	}

	_, err := event.Session.ChannelMessageSendComplex(channelID, data.ConvertToOriginal())
	return err
}

func (event *Event) EditChannelMessage(data *MessageEdit) error {
	if data == nil {
		return errors.New("Tried to use EditChannelMessage with a nil data value")
	}

	_, err := event.Session.ChannelMessageEditComplex(data.ConvertToOriginal())
	return err
}

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response.ConvertToOriginal())

	if err != nil {
		log.Error().Err(err).Msg("Discord event response failed")
	}

	return err
}

func (event *Event) RespondLater(flags ...MessageFlags) error {
	var data *ResponseData

	switch len(flags) {
	case 0:
	case 1:
		data = &ResponseData{
			Flags: flags[0],
		}
	default:
		log.Fatal().Msg("Function can only take up to 1 flags parameter")
	}

	return event.Respond(&Response{
		Type: ResponseMsgLater,
		Data: data,
	})
}

func (event *Event) RespondMsg(msg string, flags ...MessageFlags) error {
	var tmpFlags MessageFlags

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

func (event *Event) RespondModal(data *ResponseData) error {
	return event.Respond(&Response{
		Type: ResponseModal,
		Data: data,
	})
}

func (event *Event) RespondUpdateDirect(data *ResponseData) error {
	return event.Respond(&Response{
		Type: ResponseMsgUpdate,
		Data: data,
	})
}

func (event *Event) RespondUpdateDirectMsg(msg string) error {
	return event.RespondUpdateDirect(&ResponseData{
		Content: msg,
	})
}

func (event *Event) RespondUpdateLater(data *ResponseDataUpdate) error {
	_, err := event.Session.InteractionResponseEdit(event.Data.Interaction, data.ConvertToOriginal())
	return err
}

func (event *Event) RespondUpdateLaterMsg(msg string) error {
	return event.RespondUpdateLater(&ResponseDataUpdate{
		Content: &msg,
	})
}

type CommandEvent struct {
	Event
	Command *CommandOption                                       // Command triggering the event
	Options []*discordgo.ApplicationCommandInteractionDataOption // Extracted options from the event data
}

type ModalEvent struct {
	Event
	Actions []ActionsRow // Extracted components from the event data
}

type ComponentEvent struct {
	Event
	Component         *MessageComponent
	SelectionResolved *MessageComponenResolved
	SelectionValues   []string
}
