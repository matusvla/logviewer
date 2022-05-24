package viewer

import (
	"context"
	"sync"

	"github.com/matusvla/logviewer/internal/cui"
	"github.com/matusvla/logviewer/internal/model"
	"github.com/rs/zerolog"
)

type Viewer struct {
	log      zerolog.Logger
	logPath  string
	logReqCh chan *model.LogRequest
	cui      *cui.GuiViewer

	runWg sync.WaitGroup
}

func New(
	log zerolog.Logger,
	logPath string,
) (*Viewer, error) {
	logReqCh := make(chan *model.LogRequest)

	cuiViewer, err := cui.New(
		log.With().Str("component", "cui").Logger(),
		logPath,
		logReqCh,
	)
	if err != nil {
		return nil, err
	}

	return &Viewer{
		log:      log.With().Str("component", "backend").Logger(),
		logPath:  logPath,
		logReqCh: logReqCh,
		cui:      cuiViewer,
		runWg:    sync.WaitGroup{},
	}, nil
}

func (v *Viewer) Run() error {
	v.log.Info().Str("component", "backend").Msg("starting viewer")
	v.runWg.Add(1)
	defer v.runWg.Done()

	ctx, cancelFn := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	// Run listening for all incoming events
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := v.runLogViewer(ctx); err != nil {
			v.log.Error().Err(err).Msg("runLogViewer ended, cancelling context")
			cancelFn()
		}
	}()
	v.log.Info().Msg("cui backend started")
	wg.Add(1)
	go func() {
		defer wg.Done()
		v.cui.Run(ctx)
		v.log.Info().Msg("cui subsystem ended, cancelling context")
		cancelFn()
	}()
	wg.Wait()
	v.log.Info().Msg("cui subsystem Run method finished")
	return nil
}

func (v *Viewer) runLogViewer(ctx context.Context) error {
	log := v.log.With().Str("worker", "runLogViewer").Logger()
	log.Info().Msg("started")
	defer log.Info().Msg("ended")
	lv := newLogViewer(log)
	defer func() {
		if err := lv.Close(); err != nil {
			log.Error().Err(err).Msg("log viewer closing failed")
		}
	}()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("stopped due to context cancellation")
			return nil
		case logRequest, ok := <-v.logReqCh:
			if !ok {
				panic("reqCh unexpectedly closed")
			}

			switch body := logRequest.Body.(type) {
			case *model.OpenLogRequestBody:
				if err := lv.Close(); err != nil {
					log.Error().Err(err).Msg("log viewer closing failed")
				}
				respErr := lv.Open(body.FilePath)
				logRequest.RespCh <- &model.LogRequestResponse{Err: respErr}
			case *model.GetLogRequestBody:
				respBody, newLines, respErr := lv.Get(body.OffsetFromEnd, body.LineCount, body.LogLvl)
				logRequest.RespCh <- &model.LogRequestResponse{
					Body:     respBody,
					NewLines: newLines,
					Err:      respErr,
				}
			default:
				panic("unexpected log request type")
			}
		}
	}
}
