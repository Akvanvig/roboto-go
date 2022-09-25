package music

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/Akvanvig/roboto-go/internal/bot/music/ffmpeg"
	"github.com/Akvanvig/roboto-go/internal/bot/music/youtubedl"
	"github.com/Akvanvig/roboto-go/internal/util"
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

type ConnectionError string

type GuildPlayer struct {
	GuildID string
	// Internal
	mutex       sync.Mutex
	mutexVolume sync.Mutex
	mutexSkip   sync.Mutex
	skipVideo   context.CancelFunc
	stopPlayer  context.CancelFunc

	queue deque.Deque[*youtubedl.BasicVideoInfo]
}

func (player *GuildPlayer) IsConnected() bool {
	player.mutex.Lock()
	defer player.mutex.Unlock()
	return player.stopPlayer != nil
}

func (player *GuildPlayer) Connect(session *discordgo.Session, vcChannelID string, msgChannelID string) error {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stopPlayer != nil {
		return errors.New("Already connected to a channel")
	}

	vc, err := session.ChannelVoiceJoin(player.GuildID, vcChannelID, false, false)

	if err != nil {
		return err
	}

	ctxPlayer, stopPlayer := context.WithCancel(context.Background())
	player.stopPlayer = stopPlayer

	// Core player Loop
	go func() {
		var inactivityTime time.Duration
		timer := time.NewTicker(time.Second * 3)

	loop:
		for {
			select {
			case <-timer.C:
				player.mutex.Lock()

				if player.queue.Len() == 0 {
					inactivityTime += time.Second * 3

					// Break out of the loop and disconnect
					if inactivityTime == (time.Second * 30) {
						break loop
					}

					player.mutex.Unlock()
					break
				}
				videoInfo := player.queue.Front()
				player.mutex.Unlock()
				inactivityTime = 0

				// Update stream link
				videoInfo.Update()
				// Start stream
				player.mutexSkip.Lock()
				ctxVideo, skipVideo := context.WithCancel(context.Background())
				reader, err := ffmpeg.New(ctxVideo, videoInfo.StreamingUrl)

				if err != nil {
					skipVideo()
					player.mutexSkip.Unlock()
					break
				} else {
					player.skipVideo = skipVideo
					player.mutexSkip.Unlock()
				}

				readerBuffered := bufio.NewReaderSize(reader, 16384)

				// Play the stream
				{
					// These represent 1 buffer
					var buffer [OpusFrameSize]byte
					pcmBuffer := util.GetInt16Representation(buffer[:])
					encoder, _ := gopus.NewEncoder(OpusSamplingRate, OpusChannels, gopus.Audio)

					vc.Speaking(true)

					session.ChannelMessageSend(msgChannelID, "Now playing: "+videoInfo.Title)

					for {
						n, err := io.ReadFull(readerBuffered, buffer[:])

						// Finished playing
						if err != nil && n != 0 {
							break
						}

						encodedBuffer, err := encoder.Encode(pcmBuffer[:n/2], OpusSamplesPerFrame, OpusFrameSize)

						if err != nil {
							log.Error().Str("message", "Unexpected encoding error occured").Err(err).Send()
							break
						}

						vc.OpusSend <- encodedBuffer
					}

					session.ChannelMessageSend(msgChannelID, "Finished playing: "+videoInfo.Title)

					vc.Speaking(false)

					player.mutex.Lock()
					player.queue.PopFront()
					player.mutex.Unlock()
				}

				reader.Close()

			case <-ctxPlayer.Done():
				player.mutex.Lock()
				break loop
			}
		}

		timer.Stop()

		player.mutexSkip.Lock()

		if player.skipVideo != nil {
			player.skipVideo()
			player.skipVideo = nil
		}

		player.mutexSkip.Unlock()

		if player.stopPlayer != nil {
			player.stopPlayer()
			player.stopPlayer = nil
		}

		vc.Disconnect()
		player.queue.Clear()

		player.mutex.Unlock()
	}()

	return nil
}

func (player *GuildPlayer) Disconnect() error {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stopPlayer == nil {
		return errors.New("Can't disconnect the bot because it is not connected to a voice channel")
	}

	player.mutexSkip.Lock()
	defer player.mutexSkip.Unlock()

	player.stopPlayer()

	if player.skipVideo != nil {
		player.skipVideo()
	}

	return nil
}

func (player *GuildPlayer) GetQueue() ([]string, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stopPlayer == nil {
		return nil, errors.New("Can't get the queue, as the bot is not connected to a voice channel")
	}

	queueLen := player.queue.Len()
	queueSlice := make([]string, queueLen)

	for queueLen > 0 {
		queueLen -= 1
		queueSlice[queueLen] = player.queue.At(queueLen).Title
	}

	return queueSlice, nil
}

func (player *GuildPlayer) AddToQueue(videoUrl string) (*youtubedl.BasicVideoInfo, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stopPlayer == nil {
		return nil, errors.New("Can't add videos to the queue, as the bot is not connected to a voice channel")
	}

	videoInfo, err := youtubedl.GetVideoInfo(videoUrl)

	if err != nil {
		return nil, errors.New("Could not fetch video information")
	}

	player.queue.PushBack(videoInfo)

	return videoInfo, nil
}

func (player *GuildPlayer) SkipQueue(num int) (int, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stopPlayer == nil {
		return 0, errors.New("Can't skip queued videos, as the bot is not connected to a voice channel")
	}

	player.mutexSkip.Lock()
	defer player.mutexSkip.Unlock()

	if player.skipVideo == nil {
		return 0, errors.New("Can't skip queued videos, as there are none")
	}

	player.skipVideo()
	num -= 1

	if numQueued := player.queue.Len(); numQueued <= num {
		player.queue.Clear()
		return numQueued, nil
	} else {
		numSkipped := num + 1

		for num > 0 {
			player.queue.PopFront()
			num -= 1
		}

		return numSkipped, nil
	}
}

var allGuildPlayers = map[string]*GuildPlayer{}

func GetGuildPlayer(guildID string) *GuildPlayer {
	player, ok := allGuildPlayers[guildID]

	if !ok {
		player = &GuildPlayer{
			GuildID: guildID,
		}
		allGuildPlayers[guildID] = player
	}

	return player
}
