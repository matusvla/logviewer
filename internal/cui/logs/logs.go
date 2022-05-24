package logs

import (
	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/lib"
	"github.com/matusvla/logviewer/internal/model"
)

type Window struct {
	layoutManager        *layoutManager
	interactiveViewNames []*lib.ViewFocusData
	activeView           int

	pathInput *pathInput
	logViewer *viewer
}

func New(padding lib.Coordinates, logPath string, logReqCh chan *model.LogRequest) *Window {
	logViewer := newViewer(logReqCh)
	return &Window{
		layoutManager: defaultLayout(padding),
		interactiveViewNames: []*lib.ViewFocusData{
			lib.NewViewFocusData(lib.MenuBarName),
			lib.NewViewFocusData(pathInputName).WithCursor(),
			lib.NewViewFocusData(logViewerName),
		},
		pathInput: newPathInput(logPath, logViewer.requestLogFile),
		logViewer: logViewer,
	}
}

func (w *Window) Register(gui *gocui.Gui) error {
	if err := w.logViewer.register(gui); err != nil {
		return err
	}
	if err := w.pathInput.register(gui); err != nil {
		return err
	}
	if err := lib.ResetGlobalTabKeybinding(gui, w.interactiveViewNames, &w.activeView); err != nil {
		return err
	}
	return w.Layout(gui)
}

func (w *Window) Deregister(gui *gocui.Gui) error {
	if err := w.logViewer.deregister(gui); err != nil {
		return err
	}
	if err := w.pathInput.deregister(gui); err != nil {
		return err
	}
	return nil
}

func (w *Window) Layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	if maxX < 1 || maxY < 1 {
		return nil // in case that the terminal is not yet initialized we don't do anything
	}
	if err := w.pathInput.layout(gui, w.layoutManager.coordinates(pathInputName, maxX, maxY)); err != nil {
		return err
	}
	if err := w.logViewer.layout(gui, w.layoutManager.coordinates(logViewerName, maxX, maxY)); err != nil {
		return err
	}
	return nil
}
