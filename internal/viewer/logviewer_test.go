package viewer

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogViewer_Get(t *testing.T) {

	type args struct {
		lineOffsetFromEnd, lineCount int
		logLvl                       zerolog.Level
	}
	type want struct {
		output string
		err    error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "2 last lines - trace",
			args: args{
				lineOffsetFromEnd: 0,
				lineCount:         2,
				logLvl:            zerolog.TraceLevel,
			},
			want: want{
				output: "\x1b[90m21:47:55.235_767_000\x1b[0m \x1b[32mINF\x1b[0m ...nternal/viewer/viewer.go:71\x1b[36m > \x1b[0m cui subsystem Run method finished \x1b[34mcomponent=\x1b[0mbackend \x1b[35mmodule=\x1b[0mviewer\n\x1b[90m21:47:55.235_775_000\x1b[0m \x1b[32mINF\x1b[0m ...folio/cmd/viewer/main.go:47\x1b[36m > \x1b[0m viewer ended \x1b[35mmodule=\x1b[0mviewer",
			},
		},
		{
			name: "invalid offset",
			args: args{
				lineOffsetFromEnd: -10,
				lineCount:         2,
				logLvl:            zerolog.TraceLevel,
			},
			want: want{
				output: "",
				err:    io.EOF,
			},
		},
		{
			name: "offset too high",
			args: args{
				lineOffsetFromEnd: 100,
				lineCount:         2,
				logLvl:            zerolog.TraceLevel,
			},
			want: want{
				output: "",
				err:    io.EOF,
			},
		},
		{
			name: "no fatal records found",
			args: args{
				lineOffsetFromEnd: 0,
				lineCount:         2,
				logLvl:            zerolog.FatalLevel,
			},
			want: want{
				output: "",
				err:    errors.New("no records for the level fatal"),
			},
		},
		{
			name: "less records found",
			args: args{
				lineOffsetFromEnd: 0,
				lineCount:         10,
				logLvl:            zerolog.WarnLevel,
			},
			want: want{
				output: "\x1b[90m21:47:55.235_653_000\x1b[0m \x1b[31mWRN\x1b[0m ...olio/internal/cui/cui.go:89\x1b[36m > \x1b[0m turning off gui due to context cancellation \x1b[34mcomponent=\x1b[0mcui \x1b[35mmodule=\x1b[0mviewer\n\x1b[90m21:47:55.235_734_000\x1b[0m \x1b[1m\x1b[31mERR\x1b[0m\x1b[0m ...nternal/viewer/viewer.go:67\x1b[36m > \x1b[0m cui subsystem ended, cancelling context \x1b[34mcomponent=\x1b[0mbackend \x1b[35mmodule=\x1b[0mviewer",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv := newLogViewer(zerolog.New(os.Stdout))
			assert.NoError(t, lv.Open("./testdata/test.log"))
			result, _, err := lv.Get(tt.args.lineOffsetFromEnd, tt.args.lineCount, tt.args.logLvl)
			assert.Equal(t, tt.want.err, err, "error")
			assert.Equal(t, tt.want.output, string(result), "output")
			assert.NoError(t, lv.Close())
		})
	}
}
