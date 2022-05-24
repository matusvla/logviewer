package lib

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

type InputBase struct {
	name            string
	title           string
	isRegistered    bool
	lastCoordinates Coordinates
	mu              sync.RWMutex
	SetupView       func(*gocui.Gui, Coordinates) error
	UpdateViewValue func(*gocui.Gui, *gocui.View, interface{}) error
}

func NewInputBase(name string, title string) InputBase {
	return InputBase{
		name:            name,
		title:           title,
		lastCoordinates: NewCoordinates(0, 0, 1, 1),
	}
}

func (ib *InputBase) Layout(gui *gocui.Gui, coordinates Coordinates) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	ib.lastCoordinates = coordinates
	if !ib.isRegistered {
		return nil
	}
	return ib.SetupView(gui, ib.lastCoordinates)
}

func (ib *InputBase) Register(gui *gocui.Gui) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	if ib.isRegistered {
		return nil
	}
	ib.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		ib.mu.Lock()
		defer ib.mu.Unlock()
		return ib.SetupView(gui, ib.lastCoordinates)
	})
	return nil
}

func (ib *InputBase) Deregister(gui *gocui.Gui) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	if !ib.isRegistered {
		return nil
	}
	ib.isRegistered = false
	if err := gui.DeleteView(ib.name); err != nil {
		return err
	}
	DeleteKeybindings(gui, ib.name)
	return nil
}

func (ib *InputBase) Name() string {
	return ib.name
}

//----------------------------------------------------------------------------------------------------------------------

const (
	maxUint = ^uint(0)
	maxInt  = int(maxUint >> 1)
)

type UIntInput struct {
	InputBase
	maxValue      int
	value         int
	isZeroAllowed bool
}

func NewUIntInput(name string, title string) *UIntInput {
	uii := &UIntInput{
		InputBase: InputBase{
			name:            name,
			title:           title,
			lastCoordinates: NewCoordinates(0, 0, 1, 1),
		},
		maxValue: maxInt,
	}
	uii.InputBase.SetupView = uii.setupView
	uii.InputBase.UpdateViewValue = func(gui *gocui.Gui, view *gocui.View, i interface{}) error {
		return uii.makeUpdateValueFn(i.(int))(gui, view)
	}
	return uii
}

func (ni *UIntInput) WithMaxValue(maxValue int) *UIntInput {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	ni.maxValue = maxValue
	return ni
}

func (ni *UIntInput) WithZeroAllowed() *UIntInput {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	ni.isZeroAllowed = true
	if ni.value == 0 {
		ni.value = -1
	}
	return ni
}

func (ni *UIntInput) Value() interface{} {
	ni.mu.RLock()
	defer ni.mu.RUnlock()
	return ni.value
}

func (ni *UIntInput) SetValue(gui *gocui.Gui, val interface{}, format string) {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	ni.value = val.(int)

	gui.Update(func(gui *gocui.Gui) error {
		v, err := gui.View(ni.name)
		if err != nil {
			return nil
		}
		v.Clear()
		if format != "" {
			_, _ = fmt.Fprintf(v, format, ni.value)
		} else {
			_, _ = fmt.Fprint(v, ni.value)
		}
		return nil
	})
}

func (ni *UIntInput) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(ni.name, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	ni.value = 0
	v.Title = ni.title
	for i := 0; i < 10; i++ {
		if err := SetKeybinding(gui, ni.name, rune(strconv.Itoa(i)[0]), gocui.ModNone, "", ni.makeUpdateValueFn(i)); err != nil {
			return err
		}
	}
	if err := SetKeybinding(gui, ni.name, gocui.KeyBackspace, gocui.ModNone, "", ni.eraseValue); err != nil {
		return err
	}
	return SetKeybinding(gui, ni.name, gocui.KeyBackspace2, gocui.ModNone, "", ni.eraseValue)
}

func (ni *UIntInput) makeUpdateValueFn(i int) func(*gocui.Gui, *gocui.View) error {
	return func(_ *gocui.Gui, v *gocui.View) error {
		ni.mu.Lock()
		defer ni.mu.Unlock()
		var newVal int
		if !ni.isZeroAllowed {
			newVal = ni.value*10 + i
			if newVal > ni.maxValue || newVal == 0 || newVal < ni.value {
				return nil
			}
		} else {
			if ni.value == -1 {
				ni.value = 0
			}
			newVal = ni.value*10 + i
			if newVal > ni.maxValue || newVal < ni.value {
				return nil
			}
		}
		ni.value = newVal
		v.Clear()
		if _, err := fmt.Fprint(v, ni.value); err != nil {
			return err
		}
		return nil
	}
}

