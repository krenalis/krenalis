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
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/open2b/chichi"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the FileStorage and the UIHandler interfaces.
var _ interface {
	chichi.FileStorage
	chichi.UIHandler
} = (*Filesystem)(nil)

func init() {
	chichi.RegisterFileStorage(chichi.FileStorageInfo{
		Name: "Filesystem",
		Icon: icon,
	}, New)
}

// New returns a new Filesystem connector instance.
func New(conf *chichi.FileStorageConfig) (*Filesystem, error) {
	c := Filesystem{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Filesystem connector")
		}
	}
	return &c, nil
}

type Filesystem struct {
	conf     *chichi.FileStorageConfig
	settings *Settings
}

type Settings struct {
	Root string
}

// CompletePath returns the complete representation of the given path name.
func (filesystem *Filesystem) CompletePath(ctx context.Context, name string) (string, error) {
	originalName := name
	name = filepath.ToSlash(name)
	if name[0] == '/' {
		if name == "/" {
			return "", chichi.InvalidPathErrorf("path name cannot be “" + originalName + "“")
		}
		name = name[1:]
	}
	if name[len(name)-1] == '/' {
		return "", chichi.InvalidPathErrorf("path name cannot end with a slash")
	}
	if name == "." || !fs.ValidPath(name) {
		return "", chichi.InvalidPathErrorf("path name cannot contains “.” or “..” or empty elements")
	}
	return filepath.Join(filesystem.settings.Root, name), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (filesystem *Filesystem) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	path, _ := filesystem.CompletePath(ctx, name)
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	return f, fi.ModTime().UTC(), nil
}

// ServeUI serves the connector's user interface.
func (filesystem *Filesystem) ServeUI(ctx context.Context, event string, values []byte, role chichi.Role) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if filesystem.settings != nil {
			s = *filesystem.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, filesystem.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Text{Label: "Warning", Text: "The Filesystem connector exposes you local filesystem to Chichi for read and write operations. Use this with caution."},
			&chichi.Input{Name: "Root", Label: "Root Path", HelpText: "Path to an existent directory of the local filesystem which will be used as the root for the Filesystem storage.", Placeholder: "/home/user/my/dir", Type: "text", MinLength: 1, MaxLength: 253},
		},
		Values: values,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (filesystem *Filesystem) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	path, _ := filesystem.CompletePath(ctx, name)
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
	_, err = io.Copy(f, r)
	err2 := f.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	err = os.Rename(tmpPath, path)
	return err
}

// saveValues saves the user-entered values as settings.
func (filesystem *Filesystem) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Root.
	root := s.Root
	if n := len(root); n == 0 || n > 253 {
		return chichi.NewInvalidUIValuesError("root path length in bytes must be in range [1,253]")
	}
	if !filepath.IsAbs(root) {
		return chichi.NewInvalidUIValuesError(`root path must be absolute`)
	}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return chichi.NewInvalidUIValuesError("root path does not exist")
	}
	if !st.IsDir() {
		return chichi.NewInvalidUIValuesError("root path is not a directory")
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
