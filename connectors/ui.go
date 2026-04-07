// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"errors"
	"fmt"

	"github.com/krenalis/krenalis/tools/json"
)

// ErrUIEventNotExist values are returned by the ServeUI method when the event
// does not exist.
var ErrUIEventNotExist = errors.New("event does not exist")

// InvalidSettingsError is returned by the ServeUI method when the settings are
// not valid.
type InvalidSettingsError struct {
	Msg string
}

func (err *InvalidSettingsError) Error() string {
	return err.Msg
}

func NewInvalidSettingsError(msg string) error {
	return &InvalidSettingsError{msg}
}

func NewInvalidSettingsErrorf(msg string, a ...any) error {
	return &InvalidSettingsError{fmt.Sprintf(msg, a...)}
}

// UI represents the user interface of a connector that is shown to users.
type UI struct {
	Alert    *Alert      // Alert, if not empty, appears as a notification.
	Fields   []Component // Fields, if not empty, are the form inputs for settings.
	Settings json.Value  // Settings hold the settings of the fields.
	Buttons  []Button    // Buttons are the button elements that can trigger actions.
}

type Component interface {
	component()
}

type Option struct {
	Text  string
	Value any
}

type Input struct {
	Name            string
	Type            string // date|datetime-local|email|number|password|search|tel|text|time|url - default is 'text'
	Label           string
	Placeholder     string
	HelpText        string
	Rows            int // if bigger than 1, the corresponding component is a textarea.
	OnlyIntegerPart bool
	MinLength       int
	MaxLength       int
	Error           string
	Role            Role
}

func (i *Input) component() {}

type Select struct {
	Name        string
	Label       string
	Placeholder string
	HelpText    string
	Options     []Option
	Error       string
	Role        Role
}

func (s *Select) component() {}

type Checkbox struct {
	Name  string
	Label string
	Error string
	Role  Role
}

func (ck *Checkbox) component() {}

type ColorPicker struct {
	Name  string
	Label string
	Error string
	Role  Role
}

func (cp *ColorPicker) component() {}

type Radios struct {
	Name    string
	Label   string
	Options []Option
	Error   string
	Role    Role
}

func (rd *Radios) component() {}

type Range struct {
	Name     string
	Label    string
	HelpText string
	Min      int
	Max      int
	Step     int
	Error    string
	Role     Role
}

func (r *Range) component() {}

type Switch struct {
	Name  string
	Label string
	Error string
	Role  Role
}

func (s *Switch) component() {}

type KeyValue struct {
	Name           string
	Label          string
	KeyLabel       string
	KeyComponent   Component
	ValueLabel     string
	ValueComponent Component
	Error          string
	Role           Role
}

func (kv *KeyValue) component() {}

type FieldSet struct {
	Name   string
	Label  string
	Fields []Component
	Role   Role
}

// KV represents a key-value pair.
// A KeyValue component stores its data as a slice of KV, i.e., []KV.
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AlternativeFieldSets struct {
	Label    string
	HelpText string
	Sets     []FieldSet
	Role     Role
}

func (afs *AlternativeFieldSets) component() {}

type Text struct {
	Label string
	Text  string
	Role  Role
}

func (txt *Text) component() {}

type Button struct {
	Event   string
	Text    string
	Variant string // primary|neutral|danger|warning|success
	Role    Role
}

// SaveButton is a button common to connectors whose event indicates to save the
// settings (both when creating a new connection and when updating the settings
// of an existing connection).
var SaveButton = Button{
	Event: "save",
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
