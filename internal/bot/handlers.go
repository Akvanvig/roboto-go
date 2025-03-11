package bot

import (
	"context"
	"encoding/json"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/mroctopus/bottie-bot/internal/player"
	"github.com/rs/zerolog/log"
)

func (b *RobotoBot) OnDiscordEvent(event bot.Event) {
	switch e := event.(type) {
	case *events.VoiceServerUpdate:
		if e.Endpoint == nil {
			return
		}
		b.Lavalink.OnVoiceServerUpdate(context.Background(), e.GuildID, e.Token, *e.Endpoint)
	case *events.GuildVoiceStateUpdate:
		if e.VoiceState.UserID != e.Client().ApplicationID() {
			return
		}
		b.Lavalink.OnVoiceStateUpdate(context.Background(), e.VoiceState.GuildID, e.VoiceState.ChannelID, e.VoiceState.SessionID)
	}
}

// NOTE:
// This might need a mutex -> !!!!
func (b *RobotoBot) OnLavalinkEvent(p disgolink.Player, event lavalink.Event) {
	client := b.Discord

	switch e := event.(type) {
	case lavaqueue.QueueEndEvent:
		log.Debug().Msgf("LAVALINK EVENT: %s", event.Type())

		go func() {
			time.Sleep(time.Second * 10)
			track := p.Track()
			if track == nil {
				ctx := context.Background()

				err := p.Destroy(ctx)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to stop the music player")
				}

				_ = client.UpdateVoiceState(ctx, e.GuildID(), nil, false, false)
			}

		}()

	case lavalink.TrackEndEvent:
		track := e.Track

		var data player.TrackUserData
		err := json.Unmarshal(track.UserData, &data)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal track TrackUserData on Lavalink.TrackEndEvent")
			return
		}

		msg, ok := b.LavalinkTrackMessages[data.ID]
		if !ok {
			log.Warn().Msgf("Failed to find the corresponding message for track snowflake ID '%s'", data.ID)
			return
		}

		err = client.Rest().DeleteFollowupMessage(msg.AppID, msg.InteractionToken, msg.MessageID)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to delete the message with ID '%s' in channel ID '%s'", msg.MessageID, msg.ChannelID)
		}

		delete(b.LavalinkTrackMessages, data.ID)

	case lavalink.TrackStartEvent:
		// TODO

	case lavalink.TrackExceptionEvent:
		// TODO

	case lavalink.TrackStuckEvent:
		// TODO

	}
}
