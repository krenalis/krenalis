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
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/connector/types"
)

type transformer struct {
	settings Settings
}

type Settings struct {
	NodeExecutable   string // eg. "/usr/bin/node".
	PythonExecutable string // eg. "/usr/bin/python".
	FunctionsDir     string
}

func New(settings Settings) transformers.Transformer {
	return &transformer{settings: settings}
}

// CallFunction calls the function with the given name and version, with the
// given values to transform, and returns the results. If an error occurs during
// execution, it returns an *ExecutionError error. If the function does not
// exist, it returns the ErrNotExist error. If the function is in a pending
// state, it returns the ErrPendingState error.
func (tr *transformer) CallFunction(ctx context.Context, name, version string, schema types.Type, values []map[string]any) ([]transformers.Result, error) {
	name, ext, err := splitName(name)
	if err != nil {
		return nil, err
	}
	if !tr.supportLanguage(ext) {
		return nil, errors.New("language is not supported")
	}

	versionInt, err := strconv.Atoi(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q", version)
	}
	var stdout, stderr bytes.Buffer
	filename := tr.absFilename(name, versionInt, ext)
	var executable string
	var payload []byte
	switch ext {
	case ".js":
		executable = tr.settings.NodeExecutable
		payload = make([]byte, 0, 1024)
		payload = transformers.MarshalJavaScript(payload, schema, values)
	case ".py":
		executable = tr.settings.PythonExecutable
		payload, err = json.Marshal(values)
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, transformers.ErrNotExist
		}
		return nil, err
	}
	cmd := exec.CommandContext(ctx, executable, filename, string(payload))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, transformers.NewExecutionError(stderr.String())
	}
	var results []transformers.Result
	err = json.Unmarshal(stdout.Bytes(), &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Close closes the transformer.
func (tr *transformer) Close(ctx context.Context) error {
	return nil
}

// CreateFunction creates a new function with the given name and source, and
// returns its version, which has a length in the range [1, 128]. name should
// have an extension of either ".js" or ".py" depending on the source code's
// language. If a function with the same name already exists, it returns the
// ErrExist error.
func (tr *transformer) CreateFunction(ctx context.Context, name, source string) (string, error) {
	// TODO(Gianluca): on Windows, escape reserved filenames.
	// See https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file?redirectedfrom=MSDN.
	err := tr.createFunction(name, 1, source)
	if err != nil {
		return "", err
	}
	return "1", nil
}

// createFunction creates a function with the given name and version.
// If the function already exists, it returns the transformers.ErrExist error.
func (tr *transformer) createFunction(name string, version int, source string) error {
	name, ext, err := splitName(name)
	if err != nil {
		return err
	}
	if !tr.supportLanguage(ext) {
		return errors.New("language is not supported")
	}
	switch ext {
	case ".js":
		source += `
BigInt.prototype.toJSON = function() { return this.toString(); }
const results = [];
const event = Function("return " + process.argv[2])();
for ( let i = 0; i < event.length; i++ ) {
	try {
		let value = transform(event[i]);
		results[i] = { value: value };
	} catch (error) {
		if (error instanceof Error) {
			error = error.toString();
		} else {
			error = "throw error of type " + (typeof error) + ": " + JSON.stringify(error);
		}
		results[i] = { error: error };
	}
}
process.stdout.write(JSON.stringify(results))`
	case ".py":
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
	}
	filename := tr.absFilename(name, version, ext)
	err = os.WriteFile(filename, []byte(source), 0644)
	if err != nil && errors.Is(err, os.ErrExist) {
		err = transformers.ErrExist
	}
	return nil
}

// DeleteFunction deletes the function with the given name.
// If a function with the given name does not exist, it does nothing.
func (tr *transformer) DeleteFunction(ctx context.Context, name string) error {
	name, ext, err := splitName(name)
	if err != nil {
		return err
	}
	if !tr.supportLanguage(ext) {
		return errors.New("language is not supported")
	}
	entries, err := os.ReadDir(tr.settings.FunctionsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		v, ok := filenameToVersion(name, entry.Name(), ext)
		if ok {
			err := os.Remove(tr.absFilename(name, v, ext))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

// SupportLanguage reports whether language is supported as a language.
// It panics if language is not valid.
func (tr *transformer) SupportLanguage(language state.Language) bool {
	switch language {
	case state.JavaScript:
		return tr.settings.NodeExecutable != ""
	case state.Python:
		return tr.settings.PythonExecutable != ""
	}
	panic("invalid language")
}

// UpdateFunction updates the source of the function with the given name, and
// returns a new version, which has a length in the range [1, 128]. If the
// function does not exist, it returns the ErrNotExist error.
func (tr *transformer) UpdateFunction(ctx context.Context, name, source string) (string, error) {
	name, ext, err := splitName(name)
	if err != nil {
		return "", err
	}
	if !tr.supportLanguage(ext) {
		return "", errors.New("language is not supported")
	}
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
			v, ok := filenameToVersion(name, entry.Name(), ext)
			if ok && v > maxVersion {
				maxVersion = v
			}
		}
		if maxVersion == 0 {
			return "", transformers.ErrNotExist
		}
		if maxVersion == math.MaxInt64 {
			return "", errors.New("too many versions")
		}
		err = tr.createFunction(name+ext, maxVersion+1, source)
		if err != nil {
			if err == transformers.ErrExist {
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
// function with the given name.
//
// For example, if a function is named "action-12345.py" and the filename is
// "action-12345_v10.py", then "10" and "true" are returned.
//
// The boolean value reports whether the filename (and thus the returned
// version) is valid for the given name or not.
func filenameToVersion(name, filename, ext string) (int, bool) {
	s, ok := strings.CutPrefix(filename, name+"_v")
	if !ok {
		return 0, false
	}
	s, ok = strings.CutSuffix(s, ext)
	if !ok {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if (i == 0 && s[i] == '0') || s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	v, err := strconv.Atoi(s)
	return v, err == nil
}

func (tr *transformer) absFilename(name string, version int, ext string) string {
	return filepath.Join(tr.settings.FunctionsDir, fmt.Sprintf("%s_v%d%s", name, version, ext))
}

// splitName splits a function returning the name without the extension and the
// extension. It returns an error if the name is not valid.
func splitName(name string) (string, string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", "", errors.New("function name is not valid")
	}
	return name[:len(name)-3], name[len(name)-3:], nil
}

// supportLanguage is like SupportLanguage but gets an extension as argument.
func (tr *transformer) supportLanguage(ext string) bool {
	switch ext {
	case ".js":
		return tr.settings.NodeExecutable != ""
	case ".py":
		return tr.settings.PythonExecutable != ""
	}
	panic("invalid extension")
}
