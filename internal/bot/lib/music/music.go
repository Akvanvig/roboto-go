package music

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Akvanvig/roboto-go/internal/bot/api"
	"github.com/Akvanvig/roboto-go/internal/bot/lib/music/audioop"
	"github.com/Akvanvig/roboto-go/internal/bot/lib/music/ffmpeg"
	"github.com/Akvanvig/roboto-go/internal/bot/lib/music/youtubedl"
	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog/log"
	"layeh.com/gopus"
)

const PlayerDefaultVolume = 50

// PlayerTimeoutSeconds must be divisible by PlayerLoopTickSeconds
const PlayerLoopTickSeconds = 3
const PlayerTimeoutSeconds = 30

// Taken from https://github.com/Rapptz/discord.py/blob/master/discord/opus.py
const OpusSamplingRate = 48000
const OpusChannels = 2
const OpusFrameLength = 20
const OpusSampleSize = 2 * OpusChannels
const OpusSamplesPerFrame = int(OpusSamplingRate / 1000 * OpusFrameLength)
const OpusFrameSize = OpusSamplesPerFrame * OpusSampleSize

var allGuildPlayers = map[string]*GuildPlayer{}

type GuildPlayer struct {
	GuildID string
	// Internal
	channelID        string
	session          *discordgo.Session
	voiceConnection  *discordgo.VoiceConnection
	queue            deque.Deque[*BasicVideoInfo]
	volume           atomic.Uint32
	replayModeActive atomic.Bool
	// Actions
	stop context.CancelFunc
	skip chan bool
	// Mutexes
	mutex sync.Mutex
}

type BasicVideoInfo struct {
	Title        string
	Requestor    string
	RequestedAt  string
	Uploader     string
	Url          string
	RequestorUrl string
	ThumbnailUrl string
	StreamingUrl string
	ChannelUrl   string
	Duration     float64
}

func (player *GuildPlayer) loop(ctx context.Context) {
	timer := time.NewTicker(time.Second * 3)
	var inactivityTime time.Duration

loop:
	for {
		player.mutex.Lock()

		select {
		case <-timer.C:
			if player.queue.Len() == 0 {
				inactivityTime += time.Second * PlayerLoopTickSeconds

				if inactivityTime == (time.Second * PlayerTimeoutSeconds) {
					break loop
				}

				player.mutex.Unlock()
				break
			} else {
				inactivityTime = 0
			}

			videoInfo := player.queue.Front()
			player.mutex.Unlock()

			player.play(videoInfo)

			player.mutex.Lock()
			if player.queue.Len() > 0 && !player.replayModeActive.Load() {
				player.queue.PopFront()
			}
			player.mutex.Unlock()
		case <-ctx.Done():
			break loop
		}
	}

	// Stop player
	player.queue.Clear()
	player.stop()
	player.voiceConnection.Disconnect()

	// Clear state
	player.voiceConnection = nil
	player.stop = nil
	player.skip = nil

	player.mutex.Unlock()
}

func (player *GuildPlayer) play(videoInfo *BasicVideoInfo) {
	// Update stream link
	videoInfo.Update()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := ffmpeg.New(ctx, videoInfo.StreamingUrl)
	defer reader.Close()

	if err != nil {
		return
	}

	if err != nil {
		return
	}

	msg, err := player.sendNowPlaying(videoInfo)

	// Play the stream
	{
		// Buffered reader for ffmpeg
		readerBuffered := bufio.NewReaderSize(reader, 16384)

		// These represent 1 buffer
		var buffer [OpusFrameSize]byte
		pcmBuffer := util.GetInt16Representation(buffer[:])
		encoder, _ := gopus.NewEncoder(OpusSamplingRate, OpusChannels, gopus.Audio)

		player.voiceConnection.Speaking(true)

	stream:
		for {
			select {
			case <-player.skip:
				break stream
			default:
				n, err := io.ReadFull(readerBuffered, buffer[:])

				// Finished playing
				if err != nil && n != 0 {
					break stream
				}

				// Multiply volume if it's not set to the default
				volume := player.volume.Load()

				if volume != 100 {
					audioop.Mul(pcmBuffer, float64(volume)/100.0)
				}

				// Encode frame
				encodedBuffer, err := encoder.Encode(pcmBuffer[:n/2], OpusSamplesPerFrame, OpusFrameSize)

				if err != nil {
					log.Error().Str("message", "Unexpected encoding error occured").Err(err).Send()
					break stream
				}

				player.voiceConnection.OpusSend <- encodedBuffer
			}
		}

		player.voiceConnection.Speaking(false)
	}

	player.session.ChannelMessageDelete(player.channelID, msg.ID)
}

func (player *GuildPlayer) sendNowPlaying(videoInfo *BasicVideoInfo) (*discordgo.Message, error) {
	onButtonClick := func(event *api.ComponentEvent) {
		button := (*event.Component).(api.Button)
		if button.Label == "Show Current Queue" {
			event.RespondLater(discordgo.MessageFlagsEphemeral)
			queue, err := player.GetQueue()

			if err != nil {
				event.RespondUpdateLaterMsg("Failed to retrieve queue")
			} else {
				event.RespondUpdateLater(&api.ResponseDataUpdate{
					Embeds: &[]*api.MessageEmbed{
						{
							Title:       "Video Queue",
							Description: strings.Join(queue, "\n"),
						},
					},
				})
			}
		} else {
			event.RespondLater()
			player.SkipQueue(1)
			event.RespondUpdateLaterMsg("Skipped")
		}
	}

	return player.session.ChannelMessageSendComplex(player.channelID, (&api.MessageSend{
		Embeds: []*api.MessageEmbed{
			videoInfo.CreateEmbed("Now Playing", false),
		},
		Actions: []api.ActionsRow{
			{
				Components: []api.MessageComponent{
					api.Button{
						Label: "Show Current Queue",
						Style: api.SecondaryButton,
					},
					api.Button{
						Emoji: api.ComponentEmoji{
							Name: "⏩",
						},
						Label: "Skip Video/Song",
						Style: api.SecondaryButton,
					},
				},
			},
		},
		Handler: &api.MessageHandler{
			OnComponentSubmit: onButtonClick,
		},
	}).ConvertToOriginal())
}

