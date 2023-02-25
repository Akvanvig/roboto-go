//go:build dev

package util

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Note(Fredrico).
	// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Warn().Msg("Dev mode is enabled, do not use this tag for production")

	_, utilDevPath, _, _ := runtime.Caller(0)
	RootPath = filepath.Join(filepath.Dir(utilDevPath), "../..")
}
