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
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the File and UIHandler interfaces.
var _ interface {
	meergo.File
	meergo.UIHandler
} = (*JSON)(nil)

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:      "JSON",
		Icon:      icon,
		Extension: "json",
	}, New)
}

// New returns a new JSON connector instance.
func New(conf *meergo.FileConfig) (*JSON, error) {
	c := JSON{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of JSON connector")
		}
	}
	return &c, nil
}

type JSON struct {
	conf     *meergo.FileConfig
	settings *Settings
}

type Settings struct {
	Properties         []meergo.KV `json:",omitempty"`
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

	var tok json.Token
	dec := json.NewDecoder(r)
	dec.UseNumber()

	isObject := false

	jsonError := func(err error) error {
		if err == io.EOF {
			return errInvalidJSON
		}
		if _, ok := err.(*json.SyntaxError); ok {
			return errInvalidJSON
		}
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return errInvalidFormat
		}
		return err
	}

	// Read '[' or '{'.
	tok, err = dec.Token()
	if err != nil && tok == json.Delim('{') {
		isObject = true
		// Read a property name.
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
		if tok == json.Delim('}') {
			return errInvalidFormat
		}
		// Read '['.
		tok, err = dec.Token()
	}
	if err != nil {
		return jsonError(err)
	}
	if tok != json.Delim('[') {
		return errInvalidFormat
	}

	// Read the records.
	record := map[string]any{}
	for dec.More() {
		// Read '{...}'.
		err = dec.Decode(&record)
		if err != nil {
			return jsonError(err)
		}
		// Convert nil properties to json.RawMessage("null").
		for p, v := range record {
			if v == nil {
				record[p] = json.RawMessage("null")
			}
		}
		err = records.Record(record)
		if err != nil {
			return err
		}
		clear(record)
	}

	// Read ']'.
	tok, err = dec.Token()
	if err != nil {
		return jsonError(err)
	}

	// Read '}'.
	if isObject {
		tok, err = dec.Token()
		if err != nil {
			return jsonError(err)
		}
		if tok != json.Delim('}') {
			return errInvalidFormat
		}
	}

	// Read EOF.
	tok, err = dec.Token()
	if err != io.EOF {
		if err == nil {
			return errInvalidFormat
		}
		if _, ok := err.(*json.SyntaxError); ok {
			return errInvalidFormat
		}
		return err
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (j *JSON) ServeUI(ctx context.Context, event string, values []byte, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s Settings
		if j.settings != nil {
			s = *j.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, j.saveValues(ctx, values, role)
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
		Values: values,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (j *JSON) Write(ctx context.Context, w io.Writer, _ string, records meergo.RecordReader) error {
	s := j.settings
	enc := newEncoder(s.Indent, s.GenerateASCII, s.AllowSpecialFloats)
	var err error
	var record map[string]any
	var id string
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
		id, record, err = records.Record(ctx)
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
		records.Ack(id, nil)
	}
	if s.Indent {
		b = append(b, "\n\t]\n}"...)
	} else {
		b = append(b, ']', '}')
	}
	_, err = w.Write(b)
	return err
}

// saveValues saves the user-entered values as settings.
func (j *JSON) saveValues(ctx context.Context, values []byte, role meergo.Role) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Properties.
	if role == meergo.Source {
		if len(s.Properties) == 0 {
			return meergo.NewInvalidUIValuesError("must have at least one property")
		}
		hasName := map[string]struct{}{}
		for _, property := range s.Properties {
			if _, ok := hasName[property.Key]; ok {
				return meergo.NewInvalidUIValuesError(fmt.Sprintf("property name %q is repeated", property.Key))
			}
			if property.Key == "" {
				return meergo.NewInvalidUIValuesError("a property name is empty")
			}
			if !types.IsValidPropertyName(property.Key) {
				return meergo.NewInvalidUIValuesError(fmt.Sprintf("%q is not a valid property name. Property names must start"+
					" with a letter or underscore [A-Za-z_] and subsequently contain only letters, numbers, or underscores [A-Za-z0-9_]", property.Key))
			}
			if property.Value != "f" && property.Value != "t" {
				return meergo.NewInvalidUIValuesError("required is not valid")
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
	err = j.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	j.settings = &s
	return nil
}
