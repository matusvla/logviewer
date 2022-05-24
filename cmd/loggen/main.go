package main

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/matusvla/easyflag"
	"github.com/matusvla/logviewer/pkg/logging"
	"github.com/rs/zerolog"
)

//go:embed loremipsum.txt
var loremIpsum string

func main() {
	// CLI flags loading and processing
	var cliParams params
	if err := easyflag.ParseAndLoad(&cliParams); err != nil {
		fmt.Printf("CLI flags parsing failed: %s", err.Error())
		return
	}

	// Setting up the logger
	log, logFlushFn := logging.New("loggen", zerolog.TraceLevel)
	defer logFlushFn()

	outPath := cliParams.OutputPath
	if err := os.MkdirAll(path.Dir(outPath), os.ModePerm); err != nil {
		fmt.Printf("log path directory %q creation failed", outPath)
		return
	}
	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("Log file creation on path %q failed: %s", outPath, err.Error())
		return
	}
	defer outFile.Close()
	log = log.Output(outFile)

	// Generate the logs
	loremIpsumSentences := strings.Split(loremIpsum, "\n")
	percentStep := float64(cliParams.LogCount) / 100.0
	for i := uint(0); i < cliParams.LogCount; i++ {
		if percentStep < 1 || i%uint(percentStep) == 0 {
			fmt.Print("\033[<100D")
			fmt.Printf("\rPrinting logs: %d%%", int(float64(i)/percentStep))
		}
		log.WithLevel(zerolog.Level(i%5 - 1)).Msg(loremIpsumSentences[i%uint(len(loremIpsumSentences))])
		time.Sleep(cliParams.SleepDuration)
	}
	fmt.Printf("\rSuccessfully generated %d log lines into file %s", cliParams.LogCount, outPath)
}
