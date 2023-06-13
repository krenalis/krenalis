//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package filesystem implements the Filesystem connector.
package filesystem

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI and the StorageConnection interfaces.
var _ interface {
	connector.UI
	connector.StorageConnection
} = (*connection)(nil)

func init() {
	connector.RegisterStorage(connector.Storage{
		Name: "Filesystem",
		Icon: icon,
	}, open)
}

// open opens a Filesystem connection and returns it.
func open(ctx context.Context, conf *connector.StorageConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Filesystem connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	conf     *connector.StorageConfig
	settings *settings
}

type settings struct {
	Root string
}

// CompletePath returns the complete representation of the given path name or an
// InvalidPathError if name is not valid for use in calls to Open and Write.
func (c *connection) CompletePath(name string) (string, error) {
	originalName := name
	name = filepath.ToSlash(name)
	if name[0] == '/' {
		if name == "/" {
			return "", connector.InvalidPathErrorf("path name cannot be “" + originalName + "“")
		}
		name = name[1:]
	}
	if name[len(name)-1] == '/' {
		return "", connector.InvalidPathErrorf("path name cannot end with a slash")
	}
	if name == "." || !fs.ValidPath(name) {
		return "", connector.InvalidPathErrorf("path name cannot contains “.” or “..” or empty elements")
	}
	return filepath.Join(c.settings.Root, name), nil
}

// Open opens the file at the given path name and returns a ReadCloser from
// which to read the file and its last update time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Open(name string) (io.ReadCloser, time.Time, error) {
	path, _ := c.CompletePath(name)
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	return f, fi.ModTime(), nil
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
		s, err := c.ValidateSettings(values)
		if err != nil {
			return nil, nil, err
		}
		err = c.conf.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Text{Label: "Warning", Text: "The Filesystem connector exposes you local filesystem to Chichi for read and write operations. Use this with caution."},
			&ui.Input{Name: "Root", Label: "Root Path", HelpText: "Path to an existent directory of the local filesystem which will be used as the root for the Filesystem storage.", Placeholder: "/home/user/my/dir", Type: "text", MinLength: 1, MaxLength: 253},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Root.
	root := s.Root
	if n := len(root); n == 0 || n > 253 {
		return nil, ui.Errorf("root path length in bytes must be in range [1,253]")
	}
	if !filepath.IsAbs(root) {
		return nil, ui.Errorf(`root path must be absolute`)
	}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil, ui.Errorf("root path does not exist")
	}
	if !st.IsDir() {
		return nil, ui.Errorf("root path is not a directory")
	}
	return json.Marshal(&s)
}

// Write writes the data read from r into the file with the given path name.
// contentType is the file's content type.
func (c *connection) Write(r io.Reader, name, contentType string) error {
	path, _ := c.CompletePath(name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	err2 := f.Close()
	if err != nil {
		return err
	}
	return err2
}
