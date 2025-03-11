package player

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type TrackUserData struct {
	ID          snowflake.ID `json:"id"`
	User        string       `json:"username"`
	UserIconURL string       `json:"icon_url"`
	Timestamp   time.Time    `json:"timestamp"`
}

type TrackMessageData struct {
	ChannelID snowflake.ID
	//
	AppID            snowflake.ID
	InteractionToken string
	MessageID        snowflake.ID
}

func fmtDuration(duration lavalink.Duration) string {
	if duration == 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", duration.Minutes(), duration.SecondsPart())
}

func fmtTrack(track lavalink.Track, pos lavalink.Duration) string {
	var txt string
	if pos > 0 {
		txt = fmt.Sprintf("`%s/%s`", fmtDuration(pos), fmtDuration(track.Info.Length))
	} else {
		txt = fmt.Sprintf("`%s`", fmtDuration(track.Info.Length))
	}

	return txt
}

func Message[T *discord.MessageCreate | *discord.MessageUpdate](dst T, txt string, track lavalink.Track, buttons bool) T {

	var embeds []discord.Embed
	{
		var data TrackUserData
		err := json.Unmarshal(track.UserData, &data)
		if err != nil {
			// TODO
		}

		var url string
		if track.Info.URI != nil {
			url = *track.Info.URI
		}

		var thumbnail *discord.EmbedResource
		if track.Info.ArtworkURL != nil {
			thumbnail = &discord.EmbedResource{
				URL: *track.Info.ArtworkURL,
			}
		}

		embeds = []discord.Embed{
			{
				Author: &discord.EmbedAuthor{
					Name:    txt,
					IconURL: "https://media.tenor.com/V0PyK4xovxAAAAAC/peepo-dance-pepe.gif",
				},
				Title:     track.Info.Title,
				URL:       url,
				Thumbnail: thumbnail,
				Fields: []discord.EmbedField{
					{
						Name:  "Uploader",
						Value: track.Info.Author,
					},
					{
						Name:  "Length",
						Value: fmtTrack(track, 0),
					},
				},
				Footer: &discord.EmbedFooter{
					Text:    data.User,
					IconURL: data.UserIconURL,
				},
				Timestamp: &data.Timestamp,
				Color:     0,
			},
		}
	}

	var components []discord.ContainerComponent
	if buttons {
		components = []discord.ContainerComponent{discord.ActionRowComponent{
			//discord.NewPrimaryButton("", "/music/pause_play").WithEmoji(discord.ComponentEmoji{Name: "⏯"}),
			discord.NewPrimaryButton("", "/music/skip").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
			discord.NewPrimaryButton("", "/music/stop").WithEmoji(discord.ComponentEmoji{Name: "⏹"}),
		}}
	}

	switch t := any(dst).(type) {
	case *discord.MessageCreate:
		t.Embeds = embeds
		t.Components = components

	case *discord.MessageUpdate:
		t.Embeds = &embeds
		t.Components = &components
	}

	return dst
}

func AddedToQueue() {

}

func NowPlaying() {

}

func Status() {

}
