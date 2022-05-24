package main

import (
	"fmt"
	"os"
	"path"

	"github.com/matusvla/easyflag"
	"github.com/matusvla/logviewer/internal/viewer"
	"github.com/matusvla/logviewer/pkg/logging"
	"github.com/rs/zerolog"
)

func main() {
	// CLI flags loading and processing
	var cliParams params
	if err := easyflag.ParseAndLoad(&cliParams); err != nil {
		fmt.Printf("CLI flags parsing failed: %s", err.Error())
		os.Exit(1)
	}

	// Setting up the viewer's logger
	logLevel, err := zerolog.ParseLevel(cliParams.LogLevel)
	if err != nil {
		fmt.Printf("invalid log level %q", cliParams.LogLevel)
		os.Exit(1)
	}
	log, logFlushFn := logging.New("viewer", logLevel)
	defer logFlushFn()
	if logLevel != zerolog.NoLevel {
		if err := os.MkdirAll(path.Dir(cliParams.LogPath), os.ModePerm); err != nil {
			log.Fatal().Err(err).Msg("log path directory creation failed")
			os.Exit(1)
		}
		logFile, err := os.Create(cliParams.LogPath)
		if err != nil {
			log.Fatal().Err(err).Msg("log file creation failed")
			os.Exit(1)
		}
		defer logFile.Close()
		log = log.Output(logFile)
	}

	// Running the log viewer
	v, err := viewer.New(log, cliParams.LogPath)
	if err != nil {
		log.Fatal().Err(err).Msg("viewer setup failed")
		os.Exit(1)
	}
	if err := v.Run(); err != nil {
		log.Fatal().Err(err).Msg("viewer running failed")
		os.Exit(2)
	}
	log.Info().Msg("viewer ended")
}
