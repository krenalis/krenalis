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
	"os"
	"path/filepath"
	"time"

	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterStorage(connector.Storage{
		Name: "Filesystem",
		Icon: icon,
	}, open)
}

type connection struct {
	ctx         context.Context
	settings    *settings
	setSettings connector.SetSettingsFunc
}

type settings struct {
	Root string
}

// open opens a Filesystem connection and returns it.
func open(ctx context.Context, conf *connector.StorageConfig) (*connection, error) {
	c := connection{ctx: ctx, setSettings: conf.SetSettings}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Filesystem connection")
		}
	}
	return &c, nil
}

// Open opens the file at the given path and returns a ReadCloser from which to
// read the file and its last update time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Open(path string) (io.ReadCloser, time.Time, error) {
	filePath := c.filesystemPath(path)
	f, err := os.Open(filePath)
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
		err = c.setSettings(s)
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
	if n := len(s.Root); n == 0 || n > 253 {
		return nil, ui.Errorf("root path length in bytes must be in range [1,253]")
	}
	if _, err := os.Stat(s.Root); os.IsNotExist(err) {
		return nil, ui.Errorf("root path does not exist")
	}
	return json.Marshal(&s)
}

// Write writes the data read from p into the file with the given path.
// contentType is the file's content type.
func (c *connection) Write(r io.Reader, path, contentType string) error {
	filePath := c.filesystemPath(path)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

// filesystemPath returns the path on the filesystem for the path relative to
// the storage.
func (c *connection) filesystemPath(path string) string {
	if c.settings.Root == "" {
		panic("invalid or corrupted settings")
	}
	return filepath.Join(c.settings.Root, path)
}
