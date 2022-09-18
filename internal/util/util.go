package util

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog/log"
)

func GetValidUrl(urlString string) (string, error) {
	url, err := url.Parse(urlString)

	if err != nil {
		return "", err
	}

	if url.Scheme == "" {
		return url.RequestURI(), nil
	} else {
		return url.Host + url.RequestURI(), nil
	}
}

func CreateFFmpegStream(ctx context.Context, url string) (*bufio.Reader, error) {
	var ffmpegPath string

	switch globals.OS {
	case "windows":
		ffmpegPath = path.Join(globals.RootPath, "ffmpeg.exe")
	case "linux":
		ffmpegPath = path.Join(globals.RootPath, "ffmpeg")
	default:
		log.Fatal().Msg("Trying to run ffmpeg on an unsupported system")
	}

	cmd := exec.CommandContext(ctx,
		ffmpegPath,
		"-vn",
		"-i",
		url,
		"-f",
		"s16le",
		"-ar",
		"48000",
		"-ac",
		"2",
		"-loglevel",
		"warning",
		"-reconnect",
		"1",
		"-reconnect_streamed",
		"1",
		"-reconnect_delay_max",
		"5",
		"pipe:1",
	)

	writer, err := cmd.StdinPipe()

	if err != nil {
		writer.Close()
		return nil, err
	}

	reader, err := cmd.StdoutPipe()

	if err != nil {
		writer.Close()
		reader.Close()
		return nil, err
	}

	bufferedReader := bufio.NewReaderSize(reader, 16384)

	err = cmd.Start()

	go func() {
		cmd.Wait()
	}()

	if err != nil {
		return nil, err
	}

	return bufferedReader, nil
}

func GetAudioLink(url string) (audioUrl string, err error) {
	var (
		ytdlPath string
		cmd      []byte
	)

	switch globals.OS {
	case "windows":
		ytdlPath = path.Join(globals.RootPath, "yt-dlp.exe")
	case "linux":
		ytdlPath = path.Join(globals.RootPath, "yt-dlp")
	default:
		log.Fatal().Msg("Trying to run youtube-dl on an unsupported system")
	}

	if cmd, err = exec.Command(
		ytdlPath,
		"--get-url",
		"-f",
		"bestaudio/best",
		"-o",
		"%(extractor)s-%(id)s-%(title)s.%(ext)s",
		"--restrict-filenames",
		"--no-playlist",
		"--no-check-certificate",
		"--ignore-errors",
		"--quiet",
		"--no-warnings",
		"--age-limit",
		"20",
		"--default-search",
		"auto",
		"--source-address",
		"0.0.0.0",
		url,
	).Output(); err != nil {
		return audioUrl, fmt.Errorf("calling youtube-dl: %w", err)
	}

	audioUrl = strings.TrimSpace(string(cmd))
	return audioUrl, err
}
