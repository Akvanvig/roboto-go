//go:build dev

package _setup

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "net/http/pprof"
)

func setupBase() {
	// Setup Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Setup RootPath
	_, utilDevPath, _, _ := runtime.Caller(0)
	util.RootPath = filepath.Join(filepath.Dir(utilDevPath), "../..")

	// Setup Profiler
	log.Debug().Msg("Starting pprof websocket")
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	log.Debug().Msg("Dev mode is enabled. Do not use this tag in production")

}
