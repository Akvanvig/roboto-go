//go:build dev

package _setup

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Note(Fredrico).
	// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Warn().Msg("Dev mode is enabled, do not use this tag for production")

	_, utilDevPath, _, _ := runtime.Caller(0)
	util.RootPath = filepath.Join(filepath.Dir(utilDevPath), "../..")
}
