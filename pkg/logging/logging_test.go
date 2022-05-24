package logging

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestLogger(t *testing.T) {
	log, logWaitFn := New("test", zerolog.TraceLevel)
	defer logWaitFn()

	log.Trace().Send()
	log.Debug().Msg("debug")
	log.Info().Msg("info")
	log.Warn().Msg("warn")
	log.Error().Err(errors.New("test error")).Msg("debug")
	//log.Fatal().Msg("fatal")
	defer func() { recover() }() // so that the log.Panic() won't fail the test
	log.Panic().Msg("OH NO!")
	t.Fail() // We don't want to get here ever
}

func BenchmarkLoggingNonDropping(b *testing.B) {
	log, logWaitFn := New("test", zerolog.DebugLevel)
	defer logWaitFn()
	for i := 0; i < b.N; i++ {
		log.Debug().Int("i", i).Msg("mesmes")
	}
}

func BenchmarkLoggingDropping(b *testing.B) {
	log, logWaitFn := NewDropping("test", zerolog.DebugLevel)
	defer logWaitFn()
	for i := 0; i < b.N; i++ {
		log.Debug().Int("i", i).Msg("mesmes")
	}
}

// this test is only used to manually verify that the waitFn behaves as it is supposed to
func TestLogging(t *testing.T) {
	log, logWaitFn := New("test", zerolog.DebugLevel)
	defer logWaitFn()
	for i := 0; i < 20000; i++ {
		log.Debug().Int("i", i).Msg("mesmes")
	}
	defer func() { recover() }() // so that the log.Panic() won't fail the test
	log.Panic().Msg("OH NO!")
}
