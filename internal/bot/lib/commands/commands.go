package commands

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var allCommandsRaw = []*discordgo.ApplicationCommand{}
var allCommands = map[string]CommandOption{}
var allCachedResponseData = map[string]ResponseData{}
var mutexCache = sync.RWMutex{}

type Event struct {
	Session *discordgo.Session           // Discord session
	Data    *discordgo.InteractionCreate // Event data
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

func (event *Event) Respond(response *Response) error {
	err := event.Session.InteractionRespond(event.Data.Interaction, response.ConvertToOriginal())

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
	eventCore := Event{
		Session: s,
		Data:    i,
	}

	switch i.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		handler, event := eventCore.ParseCommandData()
		if handler != nil {
			handler(event)
			return
		}
	case discordgo.InteractionModalSubmit:
		handler, event := eventCore.ParseModalData()
		if handler != nil {
			handler(event)
			return
		}
	case discordgo.InteractionMessageComponent:
		handler, event := eventCore.ParseComponentData()
		if handler != nil {
			handler(event)
			return
		}
	}

	eventCore.RespondMsg("An error occurred -> You probably tried to interact with an old event", discordgo.MessageFlagsEphemeral)
}
