package modules

import (
	"fmt"
	"strings"

	. "github.com/Akvanvig/roboto-go/internal/bot/api"
	"github.com/Akvanvig/roboto-go/internal/bot/lib/music"
)

func init() {
	var (
		minVolume = 0.0
		maxVolume = 200.0
		allowDM   = false
	)

	/*
		converter := func(cmd *Command) {
			cmd.Handler.OnRunCheck = func(event *Event) error {
				if event.Data.Interaction.Member == nil {
					return errors.New("You can't play a song in a DM")
				}

				return nil
			}
		}
	*/

	InitChatCommands(&CommandGroupSettings{
		DMPermission: &allowDM,
	}, []CommandOption{
		{
			Name:        "connect",
			Description: "Connect the bot to a voice channel",
			Options: []CommandOption{
				{
					Type:        CommandOptionChannel,
					Name:        "channel",
					Description: "The voice channel",
					ChannelTypes: []ChannelType{
						ChannelTypeGuildVoice,
					},
					Required: true,
				},
			},
			Handler: &CommandHandler{
				OnRun: onConnect,
			},
		},
		{
			Name:        "disconnect",
			Description: "Disconnect the bot from a voice channel",
			Handler: &CommandHandler{
				OnRun: onDisconnect,
			},
		},
		{
			Name:        "play",
			Description: "Play a youtube video",
			Options: []CommandOption{
				{
					Type:        CommandOptionString,
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
					Type:        CommandOptionInteger,
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
					Type:        CommandOptionInteger,
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
	})
}

func onConnect(event *CommandEvent) {
	event.RespondLater()

	player := music.GetGuildPlayer(event.Data.Interaction.GuildID)
	channel := event.Options[0].ChannelValue(event.Session)

	go func() {
		err := player.Connect(event.Session, channel.ID, event.Data.ChannelID)

		if err != nil {
			event.RespondUpdateLaterMsg(err.Error())
			return
		}

		event.RespondUpdateLaterMsg(fmt.Sprintf("Connected to %s", channel.Mention()))
	}()
}

func onDisconnect(event *CommandEvent) {
	event.RespondLater()

	player := music.GetGuildPlayer(event.Data.Interaction.GuildID)

	go func() {
		err := player.Disconnect()

		if err != nil {
			event.RespondUpdateLaterMsg(err.Error())
			return
		}

		event.RespondUpdateLaterMsg("Disconnected the bot")
	}()
}

func onPlay(event *CommandEvent) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	search := event.Options[0].StringValue()

	player := music.GetGuildPlayer(guildID)

	if !player.IsConnected() {
		vs, _ := event.Session.State.VoiceState(guildID, event.Data.Interaction.Member.User.ID)

		if vs == nil {
			event.RespondUpdateLaterMsg("You must be connected to a voice channel or use the connect command to stream a video")
			return
		}

		player.Connect(event.Session, vs.ChannelID, event.Data.ChannelID)
	}

	go func() {
		videoInfo, err := player.AddToQueue(event.Data.Member, search)

		if err != nil {
			event.RespondUpdateLaterMsg(err.Error())
			return
		}

		event.RespondUpdateLater(&ResponseDataUpdate{
			Embeds: &[]*MessageEmbed{
				videoInfo.CreateEmbed("Added to Queue", true),
			},
		})
	}()
}

func onSkip(event *CommandEvent) {
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
			event.RespondUpdateLaterMsg(err.Error())
			return
		}

		event.RespondUpdateLaterMsg(fmt.Sprintf("Skipped '%d' videos", n))
	}()
}

func onQueue(event *CommandEvent) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	go func() {
		queue, err := player.GetQueue()

		if err != nil {
			event.RespondUpdateLaterMsg(err.Error())
			return
		}
		if len(queue) == 0 {
			event.RespondUpdateLaterMsg("The queue is empty")
			return
		}

		event.RespondUpdateLater(&ResponseDataUpdate{
			Embeds: &[]*MessageEmbed{
				{
					Title:       "Video Queue",
					Description: strings.Join(queue, "\n"),
				},
			},
		})
	}()
}

func onReplay(event *CommandEvent) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	{
		active := player.ToggleReplayMode()

		if active {
			event.RespondUpdateLaterMsg("Replay mode is now enabled. Rock'n'Roll baby!")
		} else {
			event.RespondUpdateLaterMsg("Replay mode is now disabled. At ease soldier!")
		}
	}
}

func onSetVolume(event *CommandEvent) {
	event.RespondLater()

	guildID := event.Data.Interaction.GuildID
	player := music.GetGuildPlayer(guildID)

	{
		volume := uint32(event.Options[0].IntValue())
		player.SetVolume(volume)

		event.RespondUpdateLaterMsg(fmt.Sprintf("Player volume set to '%d%%'", volume))
	}
}
