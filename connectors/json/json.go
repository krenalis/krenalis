//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package json

// This package is the JSON connector.
// (https://datatracker.ietf.org/doc/html/rfc8259)

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterFile(connector.File{
		Name:                   "JSON",
		DestinationDescription: "export users to a JSON file",
		Icon:                   icon,
		Extension:              "json",
	}, open)
}

type connection struct {
	ctx      context.Context
	role     connector.Role
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Indent             bool
	GenerateASCII      bool
	AllowSpecialFloats bool
}

// open opens a JSON connection and returns it.
func open(ctx context.Context, conf *connector.FileConfig) (*connection, error) {
	c := connection{ctx: ctx, role: conf.Role, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of JSON connection")
		}
	}
	return &c, nil
}

// ContentType returns the content type of the file.
func (c *connection) ContentType() string {
	return "application/json; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (c *connection) Read(r io.Reader, _ string, records connector.RecordWriter) error {
	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.SettingsUI(values)
		if err != nil {
			return nil, nil, err
		}
		err = c.firehose.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Switch{Name: "indent", Label: "Indent the generated output"},
			&ui.Switch{Name: "generateASCII", Label: "Generate an ASCII output, by escaping any non-ASCII Unicode"},
			&ui.Switch{Name: "allowSpecialFloats", Label: "Allow non-standard NaN, Infinity, and -Infinity values"},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// SettingsUI obtains the settings from UI values and returns them.
func (c *connection) SettingsUI(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

// Write writes to w the records read from records.
func (c *connection) Write(w io.Writer, _ string, records connector.RecordReader) error {
	s := c.settings
	enc := newEncoder(s.Indent, s.GenerateASCII, s.AllowSpecialFloats)
	var err error
	var record []any
	var comma bool
	b := make([]byte, 0, 4096)
	if s.Indent {
		b = append(b, "{\n\t\"records\": ["...)
		enc.depth = 2
	} else {
		b = append(b, `{"records":[`...)
	}
	t := types.Object(records.Columns())
	for {
		record, err = records.Record()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if comma {
			b = append(b, ',')
		} else {
			comma = true
		}
		if s.Indent {
			b = enc.appendIndentation(b)
		}
		b = enc.Append(b, t, record)
		if len(b) > cap(b)/2 {
			_, err = w.Write(b)
			if err != nil {
				return err
			}
			b = b[0:0]
		}
	}
	if s.Indent {
		b = append(b, "\n\t]\n}"...)
	} else {
		b = append(b, ']', '}')
	}
	_, err = w.Write(b)
	return err
}
