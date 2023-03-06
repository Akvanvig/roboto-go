package commands

import (
	"fmt"

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
type MessageAllowedMentions = discordgo.MessageAllowedMentions
type ButtonStyle = discordgo.ButtonStyle
type ComponentEmoji = discordgo.ComponentEmoji
type SelectMenuType = discordgo.SelectMenuType
type TextInputStyle = discordgo.TextInputStyle
type MessageFlags = discordgo.MessageFlags
type SelectMenuOption = discordgo.SelectMenuOption
type MessageComponenResolved = discordgo.MessageComponentInteractionDataResolved

type MessageComponent interface {
	Type() discordgo.ComponentType
}

type ActionsRow struct {
	Components []MessageComponent
}

type Button struct {
	Label    string
	Style    ButtonStyle
	Disabled bool
	Emoji    ComponentEmoji
	URL      string
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

type TextInput struct {
	Label       string
	Style       TextInputStyle
	Placeholder string
	Value       string
	Required    bool
	MinLength   int
	MaxLength   int
}

type Response struct {
	Type discordgo.InteractionResponseType
	Data *ResponseData
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

type ResponseDataUpdate struct {
	Content         *string
	Actions         *[]ActionsRow
	Embeds          *[]*MessageEmbed
	Files           []*File
	AllowedMentions *MessageAllowedMentions
	// Event handlers
	Handler *ResponseHandler
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

func (button Button) Type() discordgo.ComponentType {
	return discordgo.ButtonComponent
}

func (menu SelectMenu) Type() discordgo.ComponentType {
	return discordgo.SelectMenuComponent
}

func (input TextInput) Type() discordgo.ComponentType {
	return discordgo.TextInputComponent
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

func (response *Response) ConvertToOriginal() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: response.Type,
		Data: response.Data.ConvertToOriginal(response.Type),
	}
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
