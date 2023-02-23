//go:build dev

package util

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupRuntimeEnvironment() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Note(Fredrico).
	// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Warn().Msg("Dev mode is enabled, do not use this flag in production")

	_, mainPath, _, _ := runtime.Caller(0)
	globals.RootPath = filepath.Join(filepath.Dir(mainPath), "../..")
}

func Assert() {

}
