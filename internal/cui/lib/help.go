package lib

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

var helpMap = struct {
	items       map[string]map[string]string
	currentView string
	mu          sync.Mutex
}{
	items: make(map[string]map[string]string),
}

func SetKeybinding(gui *gocui.Gui, viewName string, key interface{}, mod gocui.Modifier, helpText string, handler func(*gocui.Gui, *gocui.View) error) error {
	if err := gui.SetKeybinding(viewName, key, mod, handler); err != nil {
		return err
	}
	if helpText == "" {
		return nil
	}

	helpMap.mu.Lock()
	defer helpMap.mu.Unlock()

	view, ok := helpMap.items[viewName]
	if !ok {
		view = make(map[string]string)
	}
	view[keyName(key)] = helpText
	helpMap.items[viewName] = view
	return nil
}

func DeleteKeybinding(gui *gocui.Gui, viewName string, key interface{}, mod gocui.Modifier) error {
	if err := gui.DeleteKeybinding(viewName, key, mod); err != nil {
		return err
	}
	helpMap.mu.Lock()
	defer helpMap.mu.Unlock()
	if _, ok := helpMap.items[viewName]; !ok {
		return nil
	}
	delete(helpMap.items[viewName], keyName(key))
	return nil
}

func DeleteKeybindings(gui *gocui.Gui, viewName string) {
	gui.DeleteKeybindings(viewName)
	helpMap.mu.Lock()
	defer helpMap.mu.Unlock()
	delete(helpMap.items, viewName)
}

func SetCurrentView(gui *gocui.Gui, viewName string) (*gocui.View, error) {
	v, err := gui.SetCurrentView(viewName)
	if err != nil {
		return nil, err
	}
	helpMap.mu.Lock()
	defer helpMap.mu.Unlock()
	helpMap.currentView = viewName
	return v, nil
}

func Close(gui *gocui.Gui) {
	helpMap = struct {
		items       map[string]map[string]string
		currentView string
		mu          sync.Mutex
	}{
		items: make(map[string]map[string]string),
	}

	popUpManagerSingleton = nil
	gui.Close()
}

type Help struct{}

func NewHelp() *Help {
	return &Help{}
}

func (h *Help) Layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	if maxX < 1 || maxY < 1 {
		return nil // in case that the terminal is not yet initialized we don't do anything
	}

	// menu bar
	v, err := gui.SetView("helpView", -1, maxY-3, maxX, maxY)
	if err != nil { // probably a view setup
		if err != gocui.ErrUnknownView {
			return err
		}
		return nil
	}
	v.Frame = false
	v.Clear()
	var viewHelpItems, generalHelpItems []string
	helpMap.mu.Lock()
	defer helpMap.mu.Unlock()
	if helpMap.currentView != "" {
		viewHelp, _ := helpMap.items[helpMap.currentView]
		for k, help := range viewHelp {
			viewHelpItems = append(viewHelpItems, fmt.Sprintf("%v - %s", string(k), help))
		}
	}
	generalHelp, _ := helpMap.items[""]
	for k, help := range generalHelp {
		generalHelpItems = append(generalHelpItems, fmt.Sprintf("%v - %s", string(k), help))
	}
	sort.Strings(viewHelpItems)
	sort.Strings(generalHelpItems)
	_, err = fmt.Fprintf(v, "%s | %s",
		strings.Join(viewHelpItems, ", "),
		GrayFGString(strings.Join(generalHelpItems, ", ")),
	)
	return err
}
