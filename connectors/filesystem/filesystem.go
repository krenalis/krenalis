//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package filesystem provides a connector for local file system.
package filesystem

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

var (
	root          string
	displayedRoot string
	confMu        sync.Mutex
)

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageSpec{
		Code:       "filesystem",
		Label:      "Filesystem",
		Categories: meergo.CategoryFileStorage,
		AsSource: &meergo.AsFileStorageSource{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsFileStorageDestination{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for file system.
func New(env *meergo.FileStorageEnv) (*Filesystem, error) {

	confMu.Lock()
	defer confMu.Unlock()

	// If root has not been set, it means that the configuration has not yet
	// been read from the environment variables, and therefore needs to be read
	// now.
	if root == "" {
		envVars, err := meergo.GetEnvVars()
		if err != nil {
			return nil, err
		}
		root = strings.TrimSpace(envVars.Get("MEERGO_CONNECTOR_FILESYSTEM_ROOT"))
		displayedRoot = strings.TrimSpace(envVars.Get("MEERGO_CONNECTOR_FILESYSTEM_DISPLAYED_ROOT"))
		const errMsgPrefix = "Filesystem connector is unavailable because the MEERGO_CONNECTOR_FILESYSTEM_ROOT environment variable"
		if root == "" {
			return nil, fmt.Errorf("%s is not set; please define it with the root directory to enable the connector", errMsgPrefix)
		}
		if err := validateRoot(root); err != nil {
			return nil, fmt.Errorf("%s has an invalid root: %s", errMsgPrefix, err)
		}
	}

	c := Filesystem{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for file system")
		}
	}
	return &c, nil
}

type Filesystem struct {
	env      *meergo.FileStorageEnv
	settings *innerSettings
}

type innerSettings struct {
	SimulateHighIOLatency bool
}

// AbsolutePath returns the absolute representation of the given path name.
func (filesystem *Filesystem) AbsolutePath(ctx context.Context, name string) (string, error) {
	return filesystem.absolutePath(ctx, name, true)
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (filesystem *Filesystem) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	path, _ := filesystem.absolutePath(ctx, name, false)
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

	var intro string
	if role == meergo.Source {
		intro = "This connector for file system allows Meergo to read files from this directory of your system:"
	} else {
		intro = "This connector for file system allows Meergo to write files into this directory of your system:"
	}

	confMu.Lock()
	defer confMu.Unlock()

	rootToShow := root
	if displayedRoot != "" {
		rootToShow = displayedRoot
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Text{Text: intro},
			&meergo.Text{Text: rootToShow},
			&meergo.Text{Label: "Testing options"},
			&meergo.Checkbox{Name: "SimulateHighIOLatency", Label: "Simulate high latency during I/O operations"},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (filesystem *Filesystem) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	path, _ := filesystem.absolutePath(ctx, name, false)
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err := os.Remove(tmpPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("connectors/filesystem: cannot remove temporary file created by filesystem", "err", err)
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

// absolutePath returns the absolute representation of the given path name.
//
// forDisplaying indicates whether the returned path will be used in a purely
// visual context, where it is necessary to use the displayed path, if
// available, or otherwise whether the returned path must be a real path on the
// filesystem (e.g. in cases where the connector needs to access files).
func (filesystem *Filesystem) absolutePath(ctx context.Context, name string, forDisplaying bool) (string, error) {
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
	confMu.Lock()
	defer confMu.Unlock()
	if forDisplaying && displayedRoot != "" {
		return filepath.Join(displayedRoot, name), nil
	}
	return filepath.Join(root, name), nil
}

// saveSettings saves the settings.
func (filesystem *Filesystem) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = filesystem.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	filesystem.settings = &s
	return nil
}

func validateRoot(root string) error {
	if n := len(root); n == 0 || n > 253 {
		return meergo.NewInvalidSettingsError("path length in bytes must be in range [1,253]")
	}
	if !filepath.IsAbs(root) {
		return meergo.NewInvalidSettingsError("path must be absolute")
	}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return meergo.NewInvalidSettingsError("path does not exist")
	}
	if !st.IsDir() {
		return meergo.NewInvalidSettingsError("path is not a directory")
	}
	return nil
}
