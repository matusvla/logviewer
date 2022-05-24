package lib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

type ChannelViewer struct {
	name, title     string
	fullContents    strings.Builder
	isRegistered    bool
	lastCoordinates Coordinates
	mu              sync.RWMutex
}

func NewChannelViewer(name, title string) *ChannelViewer {
	return &ChannelViewer{
		name:            name,
		title:           title,
		lastCoordinates: NewCoordinates(0, 0, 1, 1),
	}
}

func (cv *ChannelViewer) Layout(gui *gocui.Gui, coordinates Coordinates) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.lastCoordinates = coordinates
	if !cv.isRegistered {
		return nil
	}
	return cv.setupView(gui, coordinates)
}

func (cv *ChannelViewer) Register(gui *gocui.Gui) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		return cv.setupView(gui, cv.lastCoordinates)
	})
	cv.appendLine(gui, strings.TrimSpace(cv.fullContents.String()), true)
	return nil
}

func (cv *ChannelViewer) Deregister(gui *gocui.Gui) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.isRegistered = false
	if err := gui.DeleteView(cv.name); err != nil {
		return err
	}
	DeleteKeybindings(gui, cv.name)
	return nil
}

func (cv *ChannelViewer) Listen(gui *gocui.Gui, ch <-chan fmt.Stringer) {
	for msg := range ch {
		cv.mu.RLock()
		msgString := msg.String()
		if cv.isRegistered {
			cv.appendLine(gui, msgString, false)
		} else {
			cv.fullContents.WriteString(msgString + "\n")
		}
		cv.mu.RUnlock()
	}
}

func (cv *ChannelViewer) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(cv.name, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	v.Title = cv.title
	v.Autoscroll = true
	v.Wrap = true
	if err := SetKeybinding(gui, cv.name, gocui.KeyArrowUp, gocui.ModNone, "scroll up",
		func(g *gocui.Gui, v *gocui.View) error {
			return ScrollView(v, -1)
		}); err != nil {
		return err
	}
	if err := SetKeybinding(gui, cv.name, gocui.KeyArrowDown, gocui.ModNone, "scroll down",
		func(g *gocui.Gui, v *gocui.View) error {
			return ScrollView(v, 1)
		}); err != nil {
		return err
	}
	if err := SetKeybinding(gui, cv.name, 'a', gocui.ModNone, "toggle autoscroll",
		func(g *gocui.Gui, v *gocui.View) error {
			v.Autoscroll = true
			return nil
		}); err != nil {
		return err
	}
	if err := SetKeybinding(gui, cv.name, gocui.MouseWheelDown, gocui.ModNone, "scroll down",
		func(g *gocui.Gui, v *gocui.View) error {
			return ScrollView(v, 1)
		}); err != nil {
		return err
	}
	if err := SetKeybinding(gui, cv.name, gocui.MouseWheelUp, gocui.ModNone, "scroll up",
		func(g *gocui.Gui, v *gocui.View) error {
			return ScrollView(v, -1)
		}); err != nil {
		return err
	}
	return nil
}

func (cv *ChannelViewer) appendLine(gui *gocui.Gui, msgString string, isReplay bool) {
	gui.Update(func(gui *gocui.Gui) error {
		v, err := gui.View(cv.name)
		if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			return nil // this might happen at startup - gui is started before the view is properly set up
		}
		if msgString != "" {
			if !isReplay {
				_, _ = fmt.Fprintln(&cv.fullContents, msgString)
			}
			_, _ = fmt.Fprintln(v, msgString)
		}
		return nil
	})
}
