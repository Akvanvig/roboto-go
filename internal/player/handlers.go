package player

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/rs/zerolog/log"
)

func (p *Player) OnDiscordEvent(event bot.Event) {
	switch e := event.(type) {
	case *events.VoiceServerUpdate:
		if e.Endpoint == nil {
			return
		}
		p.Lavalink.OnVoiceServerUpdate(context.Background(), e.GuildID, e.Token, *e.Endpoint)
	case *events.GuildVoiceStateUpdate:
		if e.VoiceState.UserID != e.Client().ApplicationID() {
			return
		}
		p.Lavalink.OnVoiceStateUpdate(context.Background(), e.VoiceState.GuildID, e.VoiceState.ChannelID, e.VoiceState.SessionID)
	}
}

func (p *Player) OnLavalinkEvent(lp disgolink.Player, event lavalink.Event) {
	switch e := event.(type) {
	case lavaqueue.QueueEndEvent:
		log.Debug().Msg("ENDING QUEUE")

		go func() {
			time.Sleep(time.Second * 10)
			track := lp.Track()
			if track == nil {
				ctx := context.Background()
				err := lp.Destroy(ctx)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to stop the music player")
				}

				_ = p.Discord.UpdateVoiceState(ctx, e.GuildID(), nil, false, false)
			}

		}()

	case lavalink.TrackEndEvent:
		track := e.Track

		p.m.Lock()
		defer p.m.Unlock()

		channel := p.PlayingChannels[lp.GuildID()]
		messageID, ok := p.PlayingMessages[track.Info.Identifier]
		if !ok {
			log.Warn().Msgf("Failed to find the corresponding message for track ID '%s'", track.Info.Identifier)
			return
		}

		err := p.Discord.Rest().DeleteMessage(channel, messageID)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to delete the message with ID '%s' in channel ID '%s'", messageID, channel)
		}

		delete(p.PlayingMessages, track.Info.Identifier)

	case lavalink.TrackStartEvent:
		track := e.Track

		p.m.Lock()
		defer p.m.Unlock()

		channel := p.PlayingChannels[lp.GuildID()]
		msg, err := p.Discord.Rest().CreateMessage(channel, *Message(&discord.MessageCreate{}, "Now playing", track, true))

		if err == nil {
			p.PlayingMessages[track.Info.Identifier] = msg.ID
		}

	case lavalink.TrackExceptionEvent:
		// TODO

	case lavalink.TrackStuckEvent:
		// TODO

	}
}
