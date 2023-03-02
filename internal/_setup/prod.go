//go:build !dev

package _setup

import (
	"os"
	"path/filepath"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupBase() {
	// Setup Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Setup RootPath
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find running executable path")
	}
	util.RootPath = filepath.Dir(execPath)
}
