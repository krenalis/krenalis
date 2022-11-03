package connector

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type SettingsUI struct {
	Components []Component
	Actions    []Action
}

type Component interface {
	component()
	json.Marshaler
}

func marshalComponent(c Component, componentType string) ([]byte, error) {
	obj := map[string]any{}
	rv := reflect.ValueOf(c).Elem()
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		obj[typ.Field(i).Name] = rv.Field(i).Interface()
	}
	if _, ok := obj["ComponentType"]; ok {
		panic("BUG: field Type already defined")
	}
	obj["ComponentType"] = componentType
	return json.Marshal(obj)
}

type Option struct {
	Text  string
	Value any
}

type Input struct {
	Name        string
	Value       any
	Type        string // date|datetime-local|email|number|password|search|tel|text|time|url - default is 'text'
	Label       string
	Placeholder string
	Rows        int // if bigger than 1, the corresponding component is a textarea.
	MinLength   int
	MaxLength   int
}

func (i *Input) component() {}

func (i *Input) MarshalJSON() ([]byte, error) { return marshalComponent(i, "Input") }

type Select struct {
	Name        string
	Value       any
	Label       string
	Placeholder string
	Options     []Option
}

func (s *Select) component() {}

func (s *Select) MarshalJSON() ([]byte, error) { return marshalComponent(s, "Select") }

type Checkbox struct {
	Name  string
	Value bool
	Label string
}

func (ck *Checkbox) component() {}

func (ck *Checkbox) MarshalJSON() ([]byte, error) { return marshalComponent(ck, "Checkbox") }

type ColorPicker struct {
	Name  string
	Value string
	Label string
}

func (cp *ColorPicker) component() {}

func (cp *ColorPicker) MarshalJSON() ([]byte, error) { return marshalComponent(cp, "ColorPicker") }

type Radios struct {
	Name    string
	Value   any
	Label   string
	Options []Option
}

func (rd *Radios) component() {}

func (rd *Radios) MarshalJSON() ([]byte, error) { return marshalComponent(rd, "Radios") }

type Range struct {
	Name  string
	Value int
	Label string
	Min   int
	Max   int
	Step  int
}

func (r *Range) component() {}

func (r *Range) MarshalJSON() ([]byte, error) { return marshalComponent(r, "Range") }

type Switch struct {
	Name  string
	Value bool
	Label string
}

func (s *Switch) component() {}

func (s *Switch) MarshalJSON() ([]byte, error) { return marshalComponent(s, "Switch") }

type KeyValue struct {
	Name           string
	Value          map[string]any
	Label          string
	KeyLabel       string
	KeyComponent   Component
	ValueLabel     string
	ValueComponent Component
}

func (kv *KeyValue) component() {}

func (kv *KeyValue) MarshalJSON() ([]byte, error) { return marshalComponent(kv, "KeyValue") }

type Action struct {
	Event   string
	Text    string
	Variant string // primary|neutral|danger|warning|success
}

// UIError represents an error to be displayed in the UI.
type UIError struct {
	err error
}

func (err UIError) Error() string {
	return err.err.Error()
}

// UIErrorf formats according to a format specifier and returns a UIError
// value.
func UIErrorf(format string, a ...any) UIError {
	return UIError{err: fmt.Errorf(format, a)}
}
