package cui

import (
	"context"
	"errors"
	"sync"

	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/about"
	"github.com/matusvla/logviewer/internal/cui/lib"
	"github.com/matusvla/logviewer/internal/cui/logs"
	"github.com/matusvla/logviewer/internal/model"
	"github.com/rs/zerolog"
)

type GuiViewer struct {
	log zerolog.Logger
	gui *gocui.Gui

	appManager         *lib.MenuApp
	deregisterWindowFn func(gui *gocui.Gui) error
}

func New(
	log zerolog.Logger,
	logPath string,
	logReqCh chan *model.LogRequest,
) (*GuiViewer, error) {
	gui, err := gocui.NewGui(gocui.Output256)
	gui.InputEsc = true
	gui.Highlight = true
	gui.SelFgColor = gocui.ColorGreen

	if err != nil {
		return nil, err
	}

	padding := lib.NewCoordinates(0, 2, 0, 2)
	logsWindow := logs.New(padding, logPath, logReqCh)
	aboutWindow := about.New(log, padding)
	menuApp, err := lib.NewMenuApp([]lib.MenuItem{
		{logs.WindowName, logsWindow},
		{about.WindowName, aboutWindow},
	})
	if err != nil {
		return nil, err
	}

	gui.SetManager(
		menuApp,
		lib.NewHelp(),
		lib.NewPopUpManager(),
	)
	if err := lib.SetGlobalQuitKeybinding(gui); err != nil {
		return nil, err
	}
	gui.Mouse = true
	if err := logsWindow.Register(gui); err != nil {
		return nil, err
	}

	return &GuiViewer{
		log:                log,
		gui:                gui,
		appManager:         menuApp,
		deregisterWindowFn: logsWindow.Deregister,
	}, nil
}

func (gv *GuiViewer) Run(ctx context.Context) {
	var runWg sync.WaitGroup
	cuiCtx, cuiCtxCancelFn := context.WithCancel(ctx)

	runWg.Add(1)
	go func() {
		defer runWg.Done()
		defer cuiCtxCancelFn()
		if err := gv.gui.MainLoop(); err != nil {
			if errors.Is(err, gocui.ErrQuit) {
				gv.log.Info().Msg("gui main loop ended")
				return
			}
			gv.log.Error().Err(err).Msg("gui main loop ended with an error")
		}
	}()

	<-cuiCtx.Done()
	gv.gui.Mouse = false
	gv.log.Info().Msg("turning off gui due to context cancellation")
	runWg.Wait()
	lib.Close(gv.gui)
}
