package lib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

type popUp interface {
	name() string
	message() string
}

func noAction() error         { return nil }
func noActionUInt(uint) error { return nil }

var popUpManagerSingleton *PopUpManager

type PopUpManager struct {
	activePopUp popUp
	mu          sync.RWMutex
}

func NewPopUpManager() *PopUpManager {
	if popUpManagerSingleton == nil {
		popUpManagerSingleton = &PopUpManager{}
	}
	return popUpManagerSingleton
}

func (p *PopUpManager) Layout(gui *gocui.Gui) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	lastActiveView := gui.CurrentView()
	if p.activePopUp == nil {
		return nil
	}
	ap := p.activePopUp

	switch pu := ap.(type) {
	case *submitPopUp:
		coordinates := popUpDimensions(pu.messageFld, pu.centerX, pu.centerY, 0)
		x0, y0, x1, y1 := coordinates.Value()
		v, err := gui.SetView(ap.name(), x0, y0, x1, y1)
		// already set up
		if err == nil {
			return nil
		}
		// unexpected error
		if err != gocui.ErrUnknownView {
			return err
		}
		// not yet set up
		if _, err := fmt.Fprint(v, ap.message()); err != nil {
			panic(err)
		}
		if err := SetKeybinding(gui, ap.name(), gocui.KeyTab, gocui.ModNone, "", p.makeSubmitPopUpCleanupFn(noAction, lastActiveView)); err != nil {
			return err
		}
		if err := SetKeybinding(gui, ap.name(), gocui.KeyEsc, gocui.ModNone, "cancel", p.makeSubmitPopUpCleanupFn(noAction, lastActiveView)); err != nil {
			return err
		}
		for _, key := range pu.submitKeys {
			if err := SetKeybinding(gui, ap.name(), key, gocui.ModNone, "submit", p.makeSubmitPopUpCleanupFn(pu.actionFn, lastActiveView)); err != nil {
				return err
			}
		}
		if _, err := SetCurrentView(gui, v.Name()); err != nil {
			return err
		}
		return nil

	case *inputPopUp:
		coordinates := popUpDimensions(pu.messageFld, pu.centerX, pu.centerY, 3)
		x0, y0, x1, y1 := coordinates.Value()
		v, err := gui.SetView(ap.name(), x0, y0, x1, y1)
		// already set up
		if err == nil {
			return nil
		}
		// unexpected error
		if err != gocui.ErrUnknownView {
			return err
		}
		// not yet set up
		if _, err := fmt.Fprint(v, ap.message()); err != nil {
			panic(err)
		}
		input := NewUIntInput(PopUpInput, pu.inputTitle).WithZeroAllowed()
		if err := input.setupView(gui, NewCoordinates(x0+2, y1-3, x1-2, y1-1)); err != nil {
			return err
		}
		if err := SetKeybinding(gui, PopUpInput, gocui.KeyTab, gocui.ModNone, "", p.makeUIntInputPopUpCleanupFn(noActionUInt, input, lastActiveView)); err != nil {
			return err
		}
		if err := SetKeybinding(gui, PopUpInput, gocui.KeyEsc, gocui.ModNone, "cancel", p.makeUIntInputPopUpCleanupFn(noActionUInt, input, lastActiveView)); err != nil {
			return err
		}
		if err := SetKeybinding(gui, PopUpInput, gocui.KeyEnter, gocui.ModNone, "submit", p.makeUIntInputPopUpCleanupFn(pu.actionFn, input, lastActiveView)); err != nil {
			return err
		}
		if _, err := SetCurrentView(gui, PopUpInput); err != nil {
			return err
		}
		return nil

	default:
		panic("unknown popup type")
	}
}

func popUpDimensions(message string, centerX, centerY int, spaceForContentY int) Coordinates {
	var longestMsgLineLen int
	for _, line := range strings.Split(message, "\n") {
		if len(line) > longestMsgLineLen {
			longestMsgLineLen = len(line)
		}
	}
	x0 := centerX - longestMsgLineLen/2 - 1
	x1 := centerX + longestMsgLineLen/2 + longestMsgLineLen%2
	if x0 < 0 {
		x0, x1 = 0, x1-x0
	}
	lineCount := strings.Count(message, "\n") + 1
	y0 := centerY - lineCount/2 - 1
	y1 := centerY + lineCount/2 + lineCount%2 + spaceForContentY
	if y0 < 0 {
		y0, y1 = 0, y1-y0
	}
	return NewCoordinates(x0, y0, x1, y1)
}

func (p *PopUpManager) makeSubmitPopUpCleanupFn(action func() error, lastActiveView *gocui.View) func(g *gocui.Gui, v *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		ap := p.activePopUp
		p.mu.Lock()
		defer p.mu.Unlock()
		actionErr := action()
		DeleteKeybindings(gui, ap.name())
		if err := gui.DeleteView(ap.name()); err != nil {
			return err
		}
		if _, err := SetCurrentView(gui, lastActiveView.Name()); err != nil {
			return err
		}
		p.activePopUp = nil
		return actionErr
	}
}

func (p *PopUpManager) makeUIntInputPopUpCleanupFn(action func(uint) error, input *UIntInput, lastActiveView *gocui.View) func(g *gocui.Gui, v *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		ap := p.activePopUp
		p.mu.Lock()
		defer p.mu.Unlock()
		actionErr := action(uint(input.Value().(int)))
		DeleteKeybindings(gui, ap.name())
		if err := gui.DeleteView(ap.name()); err != nil {
			return err
		}
		if err := gui.DeleteView(input.name); err != nil {
			return err
		}
		DeleteKeybindings(gui, input.name)
		if _, err := SetCurrentView(gui, lastActiveView.Name()); err != nil {
			return err
		}
		p.activePopUp = nil
		return actionErr
	}
}

type submitPopUp struct {
	centerX, centerY    int
	actionFn            func() error
	nameFld, messageFld string
	submitKeys          []gocui.Key
}

func (s *submitPopUp) name() string    { return s.nameFld }
func (s *submitPopUp) message() string { return s.messageFld }

func SubmitPopUp(name, message string, centerX, centerY int, action func() error, submitKeys []gocui.Key) {
	popUpManagerSingleton.mu.Lock()
	popUpManagerSingleton.activePopUp = &submitPopUp{
		centerX:    centerX,
		centerY:    centerY,
		actionFn:   action,
		nameFld:    name,
		messageFld: message,
		submitKeys: submitKeys,
	}
	popUpManagerSingleton.mu.Unlock()
}

type inputPopUp struct {
	centerX, centerY    int
	actionFn            func(uint) error
	nameFld, messageFld string
	inputTitle          string
}

func (s *inputPopUp) name() string    { return s.nameFld }
func (s *inputPopUp) message() string { return s.messageFld }

func UIntInputPopUp(name, inputTitle, message string, centerX, centerY int, action func(uint) error) {
	popUpManagerSingleton.mu.Lock()
	popUpManagerSingleton.activePopUp = &inputPopUp{
		centerX:    centerX,
		centerY:    centerY,
		actionFn:   action,
		nameFld:    name,
		messageFld: message,
		inputTitle: inputTitle,
	}
	popUpManagerSingleton.mu.Unlock()
}
