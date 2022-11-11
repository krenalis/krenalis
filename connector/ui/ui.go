//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package ui

import (
	"errors"
	"fmt"
)

// ErrEventNotExist values are returned by the ServeUI methods when the event
// does not exist.
var ErrEventNotExist = errors.New("event does not exist")

// Direction represents the direction of a connection.
type Direction int

const (
	BothDir   Direction = iota // both
	SourceDir                  // source
	DestDir                    // destination
)

type Form struct {
	Fields  []Component
	Actions []Action
}

type Component interface {
	component()
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
	Direction   Direction
}

func (i *Input) component() {}

type Select struct {
	Name        string
	Value       any
	Label       string
	Placeholder string
	Options     []Option
	Direction   Direction
}

func (s *Select) component() {}

type Checkbox struct {
	Name      string
	Value     bool
	Label     string
	Direction Direction
}

func (ck *Checkbox) component() {}

type ColorPicker struct {
	Name      string
	Value     string
	Label     string
	Direction Direction
}

func (cp *ColorPicker) component() {}

type Radios struct {
	Name      string
	Value     any
	Label     string
	Options   []Option
	Direction Direction
}

func (rd *Radios) component() {}

type Range struct {
	Name      string
	Value     int
	Label     string
	Min       int
	Max       int
	Step      int
	Direction Direction
}

func (r *Range) component() {}

type Switch struct {
	Name      string
	Value     bool
	Label     string
	Direction Direction
}

func (s *Switch) component() {}

type KeyValue struct {
	Name           string
	Value          map[string]any
	Label          string
	KeyLabel       string
	KeyComponent   Component
	ValueLabel     string
	ValueComponent Component
	Direction      Direction
}

func (kv *KeyValue) component() {}

type Text struct {
	Name      string
	Value     string
	Label     string
	Direction Direction
}

func (txt *Text) component() {}

type Action struct {
	Event     string
	Text      string
	Variant   string // primary|neutral|danger|warning|success
	Direction Direction
}

// Error represents an error to be displayed in the UI.
type Error struct {
	err error
}

func (err Error) Error() string {
	return err.err.Error()
}

// Errorf formats according to a format specifier and returns an Error value.
func Errorf(format string, a ...any) Error {
	return Error{err: fmt.Errorf(format, a...)}
}
