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

// Make sure it implements the App and UIHandler interfaces.
var _ interface {
	chichi.App
	chichi.UIHandler
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
	settings *Settings
}

// Schema returns the schema of the specified target.
func (uiSample *UISample) Schema(ctx context.Context, target chichi.Targets, role chichi.Role, eventType string) (types.Type, error) {
	return types.Type{}, chichi.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (uiSample *UISample) ServeUI(ctx context.Context, event string, values []byte, role chichi.Role) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if uiSample.settings != nil {
			s = *uiSample.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, uiSample.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "MyInput", Label: "Input", Placeholder: "Insert Text", HelpText: "Help text of the input component", Rows: 1},
			&chichi.Input{Name: "MyTextarea", Label: "Textarea", Placeholder: "Insert Text", HelpText: "Help text of the textarea component", Rows: 5},
			&chichi.Select{Name: "MySelect", Label: "Select", Placeholder: "Choose an option", HelpText: "Help text of the select component", Options: []chichi.Option{{Text: "First select option", Value: "firstOption"}, {Text: "Second select option", Value: "secondOption"}, {Text: "Third select option", Value: "thirdOption"}}},
			&chichi.Checkbox{Name: "MyCheckbox", Label: "Checkbox"},
			&chichi.ColorPicker{Name: "MyColorPicker", Label: "ColorPicker"},
			&chichi.Radios{Name: "MyRadios", Label: "Radios", Options: []chichi.Option{{Text: "First radio option", Value: "firstOption"}, {Text: "Second radio option", Value: "secondOption"}, {Text: "Third radio option", Value: "thirdOption"}}},
			&chichi.Range{Name: "MyRange", Label: "Range", HelpText: "Help text of the range component", Min: 1, Max: 1000, Step: 10},
			&chichi.Switch{Name: "MySwitch", Label: "Switch"},
			&chichi.KeyValue{
				Name:       "MyKeyValue",
				Label:      "KeyValue",
				KeyLabel:   "Key label",
				ValueLabel: "Value label",
				KeyComponent: &chichi.Input{
					Name:        "MyKeyValueKey",
					Placeholder: "Insert Text",
					Rows:        1,
				},
				ValueComponent: &chichi.Input{
					Name:        "MyKeyValueValue",
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
						Name:  "FirstSet",
						Label: "First Set",
						Fields: []chichi.Component{
							&chichi.Input{Name: "MySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "MyFirstSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "MyFirstSetTextarea", Label: "Textarea", Placeholder: "Insert Text", Rows: 5},
						},
					},
					{
						Name:  "SecondSet",
						Label: "Second Set",
						Fields: []chichi.Component{
							&chichi.Input{Name: "MySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "MySecondSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "MySecondSetTextarea", Label: "Textarea", Placeholder: "Insert Text ", Rows: 5},
							&chichi.Checkbox{Name: "MySecondSetCheckbox", Label: "Set Checkbox"},
						},
					},
				},
			},
		},
		Values: values,
	}

	return ui, nil
}

// saveValues saves the user-entered values as settings.
func (uiSample *UISample) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = uiSample.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	uiSample.settings = &s
	return nil
}

type Settings struct {
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
