package viewer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/matusvla/logviewer/pkg/logging/prettyprint"
	"github.com/rs/zerolog"
)

const (
	maxOffsetListSize  = 5000
	trimOffsetListSize = 1000
)

type logViewer struct {
	log           zerolog.Logger
	file          *os.File
	offsetListMap map[zerolog.Level][]int64
}

func newLogViewer(log zerolog.Logger) *logViewer {
	return &logViewer{
		log:           log,
		offsetListMap: make(map[zerolog.Level][]int64),
	}
}

var levelRe = regexp.MustCompile(`"level":"(trace|debug|info|warn|error|fatal|panic)"`)

func (lv *logViewer) Open(logFilePath string) error {
	f, err := os.Open(logFilePath)
	if err != nil {
		return err
	}
	lv.file = f
	_, err = lv.updateOffsets(0, zerolog.TraceLevel)
	return err
}

func (lv *logViewer) Close() error {
	if lv.file != nil {
		if err := lv.file.Close(); err != nil {
			return err
		}
	}
	lv.offsetListMap = make(map[zerolog.Level][]int64)
	return nil
}

func (lv *logViewer) Get(lineOffsetFromEnd, lineCount int, logLvl zerolog.Level) ([]byte, int, error) {
	if lv.file == nil {
		return nil, 0, errors.New("no file open for get")
	}
	offsetList := lv.offsetListMap[logLvl]
	offsetListLen := len(offsetList)
	if offsetListLen == 0 {
		return nil, 0, fmt.Errorf("no records for the level %s", logLvl.String())
	}
	soIndex := offsetListLen - 1 - lineOffsetFromEnd - lineCount
	eoIndex := offsetListLen - 1 - lineOffsetFromEnd

	if eoIndex < 0 {
		return nil, 0, io.EOF
	}
	if eoIndex > len(offsetList)-1 {
		return nil, 0, io.EOF
	}
	var startOffset int64
	if soIndex > 0 {
		startOffset = offsetList[soIndex]
	}
	endOffset := offsetList[eoIndex]
	b := make([]byte, endOffset-1-startOffset)
	if _, err := lv.file.ReadAt(b, startOffset); err != nil {
		return nil, 0, err
	}

	bb := bytes.NewBuffer([]byte{})
	out := prettyprint.NewOutput(bb, logLvl, 30)
	for _, b := range bytes.Split(b, []byte("\n")) {
		if err := out.ProcessLine(string(b)); err != nil {
			return nil, 0, err
		}
	}
	newLines, err := lv.updateOffsets(lv.offsetListMap[zerolog.TraceLevel][len(lv.offsetListMap[zerolog.TraceLevel])-1], logLvl)
	if err != nil {
		return nil, 0, err
	}
	return bytes.TrimSpace(bb.Bytes()), newLines, nil
}

func (lv *logViewer) updateOffsets(fromOffset int64, newLinesLogLvl zerolog.Level) (int, error) {
	if _, err := lv.file.Seek(fromOffset, io.SeekStart); err != nil {
		return 0, err
	}

	var newLinesCount int
	scanner := bufio.NewScanner(lv.file)
	for scanner.Scan() {
		t := scanner.Bytes()
		fromOffset += int64(len(t)) + 1
		reResult := levelRe.FindSubmatch(t)
		if len(reResult) < 2 {
			continue
		}

		lvl, _ := zerolog.ParseLevel(string(reResult[1]))
		switch lvl {
		case zerolog.PanicLevel:
			lv.offsetListMap[zerolog.PanicLevel] = append(lv.offsetListMap[zerolog.PanicLevel], fromOffset)
			fallthrough
		case zerolog.FatalLevel:
			lv.offsetListMap[zerolog.FatalLevel] = append(lv.offsetListMap[zerolog.FatalLevel], fromOffset)
			fallthrough
		case zerolog.ErrorLevel:
			lv.offsetListMap[zerolog.ErrorLevel] = append(lv.offsetListMap[zerolog.ErrorLevel], fromOffset)
			fallthrough
		case zerolog.WarnLevel:
			lv.offsetListMap[zerolog.WarnLevel] = append(lv.offsetListMap[zerolog.WarnLevel], fromOffset)
			fallthrough
		case zerolog.InfoLevel:
			lv.offsetListMap[zerolog.InfoLevel] = append(lv.offsetListMap[zerolog.InfoLevel], fromOffset)
			fallthrough
		case zerolog.DebugLevel:
			lv.offsetListMap[zerolog.DebugLevel] = append(lv.offsetListMap[zerolog.DebugLevel], fromOffset)
			fallthrough
		default:
			lv.offsetListMap[zerolog.TraceLevel] = append(lv.offsetListMap[zerolog.TraceLevel], fromOffset)
		}
		for i := range lv.offsetListMap {
			if len(lv.offsetListMap[i]) > maxOffsetListSize {
				lv.offsetListMap[i] = lv.offsetListMap[i][trimOffsetListSize:]
			}
		}
		if lvl >= newLinesLogLvl {
			newLinesCount++
		}
	}
	return newLinesCount, nil
}
