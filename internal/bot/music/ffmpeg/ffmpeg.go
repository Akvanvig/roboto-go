package ffmpeg

import (
	"context"
	"io"
	"os/exec"
	"path"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog/log"
)

func New(ctx context.Context, url string) (io.ReadCloser, error) {
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
		"-reconnect",
		"1",
		"-reconnect_streamed",
		"1",
		"-reconnect_delay_max",
		"5",
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
		"-vn",
		"pipe:1",
	)

	reader, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	err = cmd.Start()

	if err != nil {
		reader.Close()
		return nil, err
	}

	return reader, nil
}