func (ni *UIntInput) eraseValue(_ *gocui.Gui, v *gocui.View) error {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	newVal := ni.value / 10
	ni.value = newVal
	v.Clear()
	if newVal > 0 {
		_, _ = fmt.Fprint(v, newVal)
		return nil
	}
	if ni.isZeroAllowed {
		ni.value = -1
	}
	return nil
}

//----------------------------------------------------------------------------------------------------------------------

type Choice struct {
	InputBase
	allowedValues []string
	valueIndex    int
	loopChoices   bool
	onChangeFn    func(*gocui.Gui, string) error
}

func NewChoice(name string, title string, allowedValues []string, loopChoices bool) *Choice {
	uii := &Choice{
		InputBase: InputBase{
			name:            name,
			title:           title,
			lastCoordinates: NewCoordinates(0, 0, 1, 1),
		},
		loopChoices:   loopChoices,
		allowedValues: allowedValues,
	}
	uii.InputBase.SetupView = uii.setupView
	return uii
}

// todo this might be useful on other Intput types as well
func (c *Choice) WithOnChangeFn(fn func(*gocui.Gui, string) error) *Choice {
	c.onChangeFn = fn
	return c
}

func (c *Choice) SetAllowedValues(values []string) {
	c.allowedValues = values
}

func (c *Choice) Value() interface{} {
	if len(c.allowedValues) < c.valueIndex {
		c.valueIndex = 0
	}
	if len(c.allowedValues) == 0 {
		return ""
	}
	return c.allowedValues[c.valueIndex]
}

func (c *Choice) SetValue(_ *gocui.Gui, _ interface{}) {
	panic("not implemented")
}

func (c *Choice) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(c.name, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	if err != gocui.ErrUnknownView {
		return err
	}
	v.Title = c.title
	if len(c.allowedValues) < c.valueIndex {
		c.valueIndex = 0
	}
	if len(c.allowedValues) > 0 {
		_, _ = fmt.Fprint(v, c.allowedValues[c.valueIndex])
	}
	if err := SetKeybinding(gui, c.name, gocui.KeyArrowUp, gocui.ModNone, "next choice", c.makeUpdateValueFn(-1)); err != nil {
		return err
	}
	return SetKeybinding(gui, c.name, gocui.KeyArrowDown, gocui.ModNone, "previous choice", c.makeUpdateValueFn(1))
}

func (c *Choice) makeUpdateValueFn(scrollBy int) func(g *gocui.Gui, v *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		newValueIndex := c.valueIndex + scrollBy
		if !c.loopChoices {
			if newValueIndex < 0 || newValueIndex > len(c.allowedValues)-1 {
				return nil
			}
		} else {
			if newValueIndex < 0 {
				newValueIndex = len(c.allowedValues) + scrollBy
			} else {
				newValueIndex %= len(c.allowedValues)
			}
		}
		c.valueIndex = newValueIndex
		v.Clear()
		newValue := c.allowedValues[c.valueIndex]
		_, _ = fmt.Fprint(v, newValue)
		if c.onChangeFn != nil {
			if err := c.onChangeFn(gui, newValue); err != nil {
				return err
			}
		}
		return nil
	}
}

//----------------------------------------------------------------------------------------------------------------------

type MultilevelChoice struct {
	InputBase
	allowedValues MultilevelChoiceItems
	valueIndices  []int
	loopChoices   bool
	indentSpaces  int
	onChangeFn    func(*gocui.Gui, MultilevelChoiceItem) error
}

type MultilevelChoiceItem interface {
	Value() string
	Subitems() MultilevelChoiceItems
}

type MultilevelChoiceItems []MultilevelChoiceItem

func (m MultilevelChoiceItems) fprintUnwrapIndex(writer io.Writer, indices []int, offsetSpaces, indentSpaces int) (unwrappedLevels int) {
	var result int
	if len(m) == 0 {
		return 0
	}
	for i, item := range m {
		bulletPoint := "-"
		if len(item.Subitems()) > 0 {
			bulletPoint = "+"
		}
		if len(indices) > 0 && indices[0] == i {
			_, _ = fmt.Fprintf(writer, "%s%s %s\n", strings.Repeat(" ", offsetSpaces), bulletPoint, GreenBGString(item.Value()))
			if len(indices) > 1 {
				result += item.Subitems().fprintUnwrapIndex(writer, indices[1:], offsetSpaces+indentSpaces, indentSpaces)
			}
		} else {
			_, _ = fmt.Fprintf(writer, "%s%s %s\n", strings.Repeat(" ", offsetSpaces), bulletPoint, item.Value())
		}
	}
	return result + 1
}

