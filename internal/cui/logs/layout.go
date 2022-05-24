package logs

import (
	"github.com/matusvla/logviewer/internal/cui/lib"
)

type layoutManager struct {
	padding     lib.Coordinates
	ohViewCount int
}

func defaultLayout(padding lib.Coordinates) *layoutManager {
	return &layoutManager{
		padding: padding,
	}
}

func (l layoutManager) coordinates(viewName string, maxX, maxY int) lib.Coordinates {
	px0, py0, px1, py1 := l.padding.Value()
	maxX -= px0 + px1
	maxY -= py0 + py1
	var x0, y0, x1, y1 int
	switch viewName {
	case pathInputName:
		x0, y0, x1, y1 = 0, 0, maxX-1, 2
	case logViewerName:
		x0, y0, x1, y1 = 0, 3, maxX-1, maxY-1
	default:
		panic("unknown view")
	}
	x0 += px0
	y0 += py0
	x1 += px0
	y1 += py0
	if x0 >= x1 || y0 >= y1 || x0 < 0 || y0 < 0 {
		return lib.NewCoordinates(0, 0, 1, 1)
	}
	return lib.NewCoordinates(x0, y0, x1, y1)
}
