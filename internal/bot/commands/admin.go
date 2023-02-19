package commands

import "github.com/bwmarrin/discordgo"

func init() {
	adminCommands := []Command{}

	addCommandsAdvanced(adminCommands, discordgo.PermissionAdministrator, nil)
}
