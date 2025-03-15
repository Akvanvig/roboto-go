package player

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
)

func fmtDuration(duration lavalink.Duration) string {
	if duration == 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", duration.Minutes(), duration.SecondsPart())
}

func fmtTrackDuration(track lavalink.Track, pos lavalink.Duration) string {
	var txt string
	if pos > 0 {
		txt = fmt.Sprintf("`%s/%s`", fmtDuration(pos), fmtDuration(track.Info.Length))
	} else {
		txt = fmt.Sprintf("`%s`", fmtDuration(track.Info.Length))
	}

	return txt
}

func Embeds(title string, simple bool, tracks ...lavalink.Track) []discord.Embed {
	embed := discord.Embed{
		Author: &discord.EmbedAuthor{
			Name:    title,
			IconURL: "https://media.tenor.com/V0PyK4xovxAAAAAC/peepo-dance-pepe.gif",
		},
		Color: 0,
	}

	num := len(tracks)
	if num == 1 {
		track := tracks[0]

		var data TrackUserData
		json.Unmarshal(track.UserData, &data)

		embed.Title = track.Info.Title

		if track.Info.URI != nil {
			embed.URL = *track.Info.URI
		}

		if track.Info.ArtworkURL != nil {
			embed.Thumbnail = &discord.EmbedResource{
				URL: *track.Info.ArtworkURL,
			}
		}

		if data.User != "" {
			embed.Footer = &discord.EmbedFooter{
				Text:    data.User,
				IconURL: data.UserIconURL,
			}
			embed.Timestamp = &data.Timestamp
		}

		if !simple {
			embed.Fields = []discord.EmbedField{
				{
					Name:  "Uploader",
					Value: track.Info.Author,
				},
				{
					Name:  "Length",
					Value: fmtTrackDuration(track, 0),
				},
			}
		}
	} else if num > 1 {
		var b strings.Builder

		numChars := 0
		for i, track := range tracks {
			var tmpB strings.Builder
			var data TrackUserData

			json.Unmarshal(track.UserData, &data)

			tmpB.WriteString(strconv.Itoa(i + 1))
			tmpB.WriteString(". [")
			tmpB.WriteString(track.Info.Title)
			tmpB.WriteString("](")
			tmpB.WriteString(*track.Info.URI)
			tmpB.WriteString(") (")
			tmpB.WriteString(fmtTrackDuration(track, 0))

			if data.User != "" {
				tmpB.WriteString(") (")
				tmpB.WriteString(data.User)
			}

			tmpB.WriteString(")\n")

			// Disscord message limit is 4000ish chars
			str := tmpB.String()
			tmpNumChars := numChars + utf8.RuneCountInString(str)
			if tmpNumChars < 4000 {
				b.WriteString(str)
				numChars = tmpNumChars
			} else {
				b.WriteString(".....")
				break
			}
		}

		embed.Description = b.String()
	}

	return []discord.Embed{embed}
}

func Components(queueEmpty bool) []discord.ContainerComponent {
	components := []discord.ContainerComponent{discord.ActionRowComponent{
		discord.NewPrimaryButton("Skip", "/music/skip").WithEmoji(discord.ComponentEmoji{Name: "👉"}).WithDisabled(queueEmpty),
		discord.NewPrimaryButton("Queue", "/music/queue").WithEmoji(discord.ComponentEmoji{Name: "👏"}).WithStyle(discord.ButtonStyleSecondary).WithDisabled(queueEmpty),
		discord.NewPrimaryButton("Stop", "/music/stop").WithEmoji(discord.ComponentEmoji{Name: "👋"}).WithStyle(discord.ButtonStyleDanger),
	}}

	return components
}
