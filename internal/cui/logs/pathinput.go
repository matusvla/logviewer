package logs

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/lib"
)

type pathInput struct {
	value    string
	callback func(*gocui.Gui, string)

	isRegistered    bool
	lastCoordinates lib.Coordinates
	mu              sync.RWMutex
}

func newPathInput(logPath string, callback func(*gocui.Gui, string)) *pathInput {
	of := pathInput{
		value:           logPath,
		callback:        callback,
		lastCoordinates: lib.NewCoordinates(0, 0, 1, 1),
	}
	return &of
}

func (pi *pathInput) layout(gui *gocui.Gui, coordinates lib.Coordinates) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.lastCoordinates = coordinates
	if !pi.isRegistered {
		return nil
	}
	return pi.setupView(gui, coordinates)
}

func (pi *pathInput) register(gui *gocui.Gui) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		return pi.setupView(gui, pi.lastCoordinates)
	})
	return nil
}

func (pi *pathInput) deregister(gui *gocui.Gui) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.isRegistered = false
	if err := gui.DeleteView(pathInputName); err != nil {
		return err
	}
	lib.DeleteKeybindings(gui, pathInputName)
	return nil
}

func (pi *pathInput) setupView(gui *gocui.Gui, coordinates lib.Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(pathInputName, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	v.Title = "Logfile path"
	v.Editable = true
	//v.Overwrite = true

	if err := lib.SetKeybinding(gui, pathInputName, gocui.KeyEnter, gocui.ModNone, "submit",
		func(g *gocui.Gui, v *gocui.View) error {
			pi.value = v.Buffer()[:len(v.Buffer())-1] // remove newline
			pi.callback(g, pi.value)
			return nil
		}); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, pathInputName, gocui.KeyArrowRight, gocui.ModNone, "autocomplete",
		func(g *gocui.Gui, v *gocui.View) error {
			pi.value = v.Buffer()[:len(v.Buffer())-1] // remove trailing space
			dir, file := pi.value, ""
			if pi.value[len(pi.value)-1] != os.PathSeparator {
				dir, file = path.Split(pi.value)
			}
			fi, err := os.ReadDir(dir)
			if err != nil {
				// todo log
				return nil // we did not get a valid directory - no change to the name
			}
			if foundFileName := filterFileNames(fi, file); foundFileName != "" {
				pi.value = path.Join(dir, foundFileName)
				if foundFileName[len(foundFileName)-1] == os.PathSeparator {
					pi.value += string(os.PathSeparator)
				}
				v.Clear()
				_, _ = fmt.Fprint(v, pi.value)
				_ = v.SetCursor(len(pi.value), 0)
			}
			return nil
		}); err != nil {
		return err
	}
	if err := lib.SetKeybinding(gui, pathInputName, gocui.KeyCtrlU, gocui.ModNone, "clean",
		func(g *gocui.Gui, v *gocui.View) error {
			pi.value = ""
			v.Clear()
			_ = v.SetCursor(0, 0)
			return nil
		}); err != nil {
		return err
	}
	_, _ = fmt.Fprint(v, pi.value)
	return nil
}

// todo test this
func filterFileNames(dirEntries []os.DirEntry, prefix string) string {
	var longestCommonPrefix string
	var wasLCMSet bool
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		if strings.HasPrefix(name, prefix) {
			if !wasLCMSet && longestCommonPrefix == "" {
				longestCommonPrefix = name
				wasLCMSet = true
				continue
			}
			lcmLength := 0
			for i := range longestCommonPrefix {
				if i == len(name) || longestCommonPrefix[i] != name[i] {
					break
				}
				lcmLength++
			}
			longestCommonPrefix = longestCommonPrefix[:lcmLength]
		}
	}
	return longestCommonPrefix
}
