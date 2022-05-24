package lib

import (
	"sync"

	"github.com/jroimartin/gocui"
)

const submitButtonName = "submitButton"

type Input interface {
	Name() string
	Value() interface{}
	SetValue(*gocui.Gui, interface{}, string)
	Layout(*gocui.Gui, Coordinates) error
	Register(*gocui.Gui) error
	Deregister(*gocui.Gui) error
}

type Form struct {
	inputs       []Input
	submitButton *ButtonWidget
	name, title  string

	C chan map[string]interface{}

	isRegistered    bool
	lastCoordinates Coordinates
	mu              sync.RWMutex
}

func NewForm(inputs []Input, name, title, submitButtonValue string) *Form {
	of := Form{
		inputs:          inputs,
		name:            name,
		title:           title,
		C:               make(chan map[string]interface{}),
		isRegistered:    false,
		lastCoordinates: NewCoordinates(0, 0, 1, 1),
		mu:              sync.RWMutex{},
	}
	of.submitButton = NewButtonWidget(name+submitButtonName, 19, 2, submitButtonValue, of.submit)
	return &of
}

func (f *Form) submit(_ *gocui.Gui, _ *gocui.View) error {
	formDataMap := make(map[string]interface{})
	for _, input := range f.inputs {
		formDataMap[input.Name()] = input.Value()
	}
	f.C <- formDataMap
	return nil
}

func (f *Form) Layout(gui *gocui.Gui, coordinates Coordinates) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCoordinates = coordinates
	if !f.isRegistered {
		return nil
	}
	if err := f.setupView(gui, f.lastCoordinates); err != nil {
		return err
	}
	x0, y0, x1, y1 := f.lastCoordinates.Value()
	inputSizeStep := ((x1 - x0 - 2) / (len(f.inputs) + 1))
	for i, input := range f.inputs {
		inputCoordinates := NewCoordinates(x0+2+i*inputSizeStep, y0+1, x0+2+(i+1)*inputSizeStep-1, y1-1)
		if err := input.Layout(gui, inputCoordinates); err != nil {
			return err
		}
	}
	i := len(f.inputs)
	buttonCoordinates := NewCoordinates(x0+2+i*inputSizeStep, y0+1, x0+2+(i+1)*inputSizeStep-1, y1-1)
	if err := f.submitButton.Layout(gui, buttonCoordinates); err != nil {
		return err
	}
	return nil
}

func (f *Form) Register(gui *gocui.Gui) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isRegistered = true
	gui.Update(func(gui *gocui.Gui) error {
		return f.setupView(gui, f.lastCoordinates)
	})
	for _, input := range f.inputs {
		if err := input.Register(gui); err != nil {
			return err
		}
	}
	if err := f.submitButton.Register(gui); err != nil {
		return err
	}
	gui.Update(func(gui *gocui.Gui) error {
		return f.setupView(gui, f.lastCoordinates)
	})
	return nil
}

func (f *Form) Deregister(gui *gocui.Gui) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isRegistered = false
	for _, input := range f.inputs {
		if err := input.Deregister(gui); err != nil {
			return err
		}
	}
	if err := f.submitButton.Deregister(gui); err != nil {
		return err
	}
	if err := gui.DeleteView(f.name); err != nil {
		return err
	}
	DeleteKeybindings(gui, f.name)
	return nil
}

func (f *Form) SubmitButtonName() string {
	return f.name + submitButtonName
}

func (f *Form) setupView(gui *gocui.Gui, coordinates Coordinates) error {
	x0, y0, x1, y1 := coordinates.Value()
	v, err := gui.SetView(f.name, x0, y0, x1, y1)
	// already set up
	if err == nil {
		return nil
	}
	// unexpected error
	if err != gocui.ErrUnknownView {
		return err
	}
	// not yet set up
	v.Title = f.title
	return nil
}
