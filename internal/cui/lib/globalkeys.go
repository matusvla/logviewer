package lib

import "github.com/jroimartin/gocui"

func SetGlobalQuitKeybinding(gui *gocui.Gui) error {
	return SetKeybinding(gui, "", gocui.KeyCtrlC, gocui.ModNone, "exit whole application", func(gui *gocui.Gui, _ *gocui.View) error {
		return gocui.ErrQuit
	})
}

type ViewFocusData struct {
	name      string
	hasCursor bool
}

func NewViewFocusData(name string) *ViewFocusData {
	return &ViewFocusData{name: name}
}

func (vfd *ViewFocusData) WithCursor() *ViewFocusData {
	vfd.hasCursor = true
	return vfd
}

func ResetGlobalTabKeybinding(gui *gocui.Gui, interactiveViews []*ViewFocusData, activeView *int) error {
	_ = DeleteKeybinding(gui, "", gocui.KeyTab, gocui.ModNone) // we want to ignore the error - it just means that no keybinding was found yet
	if err := SetKeybinding(gui, "", gocui.KeyTab, gocui.ModNone, "next element", func(gui *gocui.Gui, _ *gocui.View) error {
		newActiveView := *activeView
		// if some of the interactive fields are hidden (Unregistered)	, we want to skip them - we are looking for the first one that is visible
		for {
			newActiveView = (newActiveView + 1) % len(interactiveViews)
			viewFocusInfo := interactiveViews[newActiveView]
			_, err := SetCurrentView(gui, viewFocusInfo.name)
			if err == nil {
				gui.Cursor = viewFocusInfo.hasCursor
				if _, err := gui.SetViewOnTop(viewFocusInfo.name); err != nil {
					return err
				}
				break
			}
			if err != gocui.ErrUnknownView {
				return err
			}
		}
		*activeView = newActiveView
		return nil
	}); err != nil {
		return err
	}
	return SetKeybinding(gui, "", gocui.MouseLeft, gocui.ModNone, "select element", func(gui *gocui.Gui, view *gocui.View) error {
		viewName := view.Name()
		newActiveView := -1
		for i, vn := range interactiveViews {
			if vn.name == viewName {
				newActiveView = i
				gui.Cursor = vn.hasCursor
				break
			}
		}
		if newActiveView == -1 {
			return nil
		}
		*activeView = newActiveView
		if _, err := SetCurrentView(gui, viewName); err != nil {
			return err
		}
		if _, err := gui.SetViewOnTop(viewName); err != nil {
			return err
		}
		return nil
	})
}
