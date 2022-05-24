package prettyprint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

func PrintFromFile(filePath string, logLvl zerolog.Level, callerWidth int) {
	if err := FprintFromFile(os.Stdout, filePath, logLvl, callerWidth); err != nil {
		fmt.Println(err)
	}
}

func FprintFromFile(writer io.Writer, filePath string, logLvl zerolog.Level, callerWidth int) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	// adding more than default buffer size - if the output reaches the limit it gets stuck with "bufio.Scanner: token too long"
	// see https://stackoverflow.com/questions/21124327/how-to-read-a-text-file-line-by-line-in-go-when-some-lines-are-long-enough-to-ca
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	out := Output{
		log: zerolog.New(&zerolog.ConsoleWriter{
			Out:             writer,
			FormatTimestamp: formatTS,
			FormatFieldName: formatFieldName,
			FormatCaller: func(caller interface{}) string {
				return formatCaller(caller, callerWidth)
			},
		}).Level(logLvl),
	}
	for scanner.Scan() {
		if err := out.ProcessLine(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func StreamFromPipe(cmdString string, logLvl zerolog.Level, callerWidth int) {
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)

	args := strings.Split(cmdString, " ")

	cmd := exec.Command(args[0], args[1:]...)
	cmdReader, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	cmd.Stdout = cmd.Stderr

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	cmdEndCh := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(cmdEndCh)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(cmdReader)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		out := Output{
			log: zerolog.New(&zerolog.ConsoleWriter{
				Out:             os.Stdout,
				FormatTimestamp: formatTS,
				FormatFieldName: formatFieldName,
				FormatCaller: func(caller interface{}) string {
					return formatCaller(caller, callerWidth)
				},
			}).Level(logLvl),
		}
		for scanner.Scan() {
			line := scanner.Text()
			err := out.ProcessLine(line)
			if err != nil {
				fmt.Printf("⇢ %s\n", line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
	}()
	for {
		select {
		case <-cmdEndCh:
			return
		case <-signalC:
			// we want to ignore this as the signal is automatically propagated to the underlying process,
			// and we need to wait for it to end to catch all output
		}
	}
}

type Output struct {
	log zerolog.Logger
}

func NewOutput(writer io.Writer, logLvl zerolog.Level, callerWidth int) Output {
	return Output{
		log: zerolog.New(&zerolog.ConsoleWriter{
			Out: writer,
			FormatTimestamp: func(ts interface{}) string {
				return formatTS(ts)
			},
			FormatFieldName: func(fldName interface{}) string {
				return formatFieldName(fldName)
			},
			FormatCaller: func(caller interface{}) string {
				return formatCaller(caller, callerWidth)
			},
		}).Level(logLvl),
	}
}

func (o *Output) ProcessLine(line string) error {
	const (
		timeFldName   = "time"
		callerFldName = "caller"
		moduleFldName = "module"
	)

	var logItem LogItem
	err := json.Unmarshal([]byte(line), &logItem)
	if err != nil {
		return err
	}
	level, _ := zerolog.ParseLevel(logItem.Level) // we ignore the error - it defaults to no level

	logMsg := o.log.
		WithLevel(level).
		Str(callerFldName, logItem.Caller)

	if mdl := logItem.Module; mdl != "" {
		logMsg = logMsg.Str(moduleFldName, mdl)
	}
	if timestamp := logItem.Timestamp; !timestamp.IsZero() {
		logMsg = logMsg.Time(timeFldName, logItem.Timestamp)
	}
	for fldKey, val := range logItem.Extra {
		logMsg = logMsg.Interface(fldKey, val)
	}

	logMsg.Msg(logItem.Message)
	return nil
}
