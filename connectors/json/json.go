// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package json provides a connector for JSON.
// (https://www.ietf.org/rfc/rfc8259.txt)
package json

import (
	"context"
	_ "embed"
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterFile(connectors.FileSpec{
		Code:       "json",
		Label:      "JSON",
		Categories: connectors.CategoryFile,
		AsSource: &connectors.AsSourceFile{
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsDestinationFile{
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
		Extension: "json",
	}, New)
}

// New returns a new connector instance for JSON.
func New(env *connectors.FileEnv) (*JSON, error) {
	c := JSON{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for JSON")
		}
	}
	return &c, nil
}

type JSON struct {
	env      *connectors.FileEnv
	settings *innerSettings
}

type innerSettings struct {
	Properties         []connectors.KV `json:"properties,omitzero"`
	Indent             bool            `json:"indent"`
	GenerateASCII      bool            `json:"generateASCII"`
	AllowSpecialFloats bool            `json:"allowSpecialFloats"`
}

var errInvalidJSON = errors.New("file does not contain valid JSON")
var errInvalidFormat = fmt.Errorf("file contains valid JSON, but its structure is not supported")

// ContentType returns the content type of the file.
func (j *JSON) ContentType(ctx context.Context) string {
	return "application/json; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (j *JSON) Read(ctx context.Context, r io.Reader, _ string, records connectors.RecordWriter) error {

	columns := make([]types.Property, 0, len(j.settings.Properties))
	for _, property := range j.settings.Properties {
		c := types.Property{
			Name: property.Key,
			Type: types.JSON(),
		}
		if property.Value == "f" {
			c.ReadOptional = true
		}
		columns = append(columns, c)
	}
	err := records.Columns(columns)
	if err != nil {
		return err
	}

	var tok jsonstd.Token
	dec := jsonstd.NewDecoder(r)
	dec.UseNumber()

	isObject := false

	jsonError := func(err error) error {
		if err == io.EOF {
			return errInvalidJSON
		}
		if _, ok := err.(*jsonstd.SyntaxError); ok {
			return errInvalidJSON
		}
		if _, ok := err.(*jsonstd.UnmarshalTypeError); ok {
			return errInvalidFormat
		}
		return err
	}

	// Read '[' or '{'.
	tok, err = dec.Token()
	if err != nil {
		return jsonError(err)
	}
	if tok == jsonstd.Delim('{') {
		isObject = true
		// Read a property name.
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
		if tok == jsonstd.Delim('}') {
			return errInvalidFormat
		}
		// Read '['.
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
	}
	if tok != jsonstd.Delim('[') {
		return errInvalidFormat
	}

	// Read the records.
	record := map[string]any{}
	for dec.More() {
		// Read '{...}'.
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
		if tok != jsonstd.Delim('{') {
			return errInvalidFormat
		}
		for dec.More() {
			tok, err = dec.Token()
			if err != nil {
				return jsonError(err)
			}
			name := tok.(string)
			var value json.Value
			err = dec.Decode(&value)
			if err != nil {
				return jsonError(err)
			}
			record[name] = value
		}
		if _, err = dec.Token(); err != nil {
			return jsonError(err)
		}
		err = records.Record(record)
		if err != nil {
			return err
		}
		clear(record)
	}

	// Read ']'.
	if _, err = dec.Token(); err != nil {
		return jsonError(err)
	}

	// Read '}'.
	if isObject {
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
		if tok != jsonstd.Delim('}') {
			return errInvalidFormat
		}
	}

	// Read EOF.
	_, err = dec.Token()
	if err != io.EOF {
		if err == nil {
			return errInvalidFormat
		}
		if _, ok := err.(*jsonstd.SyntaxError); ok {
			return errInvalidFormat
		}
		return err
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (j *JSON) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if j.settings != nil {
			s = *j.settings
		}
		settings, _ = jsonstd.Marshal(s)
	case "save":
		return nil, j.saveSettings(ctx, settings, role)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.KeyValue{
				Name:           "properties",
				Label:          "Properties",
				KeyLabel:       "Name",
				ValueLabel:     "Required",
				KeyComponent:   &connectors.Input{Name: "name", Placeholder: "Name", Rows: 1},
				ValueComponent: &connectors.Select{Name: "required", Label: "Required", Options: []connectors.Option{{Text: "Required", Value: "t"}, {Text: "Optional", Value: "f"}}},
				Role:           connectors.Source,
			},
			&connectors.Checkbox{Name: "indent", Label: "Indent the generated output", Role: connectors.Destination},
			&connectors.Checkbox{Name: "generateASCII", Label: "Generate an ASCII output, by escaping any non-ASCII Unicode", Role: connectors.Destination},
			&connectors.Checkbox{Name: "allowSpecialFloats", Label: "Allow non-standard NaN, Infinity, and -Infinity values", Role: connectors.Destination},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (j *JSON) Write(ctx context.Context, w io.Writer, _ string, records connectors.RecordReader) error {
	s := j.settings
	enc := newEncoder(s.Indent, s.GenerateASCII, s.AllowSpecialFloats)
	var err error
	var record map[string]any
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
		record, err = records.Record(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
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

// saveSettings saves the settings.
func (j *JSON) saveSettings(ctx context.Context, settings json.Value, role connectors.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Properties.
	if role == connectors.Source {
		if len(s.Properties) == 0 {
			return connectors.NewInvalidSettingsError("must have at least one property")
		}
		hasName := map[string]struct{}{}
		for _, property := range s.Properties {
			if _, ok := hasName[property.Key]; ok {
				return connectors.NewInvalidSettingsError(fmt.Sprintf("property name %q is repeated", property.Key))
			}
			if property.Key == "" {
				return connectors.NewInvalidSettingsError("a property name is empty")
			}
			if !types.IsValidPropertyName(property.Key) {
				return connectors.NewInvalidSettingsError(fmt.Sprintf("%q is not a valid property name. Property names must start"+
					" with a letter or underscore [A-Za-z_] and subsequently contain only letters, numbers, or underscores [A-Za-z0-9_]", property.Key))
			}
			if property.Value != "f" && property.Value != "t" {
				return connectors.NewInvalidSettingsError("required is not valid")
			}
			hasName[property.Key] = struct{}{}
		}
	} else {
		s.Properties = nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = j.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	j.settings = &s
	return nil
}
