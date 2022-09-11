package commands

import "github.com/bwmarrin/discordgo"

func init() {
	adminCommands := CommandMap{}

	addCommandsAdvanced(adminCommands, discordgo.PermissionAdministrator)
}
