package logs

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/lib"
	"github.com/matusvla/logviewer/internal/model"
	"github.com/rs/zerolog"
)

type viewer struct {
	level zerolog.Level

	isRegistered    bool
	lastCoordinates lib.Coordinates
	mu              sync.RWMutex
	logRequestCh    chan *model.LogRequest
	offset          int

	isFollowing       bool
	followWg          sync.WaitGroup
	followCtxCancelFn context.CancelFunc
}

func newViewer(logReqCh chan *model.LogRequest) *viewer {
	return &viewer{
		level:           zerolog.TraceLevel,
		logRequestCh:    logReqCh,
		lastCoordinates: lib.NewCoordinates(0, 0, 1, 1),
	}
}

func (vw *viewer) requestLogFile(gui *gocui.Gui, logPath string) {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	vw.level = zerolog.TraceLevel
	vw.offset = 0

	// open request
	respCh := make(chan *model.LogRequestResponse)
	vw.logRequestCh <- &model.LogRequest{
		Body:   &model.OpenLogRequestBody{FilePath: logPath},
		RespCh: respCh,
	}
	if err := (<-respCh).Err; err != nil {
		gui.Update(func(gui *gocui.Gui) error {
			return vw.setupView(gui, vw.lastCoordinates, []byte(err.Error()))
		})
		return
	}
	_, sy := gui.Size() // this ensures that we load enough data when loading the log file for the first time
	_, _ = vw.getLogData(gui, 0, sy, zerolog.TraceLevel)
}

func (vw *viewer) getLogData(gui *gocui.Gui, offset, lineCount int, level zerolog.Level) (int, bool) {
	respCh := make(chan *model.LogRequestResponse)
	vw.logRequestCh <- &model.LogRequest{
		Body: &model.GetLogRequestBody{
			OffsetFromEnd: offset,
			LineCount:     lineCount,
			LogLvl:        level,
		},
		RespCh: respCh,
	}
	resp := <-respCh
	msg := resp.Body
	if err := resp.Err; err != nil {
		if errors.Is(err, io.EOF) {
			return 0, false
		}
		msg = []byte(resp.Err.Error())
	}
	gui.Update(func(gui *gocui.Gui) error {
		return vw.setupView(gui, vw.lastCoordinates, msg)
	})
	return resp.NewLines, true
}

func (vw *viewer) layout(gui *gocui.Gui, coordinates lib.Coordinates) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	vw.lastCoordinates = coordinates
	if !vw.isRegistered {
		return nil
	}
	return vw.setupView(gui, coordinates, nil)
}

func (vw *viewer) register(gui *gocui.Gui) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	vw.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		return vw.setupView(gui, vw.lastCoordinates, nil)
	})
	return nil
}

func (vw *viewer) deregister(gui *gocui.Gui) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	if vw.followCtxCancelFn != nil {
		vw.followCtxCancelFn()
	}
	vw.followWg.Wait()
	vw.isRegistered = false
	if err := gui.DeleteView(logViewerName); err != nil {
		return err
	}
	lib.DeleteKeybindings(gui, logViewerName)
	return nil
}

