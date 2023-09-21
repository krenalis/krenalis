//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"chichi/apis/transformers"
)

type transformer struct {
	settings Settings
}

type Settings struct {
	NodeExecutable   string // eg. "/usr/bin/node".
	PythonExecutable string // eg. "/usr/bin/python".
	Language         string // "node" or "python".
	FunctionsDir     string
}

func New(settings Settings) transformers.Transformer {
	return &transformer{settings: settings}
}

// CallFunction calls the function with the given name an version, with the
// given values to transform, and returns the results. If an error occurred
// during its execution, it returns an ExecutionError error.
func (tr *transformer) CallFunction(ctx context.Context, name, version string, values []map[string]any) ([]transformers.Result, error) {
	// Considering that this transformer is used for local testing and
	// development, it is recommended to return errors as
	// transformers.ExecutionError instead of internal errors, so the user of
	// Chichi can see them in the UI.
	jsonValues, err := json.Marshal(values)
	if err != nil {
		return nil, transformers.NewExecutionError(err.Error())
	}
	versionInt, err := strconv.Atoi(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q", version)
	}
	var stdout, stderr bytes.Buffer
	filename := tr.absFilename(name, versionInt)
	var executable string
	switch tr.settings.Language {
	case "node":
		executable = tr.settings.NodeExecutable
	case "python":
		executable = tr.settings.PythonExecutable
	default:
		return nil, fmt.Errorf("invalid language %q", tr.settings.Language)
	}
	cmd := exec.CommandContext(ctx, executable, filename, string(jsonValues))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, transformers.NewExecutionError(stderr.String())
	}
	var results []transformers.Result
	err = json.Unmarshal(stdout.Bytes(), &results)
	if err != nil {
		return nil, transformers.NewExecutionError(err.Error())
	}
	return results, nil
}

// Close closes the transformer.
func (tr *transformer) Close(ctx context.Context) error {
	return nil
}

// CreateFunction creates a new function with the given name and source, and
// returns its version, which has a length in the range [1, 128]. If a function
// with the same name already exists, it returns the ErrExist error.
func (tr *transformer) CreateFunction(ctx context.Context, name, source string) (string, error) {
	// TODO(Gianluca): on Windows, escape reserved filenames.
	// See https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file?redirectedfrom=MSDN.
	err := tr.createFunction(name, 1, source)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", transformers.ErrExist
		}
		return "", err
	}
	return "1", nil
}

// createFunction creates the function.
// If the function already exists, it returns an error matching the os.ErrExist
// error.
func (tr *transformer) createFunction(name string, version int, source string) error {
	switch tr.settings.Language {
	case "node":
		source += `
const results = [];
const event = JSON.parse(process.argv[2]);
for ( let i = 0; i < event.length; i++ ) {
	try {
		let value = transform(event[i]);
		results[i] = { "value": value };
	} catch (error) {
		results[i] = { "error": error };
	}
}
process.stdout.write(JSON.stringify(results))`
	case "python":
		source += `
if __name__ == "__main__":
	import json
	import sys

	results = []
	events = json.loads(sys.argv[1])
	for event in events:
		try:
			value = transform(event)
		except Exception as ex:
			results.append({"error": str(ex)})
		else:
			results.append({"value": value})
	print(json.dumps(results))
`
	default:
		return fmt.Errorf("invalid language %q", tr.settings.Language)
	}
	filename := tr.absFilename(name, version)
	err := os.WriteFile(filename, []byte(source), 0644)
	return err
}

// DeleteFunction deletes the function with the given name.
// If a function with the given name does not exist, it does nothing.
func (tr *transformer) DeleteFunction(ctx context.Context, name string) error {
	entries, err := os.ReadDir(tr.settings.FunctionsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		v, ok := filenameToVersion(name, entry.Name(), tr.settings.Language)
		if ok {
			err := os.Remove(tr.absFilename(name, v))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

// UpdateFunction updates the source of the function with the given name, and
// returns a new version, which has a length in the range [1, 128]. If the
// function does not exist, it returns the ErrNotExist error.
func (tr *transformer) UpdateFunction(ctx context.Context, name, source string) (string, error) {
	attempts := 0
fileCreation:
	for {
		entries, err := os.ReadDir(tr.settings.FunctionsDir)
		if err != nil {
			return "", err
		}
		// Filenames for functions should be like: "<name>_v<version>.<ext>"
		var maxVersion int
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			v, ok := filenameToVersion(name, entry.Name(), tr.settings.Language)
			if ok && v > maxVersion {
				maxVersion = v
			}
		}
		if maxVersion == 0 {
			return "", transformers.ErrNotExist
		}
		err = tr.createFunction(name, maxVersion+1, source)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				attempts++
				if attempts >= 10 {
					return "", fmt.Errorf("unable to create file after %d attempts in which the file already existed", attempts)
				}
				continue fileCreation
			}
			return "", err
		}
		return strconv.Itoa(maxVersion + 1), nil
	}
}

// filenameToVersion extracts the version from the filename relative to a
// function with the given name and language.
//
// For example, if a function is named "action-12345" and the filename is
// "12345_v10.py", then "10" and "true" are returned.
//
// The boolean value reports whether the filename (and thus the returned
// version) is valid for the given funcName or not.
func filenameToVersion(funcName, filename, language string) (int, bool) {
	s := strings.TrimPrefix(filename, funcName+"_v")
	if s == filename {
		return 0, false
	}
	var ext string
	switch language {
	case "node":
		ext = ".js"
	case "python":
		ext = ".py"
	default:
		panic("invalid language")
	}
	s2 := strings.TrimSuffix(s, ext)
	if s2 == s {
		return 0, false
	}
	v, err := strconv.Atoi(s2)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (tr *transformer) absFilename(name string, version int) string {
	var ext string
	switch tr.settings.Language {
	case "node":
		ext = ".js"
	case "python":
		ext = ".py"
	default:
		panic("invalid language")
	}
	return filepath.Join(tr.settings.FunctionsDir, fmt.Sprintf("%s_v%d%s", name, version, ext))
}
