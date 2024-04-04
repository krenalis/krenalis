//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package uisample implements the UISample connector.
package uisample

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Make sure it implements the App and UI interfaces.
var _ interface {
	chichi.App
	chichi.UI
} = (*UISample)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:              "UISample",
		SourceDescription: "test the UI components",
		Icon:              "",
	}, New)
}

// New returns a new UISample connector instance.
func New(conf *chichi.AppConfig) (*UISample, error) {
	c := UISample{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of UISample connector")
		}
	}
	return &c, nil
}

type UISample struct {
	conf     *chichi.AppConfig
	settings *settings
}

// Resource returns the resource from a client token.
func (uiSample *UISample) Resource(ctx context.Context) (string, error) {
	return "", nil
}

// Schema returns the schema of the specified target.
func (uiSample *UISample) Schema(ctx context.Context, target chichi.Targets, eventType string) (types.Type, error) {
	return types.Type{}, chichi.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (uiSample *UISample) ServeUI(ctx context.Context, event string, values []byte) (*chichi.Form, *chichi.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if uiSample.settings != nil {
			s = *uiSample.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		s, err := uiSample.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = uiSample.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, chichi.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, chichi.ErrEventNotExist
	}

	form := &chichi.Form{
		Fields: []chichi.Component{
			&chichi.Input{Name: "myInput", Label: "Input", Placeholder: "Insert Text", HelpText: "Help text of the input component", Rows: 1},
			&chichi.Input{Name: "myTextarea", Label: "Textarea", Placeholder: "Insert Text", HelpText: "Help text of the textarea component", Rows: 5},
			&chichi.Select{Name: "mySelect", Label: "Select", Placeholder: "Choose an option", HelpText: "Help text of the select component", Options: []chichi.Option{{Text: "First select option", Value: "firstOption"}, {Text: "Second select option", Value: "secondOption"}, {Text: "Third select option", Value: "thirdOption"}}},
			&chichi.Checkbox{Name: "myCheckbox", Label: "Checkbox"},
			&chichi.ColorPicker{Name: "myColorPicker", Label: "ColorPicker"},
			&chichi.Radios{Name: "myRadios", Label: "Radios", Options: []chichi.Option{{Text: "First radio option", Value: "firstOption"}, {Text: "Second radio option", Value: "secondOption"}, {Text: "Third radio option", Value: "thirdOption"}}},
			&chichi.Range{Name: "myRange", Label: "Range", HelpText: "Help text of the range component", Min: 1, Max: 1000, Step: 10},
			&chichi.Switch{Name: "mySwitch", Label: "Switch"},
			&chichi.KeyValue{
				Name:       "myKeyValue",
				Label:      "KeyValue",
				KeyLabel:   "Key label",
				ValueLabel: "Value label",
				KeyComponent: &chichi.Input{
					Name:        "myKeyValueKey",
					Placeholder: "Insert Text",
					Rows:        1,
				},
				ValueComponent: &chichi.Input{
					Name:        "myKeyValueValue",
					Placeholder: "Insert Text",
					Rows:        1,
				},
			},
			&chichi.Text{Text: "lorem ipsum dolor sit amet consecuctur", Label: "Text"},
			&chichi.AlternativeFieldSets{
				Label:    "AlternativeFieldSets",
				HelpText: "Help text of the alternativeFieldSets component",
				Sets: []chichi.FieldSet{
					{
						Name:  "firstSet",
						Label: "First Set",
						Fields: []chichi.Component{
							&chichi.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "myFirstSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "myFirstSetTextarea", Label: "Textarea", Placeholder: "Insert Text", Rows: 5},
						},
					},
					{
						Name:  "secondSet",
						Label: "Second Set",
						Fields: []chichi.Component{
							&chichi.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "mySecondSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "mySecondSetTextarea", Label: "Textarea", Placeholder: "Insert Text ", Rows: 5},
							&chichi.Checkbox{Name: "mySecondSetCheckbox", Label: "Set Checkbox"},
						},
					},
				},
			},
		},
		Values: values,
		Actions: []chichi.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (uiSample *UISample) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

type settings struct {
	MyInput       string
	MyTextarea    string
	MySelect      string
	MyCheckbox    bool
	MyColorPicker string
	MyRadios      string
	MyRange       int
	MySwitch      bool
	MyKeyValue    map[string]string
	FirstSet      *struct {
		MySharedInput      string
		MyFirstSetInput    string
		MyFirstSetTextarea string
	}
	SecondSet *struct {
		MySharedInput       string
		MySecondSetInput    string
		MySecondSetTextarea string
		MySecondSetCheckbox bool
	}
}