func (vw *viewer) setupView(gui *gocui.Gui, coordinates lib.Coordinates, contents []byte) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(logViewerName, x0, y0, x1, y1)
	// already set up
	if err == nil {
		if contents != nil {
			v.Clear()
			if err := v.SetOrigin(0, 0); err != nil {
				return err
			}
			if _, err := v.Write(contents); err != nil {
				return err
			}
		}
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	v.Title = "Console logs"
	v.Wrap = true
	v.Autoscroll = true
	if err := lib.SetKeybinding(gui, logViewerName, 'a', gocui.ModNone, "toggle autoscroll",
		func(g *gocui.Gui, v *gocui.View) error {
			vw.mu.Lock()
			defer vw.mu.Unlock()
			if !vw.isFollowing {

				if err := lib.DeleteKeybinding(gui, logViewerName, gocui.MouseWheelUp, gocui.ModNone); err != nil {
					panic(err)
				}
				if err := lib.DeleteKeybinding(gui, logViewerName, gocui.KeyArrowUp, gocui.ModNone); err != nil {
					panic(err)
				}
				if err := lib.DeleteKeybinding(gui, logViewerName, gocui.MouseWheelDown, gocui.ModNone); err != nil {
					panic(err)
				}
				if err := lib.DeleteKeybinding(gui, logViewerName, gocui.KeyArrowDown, gocui.ModNone); err != nil {
					panic(err)
				}
				ctx, cancelFn := context.WithCancel(context.Background())
				vw.followCtxCancelFn = cancelFn
				vw.followWg.Add(1)
				go func() {
					defer vw.followWg.Done()
					t := time.NewTicker(500 * time.Millisecond)
					for {
						select {
						case <-ctx.Done():
							return
						case <-t.C:
							_, sy := v.Size()
							_, _ = vw.getLogData(g, 0, sy, vw.level)
						}
					}
				}()
			} else {
				vw.followCtxCancelFn()
				vw.followWg.Wait()
				vw.followCtxCancelFn = nil
				if err := lib.SetKeybinding(gui, logViewerName, gocui.KeyArrowDown, gocui.ModNone, "scroll down", vw.scrollDown); err != nil {
					panic(err)
				}
				if err := lib.SetKeybinding(gui, logViewerName, gocui.MouseWheelDown, gocui.ModNone, "scroll down", vw.scrollDown); err != nil {
					panic(err)
				}
				if err := lib.SetKeybinding(gui, logViewerName, gocui.KeyArrowUp, gocui.ModNone, "scroll up", vw.scrollUp); err != nil {
					return err
				}
				if err := lib.SetKeybinding(gui, logViewerName, gocui.MouseWheelUp, gocui.ModNone, "scroll up", vw.scrollUp); err != nil {
					return err
				}
			}
			vw.isFollowing = !vw.isFollowing
			return nil
		}); err != nil {
		return err
	}

	if err := lib.SetKeybinding(gui, logViewerName, gocui.KeyArrowDown, gocui.ModNone, "scroll down", vw.scrollDown); err != nil {
		panic(err)
	}
	if err := lib.SetKeybinding(gui, logViewerName, gocui.MouseWheelDown, gocui.ModNone, "scroll down", vw.scrollDown); err != nil {
		panic(err)
	}
	if err := lib.SetKeybinding(gui, logViewerName, gocui.KeyArrowUp, gocui.ModNone, "scroll up", vw.scrollUp); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, gocui.MouseWheelUp, gocui.ModNone, "scroll up", vw.scrollUp); err != nil {
		return err
	}

	if err := lib.SetKeybinding(gui, logViewerName, 't', gocui.ModNone, "trace", vw.buildSetLevelFn(zerolog.TraceLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'd', gocui.ModNone, "debug", vw.buildSetLevelFn(zerolog.DebugLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'i', gocui.ModNone, "info", vw.buildSetLevelFn(zerolog.InfoLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'w', gocui.ModNone, "warn", vw.buildSetLevelFn(zerolog.WarnLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'e', gocui.ModNone, "error", vw.buildSetLevelFn(zerolog.ErrorLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'f', gocui.ModNone, "fatal", vw.buildSetLevelFn(zerolog.FatalLevel)); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, logViewerName, 'p', gocui.ModNone, "panic", vw.buildSetLevelFn(zerolog.PanicLevel)); err != nil {
		return err
	}
	return nil
}

func (vw *viewer) buildSetLevelFn(level zerolog.Level) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		vw.mu.Lock()
		vw.mu.Unlock()
		vw.level = level
		vw.offset = 0
		_, sy := v.Size()
		vw.getLogData(g, vw.offset, sy, level)
		return nil
	}
}

func (vw *viewer) scrollUp(g *gocui.Gui, v *gocui.View) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	_, sy := v.Size()
	newLines, ok := vw.getLogData(g, vw.offset+1, sy, vw.level)
	if ok {
		vw.offset += 1 + newLines
	}
	return nil
}

func (vw *viewer) scrollDown(g *gocui.Gui, v *gocui.View) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	if vw.offset-1 < 0 {
		return nil
	}
	_, sy := v.Size()
	newLines, ok := vw.getLogData(g, vw.offset-1, sy, vw.level)
	if ok {
		vw.offset--
		vw.offset += newLines
	}
	return nil
}
