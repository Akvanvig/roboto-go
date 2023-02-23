//go:build !dev

package util

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupRuntimeEnvironment() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Parse()

	// Note(Fredrico):
	// Else, set RootPath to executable path
	execPath, err := os.Executable()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find running executable path")
	}

	globals.RootPath = filepath.Dir(execPath)
}

func Assert() {

}
