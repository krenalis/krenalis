//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package json implements the JSON connector.
// (https://datatracker.ietf.org/doc/html/rfc8259)
package json

import (
	"context"
	_ "embed"
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:       "JSON",
		Categories: meergo.CategoryFile,
		AsSource: &meergo.AsSourceFile{
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsDestinationFile{
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
		Extension: "json",
		Icon:      icon,
	}, New)
}

// New returns a new JSON connector instance.
func New(env *meergo.FileEnv) (*JSON, error) {
	c := JSON{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of JSON connector")
		}
	}
	return &c, nil
}

type JSON struct {
	env      *meergo.FileEnv
	settings *innerSettings
}

type innerSettings struct {
	Properties         []meergo.KV `json:",omitzero"`
	Indent             bool
	GenerateASCII      bool
	AllowSpecialFloats bool
}

var errInvalidJSON = errors.New("file does not contain valid JSON")
var errInvalidFormat = fmt.Errorf("file contains valid JSON, but its structure is not supported")

// ContentType returns the content type of the file.
func (j *JSON) ContentType(ctx context.Context) string {
	return "application/json; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (j *JSON) Read(ctx context.Context, r io.Reader, _ string, records meergo.RecordWriter) error {

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
func (j *JSON) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

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
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.KeyValue{
				Name:           "Properties",
				Label:          "Properties",
				KeyLabel:       "Name",
				ValueLabel:     "Required",
				KeyComponent:   &meergo.Input{Name: "Name", Placeholder: "Name", Rows: 1},
				ValueComponent: &meergo.Select{Name: "Required", Label: "Required", Options: []meergo.Option{{Text: "Required", Value: "t"}, {Text: "Optional", Value: "f"}}},
				Role:           meergo.Source,
			},
			&meergo.Checkbox{Name: "Indent", Label: "Indent the generated output", Role: meergo.Destination},
			&meergo.Checkbox{Name: "GenerateASCII", Label: "Generate an ASCII output, by escaping any non-ASCII Unicode", Role: meergo.Destination},
			&meergo.Checkbox{Name: "AllowSpecialFloats", Label: "Allow non-standard NaN, Infinity, and -Infinity values", Role: meergo.Destination},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (j *JSON) Write(ctx context.Context, w io.Writer, _ string, records meergo.RecordReader) error {
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
func (j *JSON) saveSettings(ctx context.Context, settings json.Value, role meergo.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Properties.
	if role == meergo.Source {
		if len(s.Properties) == 0 {
			return meergo.NewInvalidSettingsError("must have at least one property")
		}
		hasName := map[string]struct{}{}
		for _, property := range s.Properties {
			if _, ok := hasName[property.Key]; ok {
				return meergo.NewInvalidSettingsError(fmt.Sprintf("property name %q is repeated", property.Key))
			}
			if property.Key == "" {
				return meergo.NewInvalidSettingsError("a property name is empty")
			}
			if !types.IsValidPropertyName(property.Key) {
				return meergo.NewInvalidSettingsError(fmt.Sprintf("%q is not a valid property name. Property names must start"+
					" with a letter or underscore [A-Za-z_] and subsequently contain only letters, numbers, or underscores [A-Za-z0-9_]", property.Key))
			}
			if property.Value != "f" && property.Value != "t" {
				return meergo.NewInvalidSettingsError("required is not valid")
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
