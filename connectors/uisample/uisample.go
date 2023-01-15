//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package uisample

import (
	"context"
	"encoding/json"
	"errors"

	"chichi/connector"
	"chichi/connector/ui"
)

// Make sure it implements the StreamConnection interface.
var _ connector.StreamConnection = &connection{}

func init() {
	connector.RegisterStream(connector.Stream{
		Name:    "UISample",
		Icon:    "",
		Connect: connect,
	})
}

// connect returns a new UISample connection.
func connect(ctx context.Context, conf *connector.StreamConfig) (connector.StreamConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of UISample connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

// Close closes the stream. Must be called if at least one Send or Receive call
// has been made. It cannot be called concurrently with Send and Receive.
func (c *connection) Close() error {
	return nil
}

// Receive receives an event from the stream. Callers call the ack function to
// notify that the event has been received. The connector resends the event if
// not acknowledged.
//
// Caller do not modify the event data, even temporarily, and event is not
// retained after the ack function has been called.
func (c *connection) Receive() ([]byte, func(), error) {
	return nil, nil, nil
}

// Send sends an event to the stream. If ack is not nil, connector calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but event is not retained after the ack
// function has been called.
func (c *connection) Send(event []byte, options connector.SendOptions, ack func(err error)) error {
	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	var s settings

	switch event {
	case "load":
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		err := c.firehose.SetSettings(values)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "myInput", Label: "Input", Placeholder: "Insert Text", HelpText: "Help text of the input component", Rows: 1},
			&ui.Input{Name: "myTextarea", Label: "Textarea", Placeholder: "Insert Text", HelpText: "Help text of the textarea component", Rows: 5},
			&ui.Select{Name: "mySelect", Label: "Select", Placeholder: "Choose an option", HelpText: "Help text of the select component", Options: []ui.Option{{Text: "First select option", Value: "firstOption"}, {Text: "Second select option", Value: "secondOption"}, {Text: "Third select option", Value: "thirdOption"}}},
			&ui.Checkbox{Name: "myCheckbox", Label: "Checkbox"},
			&ui.ColorPicker{Name: "myColorPicker", Label: "ColorPicker"},
			&ui.Radios{Name: "myRadios", Label: "Radios", Options: []ui.Option{{Text: "First radio option", Value: "firstOption"}, {Text: "Second radio option", Value: "secondOption"}, {Text: "Third radio option", Value: "thirdOption"}}},
			&ui.Range{Name: "myRange", Label: "Range", HelpText: "Help text of the range component", Min: 1, Max: 1000, Step: 10},
			&ui.Switch{Name: "mySwitch", Label: "Switch"},
			&ui.KeyValue{
				Name:       "myKeyValue",
				Label:      "KeyValue",
				KeyLabel:   "Key label",
				ValueLabel: "Value label",
				KeyComponent: &ui.Input{
					Name:        "myKeyValueKey",
					Placeholder: "Insert Text",
					Rows:        1,
				},
				ValueComponent: &ui.Input{
					Name:        "myKeyValueValue",
					Placeholder: "Insert Text",
					Rows:        1,
				},
			},
			&ui.Text{Text: "lorem ipsum dolor sit amet consecuctur", Label: "Text"},
			&ui.AlternativeFieldSets{
				Label:    "AlternativeFieldSets",
				HelpText: "Help text of the alternativeFieldSets component",
				Sets: []ui.FieldSet{
					{
						Name:  "firstSet",
						Label: "First Set",
						Fields: []ui.Component{
							&ui.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&ui.Input{Name: "myFirstSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&ui.Input{Name: "myFirstSetTextarea", Label: "Textarea", Placeholder: "Insert Text", Rows: 5},
						},
					},
					{
						Name:  "secondSet",
						Label: "Second Set",
						Fields: []ui.Component{
							&ui.Input{Name: "mySharedInput", Label: "Shared input", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&ui.Input{Name: "mySecondSetInput", Label: "Input", Placeholder: "Insert Text", Type: "text", MinLength: 1, MaxLength: 253},
							&ui.Input{Name: "mySecondSetTextarea", Label: "Textarea", Placeholder: "Insert Text ", Rows: 5},
							&ui.Checkbox{Name: "mySecondSetCheckbox", Label: "Set Checkbox"},
						},
					},
				},
			},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
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
