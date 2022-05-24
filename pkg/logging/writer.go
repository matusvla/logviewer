package logging

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

type parallelWriter struct {
	io.Writer
	writeCh     chan []byte
	writeOnChFn func([]byte) (int, error)
	write       func([]byte)

	mu            sync.RWMutex
	fatalChecking bool
	closed        bool
	cancelFn      func()
	wg            sync.WaitGroup
}

func newPW(w io.Writer, shouldDrop bool) *parallelWriter {

	writeCh := make(chan []byte, 100)
	writeOnChFn := func(b []byte) (int, error) {
		writeCh <- b
		return len(b), nil
	}
	if shouldDrop {
		writeOnChFn = func(b []byte) (int, error) {
			select {
			case writeCh <- b:
				return len(b), nil
			default:
				return 0, nil
			}
		}
	}

	writeFn := func(logMsg []byte) {
		_, err := w.Write(logMsg)
		if err != nil {
			panic(err)
		}
	}

	pw := parallelWriter{
		Writer:      w,
		writeCh:     writeCh,
		write:       writeFn,
		writeOnChFn: writeOnChFn,
	}

	// start worker - real write
	pw.wg.Add(1)
	go func() {
		defer pw.wg.Done()
		for logMsg := range pw.writeCh {
			pw.write(logMsg)
		}
	}()

	return &pw
}

func (pw *parallelWriter) fatalShutdown() {
	var fatalMsgPrefix = []byte("{\"level\":\"fatal\"")
	pw.mu.Lock()
	defer pw.mu.Unlock()
	prevWriteFn := pw.write
	pw.write = func(b []byte) {
		prevWriteFn(b)
		if bytes.Contains(b, fatalMsgPrefix) {
			os.Exit(10)
		}
	}

	prevWriteOnChFn := pw.writeOnChFn
	pw.writeOnChFn = func(b []byte) (int, error) {
		n, err := prevWriteOnChFn(b)
		if bytes.Contains(b, fatalMsgPrefix) {
			// this unblocks the lock from the Write method
			pw.mu.RUnlock()
			// giving the output writing part of the parallelWriter time to write the messages to output and os.Exit(1)
			time.Sleep(5 * time.Second)
			pw.mu.RLock() // just in case
		}
		return n, err
	}
}

func (pw *parallelWriter) Write(b []byte) (int, error) {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	if !pw.closed {
		bc := make([]byte, len(b))
		copy(bc, b)
		return pw.writeOnChFn(bc) // async
	}
	return 0, nil
}

func (pw *parallelWriter) Finalize() {
	pw.mu.Lock()
	pw.closed = true
	pw.mu.Unlock()
	close(pw.writeCh)
	pw.wg.Wait()
}
