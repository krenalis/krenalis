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
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Name:          "Filesystem",
		AsSource:      true,
		AsDestination: true,
		Icon:          icon,
	}, New)
}

// New returns a new Filesystem connector instance.
func New(conf *meergo.FileStorageConfig) (*Filesystem, error) {
	c := Filesystem{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Filesystem connector")
		}
	}
	return &c, nil
}

type Filesystem struct {
	conf     *meergo.FileStorageConfig
	settings *innerSettings
}

type innerSettings struct {
	Root                  string
	SimulateHighIOLatency bool
}

// AbsolutePath returns the absolute representation of the given path name.
func (filesystem *Filesystem) AbsolutePath(ctx context.Context, name string) (string, error) {
	originalName := name
	name = filepath.ToSlash(name)
	if name[0] == '/' {
		if name == "/" {
			return "", meergo.InvalidPathErrorf("path name cannot be “%s“", originalName)
		}
		name = name[1:]
	}
	if name[len(name)-1] == '/' {
		return "", meergo.InvalidPathErrorf("path name cannot end with a slash")
	}
	if name == "." || !fs.ValidPath(name) {
		return "", meergo.InvalidPathErrorf("path name cannot contains “.” or “..” or empty elements")
	}
	return filepath.Join(filesystem.settings.Root, name), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (filesystem *Filesystem) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	path, _ := filesystem.AbsolutePath(ctx, name)
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	var rc io.ReadCloser = f
	if filesystem.settings.SimulateHighIOLatency {
		rc = &highLatencyReadCloser{rc}
	}
	return rc, fi.ModTime().UTC(), nil
}

// ServeUI serves the connector's user interface.
func (filesystem *Filesystem) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if filesystem.settings != nil {
			s = *filesystem.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, filesystem.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Text{Label: "Warning", Text: "The Filesystem connector exposes you local filesystem to Meergo for read and write operations. Use this with caution."},
			&meergo.Input{Name: "Root", Label: "Root Path", HelpText: "Path to an existent directory of the local filesystem which will be used as the root for the Filesystem storage.", Placeholder: "/home/user/my/dir", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Checkbox{Name: "SimulateHighIOLatency", Label: "Simulate high latency during I/O operations"},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (filesystem *Filesystem) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	path, _ := filesystem.AbsolutePath(ctx, name)
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err := os.Remove(tmpPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("cannot remove temporary file created by filesystem", "err", err)
			return
		}
	}()
	if filesystem.settings.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	_, err = io.Copy(f, r)
	if filesystem.settings.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	err2 := f.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	if filesystem.settings.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	err = os.Rename(tmpPath, path)
	return err
}

// saveSettings saves the settings.
func (filesystem *Filesystem) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Root.
	root := s.Root
	if n := len(root); n == 0 || n > 253 {
		return meergo.NewInvalidsettingsError("root path length in bytes must be in range [1,253]")
	}
	if !filepath.IsAbs(root) {
		return meergo.NewInvalidsettingsError(`root path must be absolute`)
	}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return meergo.NewInvalidsettingsError("root path does not exist")
	}
	if !st.IsDir() {
		return meergo.NewInvalidsettingsError("root path is not a directory")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = filesystem.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	filesystem.settings = &s
	return nil
}
