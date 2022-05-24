package lib

import "fmt"

type Coordinates struct {
	x0, y0, x1, y1 int
}

func (c Coordinates) String() string {
	return fmt.Sprintf("[%d-%d,%d-%d]", c.x0, c.x1, c.y0, c.y1)
}

func NewCoordinates(x0, y0, x1, y1 int) Coordinates {
	return Coordinates{
		x0: x0,
		y0: y0,
		x1: x1,
		y1: y1,
	}
}

func (c Coordinates) Value() (x0, y0, x1, y1 int) {
	return c.x0, c.y0, c.x1, c.y1
}