func (player *GuildPlayer) IsConnected() bool {
	player.mutex.Lock()
	defer player.mutex.Unlock()
	return player.stop != nil
}

func (player *GuildPlayer) Connect(session *discordgo.Session, voiceChannelID string, msgChannelID string) error {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stop != nil {
		return errors.New("Already connected to a channel")
	}

	vc, err := session.ChannelVoiceJoin(player.GuildID, voiceChannelID, false, false)

	if err != nil {
		return err
	}

	player.session = session
	player.voiceConnection = vc
	player.channelID = msgChannelID

	ctx, cancel := context.WithCancel(context.Background())
	player.stop = cancel
	player.skip = make(chan bool, 1)

	// Go go go
	go player.loop(ctx)

	return nil
}

func (player *GuildPlayer) Disconnect() error {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stop == nil {
		return errors.New("Can't disconnect the bot because it is not connected to a voice channel")
	}

	player.stop()
	player.skip <- true

	return nil
}

func (player *GuildPlayer) GetQueue() ([]string, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stop == nil {
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

func (player *GuildPlayer) AddToQueue(requestor *discordgo.Member, search string) (*BasicVideoInfo, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stop == nil {
		return nil, errors.New("Can't add videos to the queue, as the bot is not connected to a voice channel")
	}

	rawInfo, err := youtubedl.GetVideoInfo(search)

	if err != nil {
		return nil, err
	}

	videoInfo := &BasicVideoInfo{
		Title:        rawInfo.Title,
		Requestor:    requestor.User.Username,
		RequestedAt:  time.Now().Format(time.RFC3339),
		Uploader:     rawInfo.Uploader,
		Url:          rawInfo.WebpageURL,
		RequestorUrl: requestor.AvatarURL(""),
		ThumbnailUrl: rawInfo.Thumbnail,
		ChannelUrl:   rawInfo.ChannelUrl,
		Duration:     rawInfo.Duration,
	}

	player.queue.PushBack(videoInfo)

	return videoInfo, nil
}

func (player *GuildPlayer) SkipQueue(num int) (int, error) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.stop == nil {
		return 0, errors.New("Can't skip queued videos, as the bot is not connected to a voice channel")
	}

	if player.queue.Len() == 0 {
		return 0, errors.New("Can't skip queued videos, as there are none")
	}

	if player.replayModeActive.Load() {
		return 0, errors.New("Can't skip videos when replay move is active")
	}

	player.skip <- true
	num -= 1

	if num == 0 {
		return 1, nil
	}

	numSkipped := num + 1

	for player.queue.Len() > 1 && num > 0 {
		player.queue.PopFront()
		num -= 1
		numSkipped += 1
	}

	return numSkipped, nil
}

func (player *GuildPlayer) ToggleReplayMode() bool {
	replayActive := !player.replayModeActive.Load()
	player.replayModeActive.Store(replayActive)
	return replayActive
}

func (player *GuildPlayer) SetVolume(percentage uint32) {
	player.volume.Store(percentage)
}

func (videoInfo *BasicVideoInfo) Update() {
	videoInfo.StreamingUrl, _ = youtubedl.FetchYoutubeVideoStreamingUrl(videoInfo.Url)
}

func (videoInfo *BasicVideoInfo) CreateEmbed(title string, simple bool) *discordgo.MessageEmbed {
	var builder strings.Builder

	{
		// Format duration
		hours := int(videoInfo.Duration / 3600)
		minutes := int(videoInfo.Duration/60) % 60
		seconds := int(videoInfo.Duration) % 60

		if hours > 0 {
			fmt.Fprintf(&builder, "%d Hours, ", hours)
		}
		if minutes > 0 {
			fmt.Fprintf(&builder, "%d Minutes, ", minutes)
		}
		fmt.Fprintf(&builder, "%d Seconds", seconds)
	}

	var timestamp string
	var iconUrl string
	var fields []*discordgo.MessageEmbedField
	var footer *discordgo.MessageEmbedFooter

	if !simple {
		timestamp = videoInfo.RequestedAt
		iconUrl = "https://media.tenor.com/V0PyK4xovxAAAAAC/peepo-dance-pepe.gif"
		fields = []*discordgo.MessageEmbedField{
			{
				Name:  "Uploader",
				Value: videoInfo.Uploader,
			},
			{
				Name:  "Length",
				Value: builder.String(),
			},
		}
		footer = &discordgo.MessageEmbedFooter{
			Text:    videoInfo.Requestor,
			IconURL: videoInfo.RequestorUrl,
		}
	}

	// Return embedded message
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    title,
			IconURL: iconUrl,
		},
		Title: videoInfo.Title,
		URL:   videoInfo.Url,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: videoInfo.ThumbnailUrl,
		},
		Fields:    fields,
		Footer:    footer,
		Timestamp: timestamp,
		// WHITE WHITE WHITE
		Color: 16777215,
	}
}

func GetGuildPlayer(guildID string) *GuildPlayer {
	player, ok := allGuildPlayers[guildID]

	if !ok {
		volume := atomic.Uint32{}
		volume.Store(PlayerDefaultVolume)

		player = &GuildPlayer{
			GuildID:          guildID,
			volume:           volume,
			replayModeActive: atomic.Bool{},
		}
		allGuildPlayers[guildID] = player
	}

	return player
}
