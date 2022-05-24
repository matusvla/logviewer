package about

import (
	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/lib"
	"github.com/rs/zerolog"
)

type Window struct {
	log                  zerolog.Logger
	layoutManager        *layoutManager
	interactiveViewNames []*lib.ViewFocusData
	activeView           int

	aboutInfo *info
}

func New(log zerolog.Logger, padding lib.Coordinates) *Window {
	return &Window{
		log:           log.With().Str("window", WindowName).Logger(),
		layoutManager: defaultLayout(padding),
		interactiveViewNames: []*lib.ViewFocusData{
			lib.NewViewFocusData(lib.MenuBarName),
		},
		aboutInfo: newAboutInfo(log.With().Str("view", aboutInfoName).Logger()),
	}
}

func (w *Window) Register(gui *gocui.Gui) error {
	w.log.Debug().Msg("registering")
	if err := w.aboutInfo.register(gui); err != nil {
		return err
	}
	if err := lib.ResetGlobalTabKeybinding(gui, w.interactiveViewNames, &w.activeView); err != nil {
		return err
	}
	return w.Layout(gui)
}

func (w *Window) Deregister(gui *gocui.Gui) error {
	w.log.Debug().Msg("deregistering")
	if err := w.aboutInfo.deregister(gui); err != nil {
		return err
	}
	return nil
}

func (w *Window) Layout(gui *gocui.Gui) error {
	w.log.Trace().Msg("laying out")
	maxX, maxY := gui.Size()
	if maxX < 1 || maxY < 1 {
		return nil // in case that the terminal is not yet initialized we don't do anything
	}
	c := w.layoutManager.coordinates(aboutInfoName, maxX, maxY)
	if err := w.aboutInfo.layout(gui, c); err != nil {
		return err
	}
	return nil
}
