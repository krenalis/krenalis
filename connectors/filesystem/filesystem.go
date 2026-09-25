// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package filesystem provides a connector for local file system.
package filesystem

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	fsPkg "io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
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
	connectors.RegisterFileStorage(connectors.FileStorageSpec{
		Code:       "filesystem",
		Label:      "File System",
		Categories: connectors.CategoryFileStorage,
		AsSource: &connectors.AsFileStorageSource{
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsFileStorageDestination{
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for File System.
func New(env *connectors.FileStorageEnv) (*FileSystem, error) {

	confMu.Lock()
	defer confMu.Unlock()

	// If root has not been set, it means that the configuration has not yet
	// been read from the environment variables, and therefore needs to be read
	// now.
	if root == "" {
		root = strings.TrimSpace(os.Getenv("KRENALIS_CONNECTOR_FILESYSTEM_ROOT"))
		displayedRoot = strings.TrimSpace(os.Getenv("KRENALIS_CONNECTOR_FILESYSTEM_DISPLAYED_ROOT"))
		const errMsgPrefix = "File System connector is unavailable because the KRENALIS_CONNECTOR_FILESYSTEM_ROOT environment variable"
		if root == "" {
			return nil, fmt.Errorf("%s is not set; please define it with the root directory to enable the connector", errMsgPrefix)
		}
		if err := validateRoot(root); err != nil {
			return nil, fmt.Errorf("%s has an invalid root: %s", errMsgPrefix, err)
		}
	}

	return &FileSystem{env: env}, nil
}

type FileSystem struct {
	env *connectors.FileStorageEnv
}

type innerSettings struct {
	SimulateHighIOLatency bool `json:"simulateHighIOLatency"`
}

// AbsolutePath returns the absolute representation of the given path name.
func (fs *FileSystem) AbsolutePath(ctx context.Context, name string) (string, error) {
	name, err := parseName(name)
	if err != nil {
		return "", err
	}
	confMu.Lock()
	defer confMu.Unlock()
	if displayedRoot != "" {
		return filepath.Join(displayedRoot, name), nil
	}
	return filepath.Join(root, name), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (fs *FileSystem) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	name, err := parseName(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	dir, err := openRoot()
	if err != nil {
		return nil, time.Time{}, rewritePathError(err)
	}
	defer dir.Close()
	f, err := dir.Open(name)
	if err != nil {
		return nil, time.Time{}, rewritePathError(err)
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, rewritePathError(err)
	}
	var rc io.ReadCloser = f
	var s innerSettings
	err = fs.env.Settings.Load(ctx, &s)
	if err != nil {
		return nil, time.Time{}, err
	}
	if s.SimulateHighIOLatency {
		rc = &highLatencyReadCloser{rc}
	}
	return rc, fi.ModTime().UTC(), nil
}

// ServeUI serves the connector's user interface.
func (fs *FileSystem) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		err := fs.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, fs.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	var intro string
	if role == connectors.Source {
		intro = "This connector for file system allows Krenalis to read files from this directory of your system:"
	} else {
		intro = "This connector for file system allows Krenalis to write files into this directory of your system:"
	}

	confMu.Lock()
	defer confMu.Unlock()

	rootToShow := root
	if displayedRoot != "" {
		rootToShow = displayedRoot
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Text{Text: intro},
			&connectors.Text{Text: rootToShow},
			&connectors.Text{Label: "Testing options"},
			&connectors.Checkbox{Name: "simulateHighIOLatency", Label: "Simulate high latency during I/O operations"},
		},
		Settings: settings,
		Buttons:  []connectors.Button{connectors.SaveButton},
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (fs *FileSystem) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	name, err := parseName(name)
	if err != nil {
		return err
	}
	dir, err := openRoot()
	if err != nil {
		return rewritePathError(err)
	}
	defer dir.Close()
	tmpName := name + ".tmp"
	f, err := dir.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return rewritePathError(err)
	}
	defer func() {
		err := dir.Remove(tmpName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			err = rewritePathError(err)
			slog.Warn("connectors/filesystem: cannot remove temporary file created by File System", "error", err)
			return
		}
	}()
	var s innerSettings
	err = fs.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}
	if s.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	_, err = io.Copy(f, r)
	if s.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	err2 := f.Close()
	if err != nil {
		return rewritePathError(err)
	}
	if err2 != nil {
		return rewritePathError(err2)
	}
	if s.SimulateHighIOLatency {
		simulateHighIOLatency()
	}
	err = dir.Rename(tmpName, name)
	return rewritePathError(err)
}

// saveSettings saves the settings.
func (fs *FileSystem) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	return fs.env.Settings.Store(ctx, s)
}

// openRoot opens the root directory.
func openRoot() (*os.Root, error) {
	confMu.Lock()
	defer confMu.Unlock()
	return os.OpenRoot(root)
}

// parseName parses the path name name, which may begin with a slash and, on
// Windows, use backslashes as separators, and returns it in the form used by
// io/fs and os.Root. It returns an *InvalidPathError if name is not valid or
// does not refer to a file.
func parseName(name string) (string, error) {
	rel := strings.TrimPrefix(filepath.ToSlash(name), "/")
	if rel == "" {
		return "", connectors.InvalidPathErrorf("path name cannot be “%s”", name)
	}
	if strings.HasSuffix(rel, "/") {
		return "", connectors.InvalidPathErrorf("path name cannot end with a slash")
	}
	if rel == "." || !fsPkg.ValidPath(rel) {
		return "", connectors.InvalidPathErrorf("path name cannot contain “.” or “..” or empty elements")
	}
	return rel, nil
}

// rewritePathError, if err is a *fs.PathError or an *os.LinkError error,
// returns a new error of the same type such that its paths are absolute and
// consistent with the displayed root of the connection, if set.
//
// For all other error types, or if the error is nil, the error is returned as
// it is.
func rewritePathError(err error) error {

	confMu.Lock()
	defer confMu.Unlock()

	rootToShow := root
	if displayedRoot != "" {
		rootToShow = displayedRoot
	}

	// rewrite removes from path the prefix that refers to the root, if
	// present, as errors returned by the os.Root methods have a path relative
	// to the root, and prepends the root to show.
	rewrite := func(path string) string {
		return filepath.Join(rootToShow, strings.TrimPrefix(path, root))
	}

	switch e := err.(type) {
	case *fsPkg.PathError:
		return &fsPkg.PathError{Op: e.Op, Path: rewrite(e.Path), Err: e.Err}
	case *os.LinkError:
		return &os.LinkError{Op: e.Op, Old: rewrite(e.Old), New: rewrite(e.New), Err: e.Err}
	}

	return err
}

func validateRoot(root string) error {
	if n := len(root); n == 0 || n > 253 {
		return connectors.NewInvalidSettingsError("path length in bytes must be in range [1,253]")
	}
	if !filepath.IsAbs(root) {
		return connectors.NewInvalidSettingsError("path must be absolute")
	}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return connectors.NewInvalidSettingsError("path does not exist")
	}
	if !st.IsDir() {
		return connectors.NewInvalidSettingsError("path is not a directory")
	}
	return nil
}
