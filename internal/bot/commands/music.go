package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/bot/music"
	"github.com/bwmarrin/discordgo"
)

func init() {
	var (
		minVolume = 0.0
		maxVolume = 200.0
	)

	// Set common properties for all commands
	converter := func(cmd *Command) {
		cmd.Handler.OnPassingCheck = func(event *Event) error {
			if event.Data.Interaction.Member == nil {
				return errors.New("You can not play a song in a DM")
			}

			return nil
		}
	}

	createChatCommands([]Command{
		{
			Name:        "connect",
			Description: "Connect the bot to a voice channel",
			Options: []CommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "channel",
					Description: "The voice channel id",
					Required:    true,
				},
			},
			Handler: &CommandHandler{
				OnRun: onConnect,
			},
		},
		{
			Name:        "disconnect",
			Description: "Disconnect the bot from voice",
			Handler: &CommandHandler{
				OnRun: onDisconnect,
			},
		},
		{
			Name:        "play",
			Description: "Play a youtube video",
			Options: []CommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "video",
					Description: "The link of the video",
					Required:    true,
				},
			},
			Handler: &CommandHandler{
				OnRun: onPlay,
			},
		},
		{
			Name:        "replay",
			Description: "Toggle replay mode",
			Handler: &CommandHandler{
				OnRun: onReplay,
			},
		},
		{
			Name:        "skip",
			Description: "Skip one or more queued videos",
			Options: []CommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "number",
					Description: "The number of videos to skip",
					Required:    false,
				},
			},
			Handler: &CommandHandler{
				OnRun: onSkip,
			},
		},
		{
			Name:        "volume",
			Description: "Set the bot volume",
			Options: []CommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "number",
					Description: "The volume percentage as an integer",
					Required:    true,
					MinValue:    &minVolume,
					MaxValue:    maxVolume,
				},
			},
			Handler: &CommandHandler{
				OnRun: onSetVolume,
			},
		},
		{
			Name:        "queue",
			Description: "Get the current queue",
			Handler: &CommandHandler{
				OnRun: onQueue,
			},
		},
	}, converter)
}

func onConnect(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	voiceChannelID := event.Options[0].StringValue()
	var voiceChannel *discordgo.Channel

	{
		guildChannels, _ := event.Session.GuildChannels(guildID)

		for _, channel := range guildChannels {
			if channel.ID == voiceChannelID {
				if channel.Type != discordgo.ChannelTypeGuildVoice {
					break
				}

				voiceChannel = channel
			}
		}

		if voiceChannel == nil {
			event.RespondUpdateMsg("The provided channel id is not valid")
			return
		}
	}

	player := music.GetGuildPlayer(guildID)

	go func() {
		err := player.Connect(event.Session, voiceChannelID, event.Data.ChannelID)

		if err != nil {
			event.RespondUpdateMsg(err.Error())
			return
		}

		event.RespondUpdateMsg("Connected to: " + voiceChannel.Name)
	}()
}

func onDisconnect(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	go func() {
		err := player.Disconnect()

		if err != nil {
			event.RespondUpdateMsg(err.Error())
			return
		}

		event.RespondUpdateMsg("Disconnected the bot")
	}()
}

func onPlay(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	search := event.Options[0].StringValue()

	player := music.GetGuildPlayer(guildID)

	if !player.IsConnected() {
		vs, _ := event.Session.State.VoiceState(guildID, event.Data.Interaction.Member.User.ID)

		if vs == nil {
			event.RespondUpdateMsg("You must be connected to a voice channel or use the connect command to stream a video")
			return
		}

		player.Connect(event.Session, vs.ChannelID, event.Data.ChannelID)
	}

	go func() {
		videoInfo, err := player.AddToQueue(event.Data.Member, search)

		if err != nil {
			event.RespondUpdateMsg(err.Error())
			return
		}

		event.RespondUpdate(&ResponseDataUpdate{
			Embeds: &[]*discordgo.MessageEmbed{
				videoInfo.CreateEmbed("Added to Queue", true),
			},
		})
	}()
}

func onSkip(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	numSkip := 1

	if len(event.Options) == 1 {
		numSkip = int(event.Options[0].IntValue())
	}

	player := music.GetGuildPlayer(guildID)

	go func() {
		n, err := player.SkipQueue(numSkip)

		if err != nil {
			event.RespondUpdateMsg(err.Error())
			return
		}

		event.RespondUpdateMsg(fmt.Sprintf("Skipped '%d' videos", n))
	}()
}

func onQueue(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	go func() {
		queue, err := player.GetQueue()

		if err != nil {
			event.RespondUpdateMsg(err.Error())
			return
		}

		if len(queue) == 0 {
			event.RespondUpdateMsg("The queue is empty")
		} else {
			event.RespondUpdateMsg("```" + strings.Join(queue, "\n") + "```")
		}
	}()
}

func onReplay(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	{
		active := player.ToggleReplayMode()

		if active {
			event.RespondUpdateMsg("Replay mode is now enabled. Rock'n'Roll baby!")
		} else {
			event.RespondUpdateMsg("Replay mode is now disabled. At ease soldier!")
		}
	}
}

func onSetVolume(event *Event) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	{
		volume := uint32(event.Options[0].IntValue())
		player.SetVolume(volume)

		event.RespondUpdateMsg(fmt.Sprintf("Player volume set to '%d%%'", volume))
	}
}
