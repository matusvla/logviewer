package lib

import (
	"fmt"
	"sync"

	"github.com/jroimartin/gocui"
)

type ButtonWidget struct {
	name    string
	x, y    int
	w       int
	label   string
	handler func(g *gocui.Gui, v *gocui.View) error

	isRegistered    bool
	lastCoordinates Coordinates
	mu              sync.RWMutex
}

func NewButtonWidget(name string, x, y int, label string, handler func(g *gocui.Gui, v *gocui.View) error) *ButtonWidget {
	return &ButtonWidget{name: name, x: x, y: y, w: len(label) + 1, label: label, handler: handler}
}

func (bw *ButtonWidget) Layout(gui *gocui.Gui, coordinates Coordinates) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.lastCoordinates = coordinates
	if !bw.isRegistered {
		return nil
	}
	return bw.setupView(gui, bw.lastCoordinates)
}

func (bw *ButtonWidget) Register(gui *gocui.Gui) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		return bw.setupView(gui, bw.lastCoordinates)
	})
	return nil
}

func (bw *ButtonWidget) Deregister(gui *gocui.Gui) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.isRegistered = false
	if err := gui.DeleteView(bw.name); err != nil {
		return err
	}
	DeleteKeybindings(gui, bw.name)
	return nil
}

func (bw *ButtonWidget) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(bw.name, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		if err := SetKeybinding(gui, bw.name, gocui.KeyEnter, gocui.ModNone, "submit", bw.handler); err != nil {
			return err
		}
		_, _ = fmt.Fprint(v, bw.label)
	}
	return nil
}