func NewMultilevelChoice(name string, title string, allowedValues MultilevelChoiceItems, loopChoices bool, indentSpaces int) *MultilevelChoice {
	uii := &MultilevelChoice{
		InputBase: InputBase{
			name:            name,
			title:           title,
			lastCoordinates: NewCoordinates(0, 0, 1, 1),
		},
		allowedValues: allowedValues,
		loopChoices:   loopChoices,
		indentSpaces:  indentSpaces,
	}
	uii.InputBase.SetupView = uii.setupView
	return uii
}

func (c *MultilevelChoice) Value() (interface{}, []int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.valueIndices) < 1 {
		return MultilevelChoiceItem(nil), nil
	}
	result := c.allowedValues[c.valueIndices[0]]
	for _, index := range c.valueIndices[1:] {
		if !(len(result.Subitems()) < index) {
			return result, c.valueIndices
		}
		result = result.Subitems()[index]
	}

	return result, c.valueIndices
}

func (c *MultilevelChoice) SetValue(gui *gocui.Gui, val interface{}) {
	var isEmpty bool
	if val == nil {
		isEmpty = true
	}
	c.mu.Lock()
	c.allowedValues = val.(MultilevelChoiceItems)
	if c.allowedValues == nil {
		isEmpty = true
	}
	c.mu.Unlock()

	gui.Update(func(gui *gocui.Gui) error {
		v, err := gui.View(c.name)
		if err != nil {
			if err == gocui.ErrUnknownView {
				return nil
			}
			return err
		}
		v.Clear()
		if isEmpty {
			_, err := fmt.Fprint(v, "No available values")
			return err

		}
		c.allowedValues.fprintUnwrapIndex(v, c.valueIndices, 0, c.indentSpaces)
		return nil
	})
}

func (c *MultilevelChoice) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(c.name, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	if err != gocui.ErrUnknownView {
		return err
	}
	v.Title = c.title
	if err := c.makeVerticalScrollFn(0)(gui, v); err != nil {
		return err
	}
	if err := SetKeybinding(gui, c.name, gocui.KeyArrowUp, gocui.ModNone, "move up", c.makeVerticalScrollFn(-1)); err != nil {
		return err
	}
	if err := SetKeybinding(gui, c.name, gocui.KeyArrowDown, gocui.ModNone, "move down", c.makeVerticalScrollFn(1)); err != nil {
		return err
	}
	if err := SetKeybinding(gui, c.name, gocui.KeyArrowRight, gocui.ModNone, "unwrap", func(gui *gocui.Gui, v *gocui.View) error {
		if c.allowedValues == nil {
			return nil
		}
		if c.valueIndices == nil {
			c.valueIndices = append(c.valueIndices, 0)
			return nil
		}
		c.valueIndices = append(c.valueIndices, 0)
		v.Clear()
		unwrappedLevels := c.allowedValues.fprintUnwrapIndex(v, c.valueIndices, 0, c.indentSpaces)
		if unwrappedLevels < len(c.valueIndices) {
			c.valueIndices = c.valueIndices[:len(c.valueIndices)-1]
		}
		return nil
	}); err != nil {
		return err
	}
	if err := SetKeybinding(gui, c.name, gocui.KeyArrowLeft, gocui.ModNone, "wrap", func(gui *gocui.Gui, v *gocui.View) error {
		if c.allowedValues == nil {
			return nil
		}
		if len(c.valueIndices) < 2 {
			return nil
		}
		c.valueIndices = c.valueIndices[:len(c.valueIndices)-1]
		v.Clear()
		c.allowedValues.fprintUnwrapIndex(v, c.valueIndices, 0, c.indentSpaces)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (c *MultilevelChoice) makeVerticalScrollFn(scrollBy int) func(g *gocui.Gui, v *gocui.View) error {
	return func(gui *gocui.Gui, v *gocui.View) error {
		if c.allowedValues == nil {
			return nil
		}
		if c.valueIndices == nil {
			c.valueIndices = append(c.valueIndices, 0)
		}
		newValueIndex := c.valueIndices[len(c.valueIndices)-1] + scrollBy

		layer := c.allowedValues
		for _, index := range c.valueIndices[:len(c.valueIndices)-1] {
			layer = layer[index].Subitems()
		}
		length := len(layer)
		if length == 0 {
			return nil
		}
		if !c.loopChoices {
			if newValueIndex < 0 || newValueIndex > length-1 {
				return nil
			}
		} else {
			if newValueIndex < 0 {
				newValueIndex = length + scrollBy
			} else {
				newValueIndex %= length
			}
		}
		c.valueIndices[len(c.valueIndices)-1] = newValueIndex
		v.Clear()
		c.allowedValues.fprintUnwrapIndex(v, c.valueIndices, 0, c.indentSpaces)
		return nil
	}
}
