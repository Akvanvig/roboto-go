package api

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var allCommandsRaw = []*discordgo.ApplicationCommand{}
var allCommands = map[string]CommandOption{}
var allCachedResponseData = map[string]ResponseData{}
var mutexCache = sync.RWMutex{}

func convertCommandOptions(parentKey string, options []CommandOption, converters ...func(cmd *CommandOption)) []*discordgo.ApplicationCommandOption {
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
					"Chat subcommand of type '%d' can't have 'Name' with spaces in it. Ignoring command '%s' in module group '%s'",
					cmd.Type, cmd.Name, parentKey))
				continue
			}

			if cmd.Description == "" {
				log.Warn().Msg(fmt.Sprintf(
					"Chat subcommand of type '%d' must have a 'Description'. Ignoring command '%s' in module group '%s'",
					cmd.Type, cmd.Name, parentKey))
				continue
			}

			if cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
				key = fmt.Sprintf("%s_%s", parentKey, cmd.Name)

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
			Options:                  convertCommandOptions(key, cmd.Options),
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

func initContextCommands(settings *CommandGroupSettings, commands []CommandOption, callerName string, contextType discordgo.ApplicationCommandType, converters ...func(cmd *CommandOption)) {
	for i := 0; i < len(commands); i++ {
		cmd := &commands[i]

		if cmd.Name == "" {
			log.Warn().Msg(fmt.Sprintf(
				"A command is missing a 'Name' field. Ignoring command at the index '%d' in module group '%s'",
				i, callerName))
			continue
		}

		if cmd.Type != 0 {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't have a set subtype. Ignoring set value on command '%s' in module group '%s'",
				contextType, cmd.Name, callerName))
			cmd.Type = 0
		}

		if cmd.Description != "" || cmd.DescriptionLocalizations != nil {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't have a description. Ignoring set value on command '%s' in module group '%s'",
				cmd.Type, cmd.Name, callerName))
			cmd.Description = ""
			cmd.DescriptionLocalizations = nil
		}

		if cmd.Options != nil {
			log.Warn().Msg(fmt.Sprintf(
				"Click context command of type '%d' can't contain options array. Ignoring set value on command '%s' in module group '%s'",
				cmd.Type, cmd.Name, callerName))
			cmd.Options = nil
		}

		key := fmt.Sprintf("clickcontext_%s", cmd.Name)

		for j := 0; j < len(converters); j++ {
			converters[j](cmd)
		}

		// Create topmost command
		cmdRaw := &discordgo.ApplicationCommand{
			Name: cmd.Name,
			Type: contextType,
		}

		if settings != nil {
			cmdRaw.DefaultMemberPermissions = settings.DefaultMemberPermissions
			cmdRaw.DMPermission = settings.DMPermission
			cmdRaw.NSFW = settings.NSFW
		}

		allCommands[key] = *cmd
		allCommandsRaw = append(allCommandsRaw, cmdRaw)
	}
}

func InitUserCommands(settings *CommandGroupSettings, commands []CommandOption, converters ...func(cmd *CommandOption)) {
	callerName := util.GetCallingFuncFileName()
	initContextCommands(settings, commands, callerName, discordgo.UserApplicationCommand, converters...)
}

func InitMessageCommands(settings *CommandGroupSettings, commands []CommandOption, converters ...func(cmd *CommandOption)) {
	callerName := util.GetCallingFuncFileName()
	initContextCommands(settings, commands, callerName, discordgo.MessageApplicationCommand, converters...)
}

func InitChatCommands(settings *CommandGroupSettings, commands []CommandOption, converters ...func(cmd *CommandOption)) {
	callerName := util.GetCallingFuncFileName()

	// Correction invalid top types
	for i := 0; i < len(commands); i++ {
		cmd := &commands[i]

		if cmd.Type == discordgo.ApplicationCommandOptionSubCommandGroup || cmd.Type == discordgo.ApplicationCommandOptionSubCommand {
			continue
		}

		if cmd.Type != 0 {
			log.Warn().Msg(fmt.Sprintf(
				"Chat command type always has to be a subcommand or subcommand group at the top level. Forcefully correcting type on command '%s' in module group '%s'",
				cmd.Name, callerName))
		}

		cmd.Type = discordgo.ApplicationCommandOptionSubCommand
	}

	// Create topmost command
	cmdRaw := &discordgo.ApplicationCommand{
		Name:        callerName,
		Type:        discordgo.ChatApplicationCommand,
		Description: fmt.Sprintf("Commands belonging to the %s category", callerName),
		Options:     convertCommandOptions(callerName, commands, converters...),
	}

	if settings != nil {
		cmdRaw.DefaultMemberPermissions = settings.DefaultMemberPermissions
		cmdRaw.DMPermission = settings.DMPermission
		cmdRaw.NSFW = settings.NSFW
	}

	allCommandsRaw = append(allCommandsRaw, cmdRaw)
}

func SyncCommands(s *discordgo.Session) error {
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

func ProcessCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
