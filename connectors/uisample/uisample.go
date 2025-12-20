// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package uisample provides a sample connector for UI integration.
package uisample

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "ui-sample",
		Label:      "UISample",
		Categories: connectors.CategoryTesting,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Test the UI components",
				Overview: overview,
			},
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Test the UI components",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for UI sample.
func New(env *connectors.ApplicationEnv) (*UISample, error) {
	c := UISample{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for UI Sample")
		}
	}
	return &c, nil
}

type UISample struct {
	env      *connectors.ApplicationEnv
	settings *innerSettings
}

// RecordSchema returns the schema of the specified target and role.
func (uiSample *UISample) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	return types.Type{}, connectors.ErrEventTypeNotExist
}

// Records returns the records of the specified target.
func (uiSample *UISample) Records(ctx context.Context, target connectors.Targets, lastChangeTime time.Time, ids []string, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	return nil, "", io.EOF
}

// ServeUI serves the connector's user interface.
func (uiSample *UISample) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if uiSample.settings != nil {
			s = *uiSample.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, uiSample.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "myInput", Label: "Input", Placeholder: "Insert Text", HelpText: "Help text of the input component", Rows: 1},
			&connectors.Input{Name: "myTextarea", Label: "Textarea", Placeholder: "Insert Text", HelpText: "Help text of the textarea component", Rows: 5},
			&connectors.Select{Name: "mySelect", Label: "Select", Placeholder: "Choose an option", HelpText: "Help text of the select component", Options: []connectors.Option{{Text: "First select option", Value: "firstOption"}, {Text: "Second select option", Value: "secondOption"}, {Text: "Third select option", Value: "thirdOption"}}},
			&connectors.Checkbox{Name: "myCheckbox", Label: "Checkbox"},
			&connectors.ColorPicker{Name: "myColorPicker", Label: "ColorPicker"},
			&connectors.Radios{Name: "myRadios", Label: "Radios", Options: []connectors.Option{{Text: "First radio option", Value: "firstOption"}, {Text: "Second radio option", Value: "secondOption"}, {Text: "Third radio option", Value: "thirdOption"}}},
			&connectors.Range{Name: "myRange", Label: "Range", HelpText: "Help text of the range component", Min: 1, Max: 1000, Step: 10},
			&connectors.Switch{Name: "mySwitch", Label: "Switch"},
			&connectors.KeyValue{
				Name:       "myKeyValue",
				Label:      "KeyValue",
				KeyLabel:   "Key label",
				ValueLabel: "Value label",
				KeyComponent: &connectors.Input{
					Name:        "myKeyValueKey",
					Placeholder: "Insert Text",
					Rows:        1,
				},
				ValueComponent: &connectors.Input{
					Name:        "myKeyValueValue",
					Placeholder: "Insert Text",
					Rows:        1,
				},
			},
			&connectors.Text{Text: "lorem ipsum dolor sit amet consecuctur", Label: "Text"},
			&connectors.AlternativeFieldSets{
				Label:    "AlternativeFieldSets",
				HelpText: "Help text of the alternativeFieldSets component",
				Sets: []connectors.FieldSet{
					{
						Name:  "firstSet",
						Label: "First Set",
						Fields: []connectors.Component{
							&connectors.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&connectors.Input{Name: "myFirstSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&connectors.Input{Name: "myFirstSetTextarea", Label: "Textarea", Placeholder: "Insert Text", Rows: 5},
						},
					},
					{
						Name:  "secondSet",
						Label: "Second Set",
						Fields: []connectors.Component{
							&connectors.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&connectors.Input{Name: "mySecondSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&connectors.Input{Name: "mySecondSetTextarea", Label: "Textarea", Placeholder: "Insert Text ", Rows: 5},
							&connectors.Checkbox{Name: "mySecondSetCheckbox", Label: "Set Checkbox"},
						},
					},
				},
			},
		},
		Settings: settings,
	}

	return ui, nil
}

// Upsert updates or creates records in the API for the specified target.
func (uiSample *UISample) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {
	return nil
}

// saveSettings saves the settings.
func (uiSample *UISample) saveSettings(ctx context.Context, options json.Value) error {
	var s innerSettings
	err := options.Unmarshal(&s)
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = uiSample.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	uiSample.settings = &s
	return nil
}

type innerSettings struct {
	MyInput       string          `json:"myInput"`
	MyTextarea    string          `json:"myTextarea"`
	MySelect      string          `json:"mySelect"`
	MyCheckbox    bool            `json:"myCheckbox"`
	MyColorPicker string          `json:"myColoPicker"`
	MyRadios      string          `json:"myRadios"`
	MyRange       int             `json:"myRange"`
	MySwitch      bool            `json:"mySwitch"`
	MyKeyValue    []connectors.KV `json:"myKeyValue"`
	FirstSet      *struct {
		MySharedInput      string `json:"mySharedInput"`
		MyFirstSetInput    string `json:"myFirstSetInput"`
		MyFirstSetTextarea string `json:"myFirstSetTextarea"`
	} `json:"firstSet"`
	SecondSet *struct {
		MySharedInput       string `json:"mySharedInput"`
		MySecondSetInput    string `json:"mySecondSetInput"`
		MySecondSetTextarea string `json:"mySecondSetTextarea"`
		MySecondSetCheckbox bool   `json:"mySecondSetCheckbox"`
	} `json:"secondSet"`
}
