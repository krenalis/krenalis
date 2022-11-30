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

// Role represents the role of a connection.
type Role int

const (
	BothRole        Role = iota // both
	SourceRole                  // source
	DestinationRole             // destination
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
	Role        Role
}

func (i *Input) component() {}

type Select struct {
	Name        string
	Value       any
	Label       string
	Placeholder string
	Options     []Option
	Role        Role
}

func (s *Select) component() {}

type Checkbox struct {
	Name  string
	Value bool
	Label string
	Role  Role
}

func (ck *Checkbox) component() {}

type ColorPicker struct {
	Name  string
	Value string
	Label string
	Role  Role
}

func (cp *ColorPicker) component() {}

type Radios struct {
	Name    string
	Value   any
	Label   string
	Options []Option
	Role    Role
}

func (rd *Radios) component() {}

type Range struct {
	Name  string
	Value int
	Label string
	Min   int
	Max   int
	Step  int
	Role  Role
}

func (r *Range) component() {}

type Switch struct {
	Name  string
	Value bool
	Label string
	Role  Role
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
	Role           Role
}

func (kv *KeyValue) component() {}

type Text struct {
	Name  string
	Value string
	Label string
	Role  Role
}

func (txt *Text) component() {}

type Action struct {
	Event   string
	Text    string
	Variant string // primary|neutral|danger|warning|success
	Role    Role
}

// Alert represents an alert message to be shown in the UI.
type Alert struct {

	// Message is the message of the alert.
	Message string

	// Variant is the variant of the alert message.
	Variant AlertVariant
}

// PrimaryAlert returns a primary alert.
func PrimaryAlert(msg string) *Alert { return &Alert{Message: msg, Variant: Primary} }

// SuccessAlert returns a success alert.
func SuccessAlert(msg string) *Alert { return &Alert{Message: msg, Variant: Success} }

// NeutralAlert returns a neutral alert.
func NeutralAlert(msg string) *Alert { return &Alert{Message: msg, Variant: Neutral} }

// WarningAlert returns a warning alert.
func WarningAlert(msg string) *Alert { return &Alert{Message: msg, Variant: Warning} }

// DangerAlert returns a danger alert.
func DangerAlert(msg string) *Alert { return &Alert{Message: msg, Variant: Danger} }

// AlertVariant represents the alert variants. The variants are taken from
// Shoelace (see https://shoelace.style/components/alert).
type AlertVariant int

const (
	Primary AlertVariant = iota
	Success
	Neutral
	Warning
	Danger
)

func (v AlertVariant) String() string {
	switch v {
	case Primary:
		return "primary"
	case Success:
		return "success"
	case Neutral:
		return "neutral"
	case Warning:
		return "warning"
	case Danger:
		return "danger"
	default:
		panic(fmt.Sprintf("invalid alert variant %d", v))
	}
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
