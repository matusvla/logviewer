package about

import (
	_ "embed"
	"sync"
	"text/template"

	"github.com/jroimartin/gocui"
	"github.com/matusvla/logviewer/internal/cui/lib"
	"github.com/rs/zerolog"
)

//go:embed LICENSE_symlink
var licenseText string

var aboutInfoTemplate = template.Must(template.New("aboutInfo").Parse(
	`Log viewer

{{ .LicenseText }}
`))

type info struct {
	log             zerolog.Logger
	isRegistered    bool
	lastCoordinates lib.Coordinates
	mu              sync.RWMutex
}

func newAboutInfo(log zerolog.Logger) *info {
	return &info{
		log:             log,
		lastCoordinates: lib.NewCoordinates(0, 0, 1, 1),
	}
}

func (in *info) layout(gui *gocui.Gui, coordinates lib.Coordinates) error {
	in.log.Trace().Stringer("coordinates", coordinates).Msg("laying out")
	in.mu.RLock()
	defer in.mu.RUnlock()
	in.lastCoordinates = coordinates
	if !in.isRegistered {
		in.log.Trace().Stringer("coordinates", coordinates).Msg("skiping layout - unregistered view")
		return nil
	}
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(aboutInfoName, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		return nil
	}
	v.Title = "About"
	v.Clear()
	err = aboutInfoTemplate.Execute(v, struct {
		LicenseText string
	}{
		LicenseText: licenseText,
	})
	if err != nil {
		panic(err)
	}
	return nil
}

func (in *info) register(gui *gocui.Gui) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.isRegistered = true
	x0, y0, x1, y1 := in.lastCoordinates.Value()
	v, err := gui.SetView(aboutInfoName, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		return nil // this might happen at startup - gui is started before the view is properly set up
	}
	v.Clear()
	return nil
}

func (in *info) deregister(gui *gocui.Gui) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.isRegistered = false
	return gui.DeleteView(aboutInfoName)
}
