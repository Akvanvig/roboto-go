package youtubedl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog/log"
)

/*
Copyright (c) 2019 Mattias Wadman

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Note(Fredrico):
// A lot of this code is inspired by https://github.com/wader/goutubedl

// Info youtube-dl info
type Info struct {
	ID                 string  `json:"id"`                   // Video identifier
	Title              string  `json:"title"`                // Video title
	URL                string  `json:"url"`                  // Video URL
	AltTitle           string  `json:"alt_title"`            // A secondary title of the video
	DisplayID          string  `json:"display_id"`           // An alternative identifier for the video
	Uploader           string  `json:"uploader"`             // Full name of the video uploader
	Creator            string  `json:"creator"`              // The creator of the video
	ReleaseDate        string  `json:"release_date"`         // The date (YYYYMMDD) when the video was released
	Timestamp          float64 `json:"timestamp"`            // UNIX timestamp of the moment the video became available
	UploadDate         string  `json:"upload_date"`          // Video upload date (YYYYMMDD)
	UploaderID         string  `json:"uploader_id"`          // Nickname or id of the video uploader
	Channel            string  `json:"channel"`              // Full name of the channel the video is uploaded on
	ChannelID          string  `json:"channel_id"`           // Id of the channel
	Duration           float64 `json:"duration"`             // Length of the video in seconds
	ViewCount          float64 `json:"view_count"`           // How many users have watched the video on the platform
	LikeCount          float64 `json:"like_count"`           // Number of positive ratings of the video
	DislikeCount       float64 `json:"dislike_count"`        // Number of negative ratings of the video
	CommentCount       float64 `json:"comment_count"`        // Number of comments on the video
	AgeLimit           float64 `json:"age_limit"`            // Age restriction for the video (years)
	IsLive             bool    `json:"is_live"`              // Whether this video is a live stream or a fixed-length video
	StartTime          float64 `json:"start_time"`           // Time in seconds where the reproduction should start, as specified in the URL
	EndTime            float64 `json:"end_time"`             // Time in seconds where the reproduction should end, as specified in the URL
	Playlist           string  `json:"playlist"`             // Name or id of the playlist that contains the video
	PlaylistIndex      float64 `json:"playlist_index"`       // Index of the video in the playlist padded with leading zeros according to the total length of the playlist
	PlaylistID         string  `json:"playlist_id"`          // Playlist identifier
	PlaylistTitle      string  `json:"playlist_title"`       // Playlist title
	PlaylistUploader   string  `json:"playlist_uploader"`    // Full name of the playlist uploader
	PlaylistUploaderID string  `json:"playlist_uploader_id"` // Nickname or id of the playlist uploader

	// Available for the media that is a track or a part of a music album:
	Track       string  `json:"track"`        // Title of the track
	TrackNumber float64 `json:"track_number"` // Number of the track within an album or a disc
	TrackID     string  `json:"track_id"`     // Id of the track
	Artist      string  `json:"artist"`       // Artist(s) of the track
	Genre       string  `json:"genre"`        // Genre(s) of the track
	Album       string  `json:"album"`        // Title of the album the track belongs to
	AlbumType   string  `json:"album_type"`   // Type of the album
	AlbumArtist string  `json:"album_artist"` // List of all artists appeared on the album
	DiscNumber  float64 `json:"disc_number"`  // Number of the disc or other physical medium the track belongs to
	ReleaseYear float64 `json:"release_year"` // Year (YYYY) when the album was released

	Type        string `json:"_type"`
	Direct      bool   `json:"direct"`
	WebpageURL  string `json:"webpage_url"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	// not unmarshalled, populated from image thumbnail file
	ThumbnailBytes []byte      `json:"-"`
	Thumbnails     []Thumbnail `json:"thumbnails"`

	Formats   []Format              `json:"formats"`
	Subtitles map[string][]Subtitle `json:"subtitles"`

	// Playlist entries if _type is playlist
	Entries []Info `json:"entries"`

	// Info can also be a mix of Info and one Format
	Format
}

type Thumbnail struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Preference int    `json:"preference"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Resolution string `json:"resolution"`
}

type Format struct {
	URL            string            `json:"url"`             // Url of video format
	Ext            string            `json:"ext"`             // Video filename extension
	Format         string            `json:"format"`          // A human-readable description of the format
	FormatID       string            `json:"format_id"`       // Format code specified by `--format`
	FormatNote     string            `json:"format_note"`     // Additional info about the format
	Width          float64           `json:"width"`           // Width of the video
	Height         float64           `json:"height"`          // Height of the video
	Resolution     string            `json:"resolution"`      // Textual description of width and height
	TBR            float64           `json:"tbr"`             // Average bitrate of audio and video in KBit/s
	ABR            float64           `json:"abr"`             // Average audio bitrate in KBit/s
	ACodec         string            `json:"acodec"`          // Name of the audio codec in use
	ASR            float64           `json:"asr"`             // Audio sampling rate in Hertz
	VBR            float64           `json:"vbr"`             // Average video bitrate in KBit/s
	FPS            float64           `json:"fps"`             // Frame rate
	VCodec         string            `json:"vcodec"`          // Name of the video codec in use
	Container      string            `json:"container"`       // Name of the container format
	Filesize       float64           `json:"filesize"`        // The number of bytes, if known in advance
	FilesizeApprox float64           `json:"filesize_approx"` // An estimate for the number of bytes
	Protocol       string            `json:"protocol"`        // The protocol that will be used for the actual download
	HTTPHeaders    map[string]string `json:"http_headers"`
}

// Subtitle youtube-dl subtitle
type Subtitle struct {
	URL      string `json:"url"`
	Ext      string `json:"ext"`
	Language string `json:"-"`
	// not unmarshalled, populated from subtitle file
	Bytes []byte `json:"-"`
}

type BasicVideoInfo struct {
	Title        string
	Url          string
	StreamingUrl string
}

func (videoInfo *BasicVideoInfo) Update() {
	videoInfo.StreamingUrl, _ = fetchYoutubeVideoStreamingUrl(videoInfo.Url)
}

func youtubedlPath() string {
	var ytdlPath string

	switch globals.OS {
	case "windows":
		ytdlPath = filepath.Join(globals.RootPath, "yt-dlp.exe")
	case "linux":
		ytdlPath = filepath.Join(globals.RootPath, "yt-dlp")
	default:
		log.Fatal().Msg("Trying to run youtube-dl on an unsupported system")
	}

	return ytdlPath
}

func fetchYoutubeVideoStreamingUrl(url string) (string, error) {
	cmd := exec.Command(
		youtubedlPath(),
		"--get-url",
		"--ignore-errors",
		"--no-cache-dir",
		"--restrict-filenames",
		"--no-playlist",
		"--no-check-certificate",
		"--quiet",
		"--no-warnings",
		"-f",
		"bestaudio/best",
		url,
	)

	output, err := cmd.Output()

	if err != nil {
		return "", errors.New("No video found with the given link")
	}

	return strings.TrimSpace(string(output)), nil
}

func fetchYoutubeVideoInfo(search string) (*Info, error) {
	cmd := exec.Command(
		youtubedlPath(),
		"--ignore-errors",
		"--no-cache-dir",
		"--default-search",
		"ytsearch",
		"--skip-download",
		"--restrict-filenames",
		"--no-playlist",
		"-J",
		search,
	)

	buffer := bytes.NewBufferString("")
	bufferErr := bytes.NewBufferString("")

	cmd.Stdout = buffer
	cmd.Stderr = bufferErr

	err := cmd.Run()

	if err != nil {
		return nil, errors.New("No video found with the given link")
	}

	// Note(Fredrico):
	// This should probably be improved somewhat, but it works for now
	if bufferErr.Len() != 0 {
		errorScanner := bufio.NewScanner(bufferErr)

		for errorScanner.Scan() {
			const errorPrefix = "ERROR: "

			if strings.HasPrefix(errorScanner.Text(), errorPrefix) {
				return nil, errors.New("Something especially weird happened")
			}
		}
	}

	info := Info{}
	err = json.Unmarshal(buffer.Bytes(), &info)

	if err != nil {
		return nil, err
	}

	if info.Type != "playlist" {
		return &info, nil
	}

	if len(info.Entries) == 0 {
		return nil, errors.New("No search results found")
	}

	return &info.Entries[0], nil
}

func GetVideoInfo(search string) (*BasicVideoInfo, error) {
	rawInfo, err := fetchYoutubeVideoInfo(search)

	if err != nil {
		return nil, err
	}

	return &BasicVideoInfo{
		Title: rawInfo.Title,
		Url:   rawInfo.WebpageURL,
	}, nil
}
