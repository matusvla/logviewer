package prettyprint

import (
	"fmt"
	"strings"
	"time"

	"github.com/matusvla/logviewer/pkg/logging"
)

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorMagenta
	colorBlue
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

const (
	callerSeparatorMark = "\x1b[36m > \x1b[0m"
	noTimeString        = "--:--:--.---_---_---"
	secondTimeFormat    = "15:04:05"
)

func formatTS(ts interface{}) (result string) {
	defer func() { // format result - color gray
		result = fmt.Sprintf("\x1b[%dm%v\x1b[0m", colorDarkGray, result)
	}()

	tsStr, ok := ts.(string)
	if !ok {
		return noTimeString
	}
	t, err := time.Parse(logging.TimeFormat, tsStr)
	if err != nil {
		return tsStr
	}
	timestamp := t.Format(secondTimeFormat)
	nsAll := t.Nanosecond()
	ms := (nsAll / 1_000_000) % 1000
	us := (nsAll / 1000) % 1000
	ns := nsAll % 1000
	return fmt.Sprintf("%s.%03d_%03d_%03d", timestamp, ms, us, ns)
}

func formatFieldName(ts interface{}) string {
	fldName, ok := ts.(string)
	if !ok {
		return ""
	}
	if fldName == logging.ModuleFieldName {
		return fmt.Sprintf("\x1b[%dm%v=\x1b[0m", colorBlue, fldName)
	}
	return fmt.Sprintf("\x1b[%dm%v=\x1b[0m", colorMagenta, fldName)
}

func formatCaller(caller interface{}, width int) string {
	const placeholderChar = "_"
	value, ok := caller.(string)
	if !ok {
		return strings.Repeat(placeholderChar, width)
	}
	n := len(value)
	switch {
	case n < width:
		return strings.Repeat(placeholderChar, width-n) + value + callerSeparatorMark
	case n == width:
		return value + callerSeparatorMark
	default:
		return "..." + value[n-width+3:n] + callerSeparatorMark
	}
}
