package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	TimeFormat      = time.RFC3339Nano
	ModuleFieldName = "module"
)

func init() {
	zerolog.TimeFieldFormat = TimeFormat
}

//	New prepares a new zerolog.Logger logging to os.Stderr.
//	It guarantees that no logs will be dropped and will sacrifice performance to achieve this.
//	However, this takes effect onlyin the case of high-frequency logging,
//	otherwise it performs at the same velocity as a zerolog.Logger created using the NewDropping function.
//	The New function returns the zerolog.Logger and a flush function which should be deferred immediately after the call of the New function.
//	This ensures that the program does not finish before all the buffered logs have been written to the output
//	If we do not need this functionality, this function can be ignored.
func New(module string, severity zerolog.Level) (zerolog.Logger, func()) {
	wr := newPW(os.Stderr, false)
	return newLogger(wr, severity).With().Str(ModuleFieldName, module).Logger(), wr.Finalize
}

// NewDropping prepares a new unsafe zerolog.Logger logging to os.Stderr.
// It focuses on the performance and can drop logs without any trace in case of a buffer overflow.
// This can happen when the frequency of the logs is so high that we don't manage to write them to os.Stderr at this speed
// The flush function return value is described above (see New).
func NewDropping(module string, severity zerolog.Level) (zerolog.Logger, func()) {
	wr := newPW(os.Stderr, true)
	return newLogger(wr, severity).With().Str(ModuleFieldName, module).Logger(), wr.Finalize
}

func newLogger(w *parallelWriter, severity zerolog.Level) zerolog.Logger {
	return zerolog.
		New(io.Writer(w)).
		With().
		Caller().
		Timestamp().
		Logger().
		Hook(nanoTsHook{}).
		Hook(zerolog.LevelHook{FatalHook: &fatalHook{w}}).
		Level(severity)
}

// nanoTsHook adds a Unix nanosecond timestamp to the event into a field "ts"
type nanoTsHook struct{}

func (nth nanoTsHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	e.Int64("ts", time.Now().UnixNano())
}

type fatalHook struct {
	w *parallelWriter
}

func (fh *fatalHook) Run(_ *zerolog.Event, _ zerolog.Level, _ string) {
	fh.w.fatalShutdown()
}
