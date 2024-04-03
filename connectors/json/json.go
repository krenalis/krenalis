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

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the File and UI interfaces.
var _ interface {
	chichi.File
	chichi.UI
} = (*JSON)(nil)

func init() {
	chichi.RegisterFile(chichi.FileInfo{
		Name:      "JSON",
		Icon:      icon,
		Extension: "json",
	}, New)
}

// New returns a new JSON connector instance.
func New(conf *chichi.FileConfig) (*JSON, error) {
	c := JSON{role: conf.Role, setSettings: conf.SetSettings}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of JSON connector")
		}
	}
	return &c, nil
}

type JSON struct {
	role        chichi.Role
	settings    *settings
	setSettings chichi.SetSettingsFunc
}

type settings struct {
	Indent             bool
	GenerateASCII      bool
	AllowSpecialFloats bool
}

// ContentType returns the content type of the file.
func (j *JSON) ContentType(ctx context.Context) string {
	return "application/json; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (j *JSON) Read(ctx context.Context, r io.Reader, _ string, records chichi.RecordWriter) error {

	var err error
	var tok json.Token

	dec := json.NewDecoder(r)

	// Read "[{".
	for {
		tok, err = dec.Token()
		if err != nil {
			break
		}
		if tok == json.Delim('[') {
			tok, err = dec.Token()
			if err != nil || tok == json.Delim('{') {
				break
			}
		}
	}
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	// Read the records.
	nameOfKey := map[string]string{}
	columns := make([]types.Property, 0, 10)
	record := map[string]any{}
Records:
	for {
		tok, err = dec.Token()
		if err != nil {
			break
		}
		switch tok := tok.(type) {
		case string:
			var key = tok
			var value any
			err = dec.Decode(&value)
			if err != nil {
				break Records
			}
			var name string
			if columns == nil {
				var ok bool
				name, ok = nameOfKey[key]
				if !ok {
					return fmt.Errorf("key %q does not exist for the first object", key)
				}
			} else {
				name = chichi.SuggestPropertyName(key)
				if name == "" {
					return fmt.Errorf("key %q cannot be converted to a valid property name", key)
				}
				for n, k := range nameOfKey {
					if name == n {
						if key == k {
							return fmt.Errorf("key %q is repeated", key)
						}
						return fmt.Errorf("keys %q and %q cannot be converted into two different property names", key, k)
					}
				}
				columns = append(columns, types.Property{
					Name:     name,
					Type:     types.JSON(),
					Nullable: true,
				})
				nameOfKey[key] = name
			}
			record[name] = value
		case json.Delim:
			switch tok {
			case '}':
				if columns != nil {
					err = records.Columns(columns)
					if err != nil {
						return err
					}
					columns = nil
				}
				err = records.RecordMap(record)
				if err != nil {
					return err
				}
			case '{':
				for k := range record {
					delete(record, k)
				}
			case ']':
				break Records
			}
		default:
			panic("unreachable code")
		}
	}
	for err == nil {
		_, err = dec.Token()
	}
	if err != nil && err != io.EOF {
		return fmt.Errorf("file contains invalid JSON: %s", err)
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (j *JSON) ServeUI(ctx context.Context, event string, values []byte) (*chichi.Form, *chichi.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if j.settings != nil {
			s = *j.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := j.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = j.setSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, chichi.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, chichi.ErrEventNotExist
	}

	form := &chichi.Form{
		Fields: []chichi.Component{
			&chichi.Switch{Name: "indent", Label: "Indent the generated output", Role: chichi.Destination},
			&chichi.Switch{Name: "generateASCII", Label: "Generate an ASCII output, by escaping any non-ASCII Unicode", Role: chichi.Destination},
			&chichi.Switch{Name: "allowSpecialFloats", Label: "Allow non-standard NaN, Infinity, and -Infinity values", Role: chichi.Destination},
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
func (j *JSON) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

// Write writes to w the records read from records.
func (j *JSON) Write(ctx context.Context, w io.Writer, _ string, records chichi.RecordReader) error {
	s := j.settings
	enc := newEncoder(s.Indent, s.GenerateASCII, s.AllowSpecialFloats)
	var err error
	var record []any
	var gid int
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
		gid, record, err = records.Record(ctx)
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
		records.Ack(gid, nil)
	}
	if s.Indent {
		b = append(b, "\n\t]\n}"...)
	} else {
		b = append(b, ']', '}')
	}
	_, err = w.Write(b)
	return err
}
