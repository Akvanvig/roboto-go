//go:build !dev

package util

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Note(Fredrico):
	// In dev, we set RootPath to be the executable's directory
	execPath, err := os.Executable()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find running executable path")
	}

	RootPath = filepath.Dir(execPath)
}
