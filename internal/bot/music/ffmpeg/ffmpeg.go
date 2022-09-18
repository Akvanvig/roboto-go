package ffmpeg

import (
	"bufio"
	"context"
	"os/exec"
	"path"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog/log"
)

func CreateStream(ctx context.Context, url string) (*bufio.Reader, error) {
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
