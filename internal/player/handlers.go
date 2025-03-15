package player

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/rs/zerolog/log"
)

func (p *Player) onVoiceServerUpdate(e *events.VoiceServerUpdate) {
	if e.Endpoint != nil {
		p.lavalink.OnVoiceServerUpdate(context.Background(), e.GuildID, e.Token, *e.Endpoint)
	}
}

func (p *Player) onGuildVoiceStateUpdate(e *events.GuildVoiceStateUpdate) {
	if e.VoiceState.UserID == e.Client().ApplicationID() {
		p.lavalink.OnVoiceStateUpdate(context.Background(), e.VoiceState.GuildID, e.VoiceState.ChannelID, e.VoiceState.SessionID)
	}
}

func (p *Player) onTrackStart(lp disgolink.Player, e lavalink.TrackStartEvent) {
	track := e.Track

	p.m.Lock()
	defer p.m.Unlock()

	channelID := p.playingChannels[lp.GuildID()]
	msg, err := p.discord.Rest().CreateMessage(channelID, discord.MessageCreate{
		Embeds:     PlayerEmbedTracks("Now playing", false, track),
		Components: PlayerComponents(false),
	})

	if err == nil {
		p.playingMessages[channelID] = msg.ID
	}
}

func (p *Player) onTrackEnd(lp disgolink.Player, e lavalink.TrackEndEvent) {
	track := e.Track

	p.m.Lock()
	defer p.m.Unlock()

	channelID := p.playingChannels[lp.GuildID()]
	messageID, ok := p.playingMessages[channelID]
	if !ok {
		log.Warn().Msgf("Failed to find the corresponding message for track ID '%s'", track.Info.Identifier)
		return
	}

	err := p.discord.Rest().DeleteMessage(channelID, messageID)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to delete the message with ID '%s' in channel ID '%s'", messageID, channelID)
	}
}

func (p *Player) onTrackException(lp disgolink.Player, e lavalink.TrackExceptionEvent) {
	// A suspicious exception indicates that youtube tried blocking us
	if e.Exception.Severity == lavalink.SeveritySuspicious {
		log.Warn().Msgf("Failed to play the track '%s': %s", e.Track.Info.Title, e.Exception.Error())
	}
}

func (p *Player) onQueueEnd(lp disgolink.Player, e lavaqueue.QueueEndEvent) {
	go func() {
		time.Sleep(time.Second * 10)
		track := lp.Track()
		if track == nil {
			ctx := context.Background()
			_ = p.discord.UpdateVoiceState(ctx, e.GuildID(), nil, false, false)
		}

	}()
}

func (p *Player) onWebSocketClosed(lp disgolink.Player, e lavalink.WebSocketClosedEvent) {
	p.m.Lock()
	defer p.m.Unlock()

	guildID := lp.GuildID()
	channelID := p.playingChannels[guildID]

	delete(p.playingChannels, guildID)
	delete(p.playingMessages, channelID)
}
