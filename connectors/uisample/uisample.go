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
	"errors"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Make sure it implements the App and UIHandler interfaces.
var _ interface {
	meergo.App
	meergo.UIHandler
} = (*UISample)(nil)

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:              "UISample",
		SourceDescription: "test the UI components",
		Icon:              "",
	}, New)
}

// New returns a new UISample connector instance.
func New(conf *meergo.AppConfig) (*UISample, error) {
	c := UISample{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of UISample connector")
		}
	}
	return &c, nil
}

type UISample struct {
	conf     *meergo.AppConfig
	settings *Settings
}

// Schema returns the schema of the specified target.
func (uiSample *UISample) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	return types.Type{}, meergo.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (uiSample *UISample) ServeUI(ctx context.Context, event string, values json.Value, role meergo.Role) (*meergo.UI, error) {

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
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "MyInput", Label: "Input", Placeholder: "Insert Text", HelpText: "Help text of the input component", Rows: 1},
			&meergo.Input{Name: "MyTextarea", Label: "Textarea", Placeholder: "Insert Text", HelpText: "Help text of the textarea component", Rows: 5},
			&meergo.Select{Name: "MySelect", Label: "Select", Placeholder: "Choose an option", HelpText: "Help text of the select component", Options: []meergo.Option{{Text: "First select option", Value: "firstOption"}, {Text: "Second select option", Value: "secondOption"}, {Text: "Third select option", Value: "thirdOption"}}},
			&meergo.Checkbox{Name: "MyCheckbox", Label: "Checkbox"},
			&meergo.ColorPicker{Name: "MyColorPicker", Label: "ColorPicker"},
			&meergo.Radios{Name: "MyRadios", Label: "Radios", Options: []meergo.Option{{Text: "First radio option", Value: "firstOption"}, {Text: "Second radio option", Value: "secondOption"}, {Text: "Third radio option", Value: "thirdOption"}}},
			&meergo.Range{Name: "MyRange", Label: "Range", HelpText: "Help text of the range component", Min: 1, Max: 1000, Step: 10},
			&meergo.Switch{Name: "MySwitch", Label: "Switch"},
			&meergo.KeyValue{
				Name:       "MyKeyValue",
				Label:      "KeyValue",
				KeyLabel:   "Key label",
				ValueLabel: "Value label",
				KeyComponent: &meergo.Input{
					Name:        "MyKeyValueKey",
					Placeholder: "Insert Text",
					Rows:        1,
				},
				ValueComponent: &meergo.Input{
					Name:        "MyKeyValueValue",
					Placeholder: "Insert Text",
					Rows:        1,
				},
			},
			&meergo.Text{Text: "lorem ipsum dolor sit amet consecuctur", Label: "Text"},
			&meergo.AlternativeFieldSets{
				Label:    "AlternativeFieldSets",
				HelpText: "Help text of the alternativeFieldSets component",
				Sets: []meergo.FieldSet{
					{
						Name:  "FirstSet",
						Label: "First Set",
						Fields: []meergo.Component{
							&meergo.Input{Name: "MySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&meergo.Input{Name: "MyFirstSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&meergo.Input{Name: "MyFirstSetTextarea", Label: "Textarea", Placeholder: "Insert Text", Rows: 5},
						},
					},
					{
						Name:  "SecondSet",
						Label: "Second Set",
						Fields: []meergo.Component{
							&meergo.Input{Name: "MySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&meergo.Input{Name: "MySecondSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&meergo.Input{Name: "MySecondSetTextarea", Label: "Textarea", Placeholder: "Insert Text ", Rows: 5},
							&meergo.Checkbox{Name: "MySecondSetCheckbox", Label: "Set Checkbox"},
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
func (uiSample *UISample) saveValues(ctx context.Context, values json.Value) error {
	var s Settings
	err := values.Unmarshal(&s)
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
	MyKeyValue    []meergo.KV
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
