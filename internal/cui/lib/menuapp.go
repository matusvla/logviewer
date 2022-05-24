package lib

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jroimartin/gocui"
)

const (
	menubarBGColor       = gocui.ColorDefault
	menubarActiveBGColor = gocui.ColorGreen
)

type (
	WindowManager interface {
		Register(*gocui.Gui) error
		Deregister(*gocui.Gui) error
		Layout(*gocui.Gui) error
	}
	MenuItem struct {
		WindowName    string
		WindowManager WindowManager
	}
	menuItems []MenuItem
)

func (its menuItems) highlightString(highlightIndex int) string {
	var sb strings.Builder
	for i, item := range its {
		sb.WriteString("  ")
		if i == highlightIndex {
			sb.WriteString(BoldString(GreenBGString(item.WindowName)))
			continue
		}
		sb.WriteString(item.WindowName)
	}
	return sb.String()
}

type MenuApp struct {
	items            menuItems
	activeIndex      int
	deregisterMenuFn func(*gocui.Gui) error
}

func NewMenuApp(items []MenuItem) (*MenuApp, error) {
	if len(items) == 0 {
		return nil, errors.New("no items specified for the menu app")
	}
	return &MenuApp{
		items:            items,
		deregisterMenuFn: items[0].WindowManager.Deregister,
	}, nil
}

func (m *MenuApp) Layout(gui *gocui.Gui) error {
	maxX, _ := gui.Size()
	if maxX < 1 {
		return nil // in case that the terminal is not yet initialized we don't do anything
	}
	menuView, err := gui.SetView(MenuBarName, -1, -1, maxX, 1)
	// already set up
	if err == nil {
		currentView := gui.CurrentView()
		if currentView == nil || currentView.Name() == menuView.Name() {
			if currentView == nil {
				if _, err := SetCurrentView(gui, MenuBarName); err != nil {
					return err
				}
			}
			menuView.Clear()
			_, _ = fmt.Fprint(menuView, m.items.highlightString(m.activeIndex))
			menuView.BgColor = menubarActiveBGColor
		} else {
			menuView.Clear()
			_, _ = fmt.Fprint(menuView, m.items.highlightString(m.activeIndex))
			menuView.BgColor = menubarBGColor
		}
		return m.items[m.activeIndex].WindowManager.Layout(gui)
	}
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	if err := SetKeybinding(gui, MenuBarName, gocui.KeyArrowRight, gocui.ModNone, "next window", m.makeUpdateValueFn(1)); err != nil {
		return err
	}
	if err := SetKeybinding(gui, MenuBarName, gocui.KeyArrowLeft, gocui.ModNone, "previous window", m.makeUpdateValueFn(-1)); err != nil {
		return err
	}
	menuView.Frame = false
	return nil
}

func (m *MenuApp) makeUpdateValueFn(scrollBy int) func(g *gocui.Gui, v *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		if err := m.deregisterMenuFn(gui); err != nil {
			return err
		}
		newActiveIndex := m.activeIndex + scrollBy
		if newActiveIndex < 0 {
			newActiveIndex = len(m.items) + scrollBy
		} else {
			newActiveIndex %= len(m.items)
		}
		m.activeIndex = newActiveIndex
		v.Clear()
		_, _ = fmt.Fprint(v, m.items.highlightString(m.activeIndex))
		if err := m.items[newActiveIndex].WindowManager.Register(gui); err != nil {
			return err
		}
		m.deregisterMenuFn = m.items[newActiveIndex].WindowManager.Deregister
		return nil
	}
}
