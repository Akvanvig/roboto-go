package commands

import (
	"errors"

	"github.com/Akvanvig/roboto-go/internal/bot/music"
	"github.com/bwmarrin/discordgo"
)

func isGuildCmd(cmd *Command, event *Event) error {
	if event.Data.Interaction.Member == nil {
		return errors.New("You can not play a song in a DM")
	}

	return nil
}

// Note(Fredrico):
// All of this is going to be a hell to synchronize.
// Remember: We need to make this async
func onPlay(cmd *Command, event *Event) {
	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID, false)

	if player == nil {
		vs, _ := event.Session.State.VoiceState(guildID, event.Data.Interaction.Member.User.ID)

		if vs == nil {
			event.RespondMsg("You must be connected to a voice channel or use the connect command to stream a video")
			return
		}

		player = music.GetGuildPlayer(guildID, true)
		player.Connect(event.Session, vs.ChannelID)
	}

	player.Play("")

	event.RespondMsg("Congratulations! You played a video")
}

func onConnect(cmd *Command, event *Event) {
	guildID := event.Data.Interaction.GuildID
	channelID := event.Data.Interaction.ApplicationCommandData().Options[0].StringValue()

	guildChannels, _ := event.Session.GuildChannels(guildID)
	var voiceChannel *discordgo.Channel

	for _, channel := range guildChannels {
		if channel.ID != channelID {
			continue
		}

		if channel.Type == discordgo.ChannelTypeGuildVoice {
			voiceChannel = channel
		}

		break
	}

	if voiceChannel == nil {
		event.RespondMsg("The provided channel id is not valid")
		return
	}

	player := music.GetGuildPlayer(guildID, true)

	switch err := player.Connect(event.Session, channelID); err {
	case nil:
		event.RespondMsg("Connected to: " + voiceChannel.Name)
	case err.(music.ConnectionError):
		event.RespondMsg("The bot is already connected to the given voice channel")
	default:
		event.RespondError(err)
	}
}

func onDisconnect(cmd *Command, event *Event) {
	guildID := event.Data.Interaction.GuildID
	err := music.DeleteGuildPlayer(guildID)

	if err != nil {
		event.RespondMsg("The bot is not connected to a voice channel")
	} else {
		event.RespondMsg("The bot was disconnected from the voice channel")
	}
}

func init() {
	musicCommands := []*Command{
		{
			State: CommandBase{
				Name:        "play",
				Description: "Stream a youtube video",
				Options: []*CommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "video",
						Description: "The link or the name of the video",
						Required:    true,
					},
				},
			},
			Handler: onPlay,
		},
		{
			State: CommandBase{
				Name:        "connect",
				Description: "Connect bot to a voice channel",
				Options: []*CommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "channel",
						Description: "The voice channel id",
						Required:    true,
					},
				},
			},
			Handler: onConnect,
		},
		{
			State: CommandBase{
				Name:        "disconnect",
				Description: "Disconnect the bot from voice",
			},
			Handler: onDisconnect,
		},
	}

	addCommandsAdvanced(musicCommands, 0, isGuildCmd)
}
