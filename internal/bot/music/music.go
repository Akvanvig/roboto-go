package music

import (
	"context"
	"encoding/binary"
	"errors"
	"io/fs"
	"sync"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/Akvanvig/roboto-go/internal/util/youtubedl"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog/log"
	"layeh.com/gopus"
)

// Taken from https://github.com/Rapptz/discord.py/blob/master/discord/opus.py
const OpusSamplingRate = 48000
const OpusChannels = 2
const OpusFrameLength = 20
const OpusSampleSize = 2 * OpusChannels
const OpusSamplesPerFrame = int(OpusSamplingRate / 1000 * OpusFrameLength)
const OpusFrameSize = OpusSamplesPerFrame * OpusSampleSize

// Note(Fredrico):
// We need to make the bot disconnect on inactivity.
// We should also make the guild map internal to better synchronize access

type ConnectionError string

type GuildPlayer struct {
	GuildID         string // Required
	Volume          int8   // Optional
	Queue           deque.Deque[string]
	VoiceConnection *discordgo.VoiceConnection
	Ctx             context.Context
	Leave           context.CancelFunc
}

func (err ConnectionError) Error() string {
	return string("A connection error occured: " + err)
}

func (player *GuildPlayer) Connect(session *discordgo.Session, channelID string) error {
	var err error

	if player.VoiceConnection != nil {
		if player.VoiceConnection.ChannelID == channelID {
			return ConnectionError("The bot is already connected to the given voice channel")
		}

		err = player.VoiceConnection.Disconnect()

		if err != nil {
			return err
		}
	}

	vc, err := session.ChannelVoiceJoin(player.GuildID, channelID, false, false)

	if err != nil {
		return err
	}

	_, cancel := context.WithCancel(context.Background())

	/*
		func() {
			var inactivityTime time.Duration
			timer := time.NewTimer(time.Second * 5)

			for {
				select {
				case <-timer.C:
					if player.Queue.Len() == 0 {
						inactivityTime += (time.Second * 5)
						continue
					} else {
						inactivityTime = 0
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	*/

	// Temporary
	player.VoiceConnection = vc
	player.Leave = cancel

	return nil
}

func (player *GuildPlayer) Disconnect() error {
	vc := player.VoiceConnection

	if vc == nil {
		return nil
	}

	player.Leave()
	err := vc.Disconnect()

	if err != nil {
		return err
	}

	return nil
}

func (player *GuildPlayer) AddToQueue(videoInfo *youtubedl.BasicVideoInfo) error {
	ctx, _ := context.WithCancel(context.Background())
	audioReader, err := util.CreateFFmpegStream(ctx, videoInfo.StreamingUrl)

	if err != nil {
		log.Fatal().Err(err).Msg("Fuck 2")
		return errors.New("Tmp2")
	}

	go func() {
		var buffer [OpusSamplesPerFrame * OpusChannels]int16

		encoder, _ := gopus.NewEncoder(OpusSamplingRate, OpusChannels, gopus.Audio)

		player.VoiceConnection.Speaking(true)
		defer player.VoiceConnection.Speaking(false)

		for {
			err := binary.Read(audioReader, binary.LittleEndian, &buffer)

			// Note(Fredrico):
			// A closed pipe means ffmpeg has finished playing
			if _, ok := err.(*fs.PathError); ok {
				break
			}

			if err != nil {
				log.Fatal().Err(err).Msg("BORK!")
			}

			encodedBuffer, err := encoder.Encode(buffer[:], OpusSamplesPerFrame, OpusFrameSize)

			if err != nil {
				log.Fatal().Err(err).Msg("BORK 2!")
			}

			player.VoiceConnection.OpusSend <- encodedBuffer
		}
	}()

	return nil
}

var allGuildPlayers = map[string]*GuildPlayer{}
var allGuildPlayersMutex = sync.Mutex{}

func GetGuildPlayer(guildID string, createIfNotFound bool) *GuildPlayer {
	allGuildPlayersMutex.Lock()

	player, ok := allGuildPlayers[guildID]

	if !ok && createIfNotFound {
		player = &GuildPlayer{
			GuildID: guildID,
		}
		allGuildPlayers[guildID] = player
	}

	allGuildPlayersMutex.Unlock()

	return player
}

func DeleteGuildPlayer(guildID string) error {
	allGuildPlayersMutex.Lock()

	var err error
	player := allGuildPlayers[guildID]

	if player == nil {
		err = errors.New("Something something")
	} else {
		// Note(Fredrico):
		// Maybe disconnect does not have to be mutex locked...
		player.Disconnect()
		delete(allGuildPlayers, guildID)
	}

	allGuildPlayersMutex.Unlock()

	return err
}