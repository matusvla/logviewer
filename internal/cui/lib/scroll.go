package lib

import (
	"github.com/jroimartin/gocui"
)

func ScrollView(v *gocui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		_, sy := v.Size()
		if newOY := oy + dy; newOY >= 0 && newOY < len(v.BufferLines())-sy+1 {
			if err := v.SetOrigin(ox, newOY); err != nil {
				return err
			}
		}
	}
	return nil
}
