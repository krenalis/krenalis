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
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/core/transformers/embed"
	"github.com/meergo/meergo/types"
)

type function struct {
	settings Settings
}

type Settings struct {
	NodeExecutable   string // eg. "/usr/bin/node".
	PythonExecutable string // eg. "/usr/bin/python".
	FunctionsDir     string
}

func New(settings Settings) transformers.Provider {
	return &function{settings: settings}
}

// Call calls the function with the given identifier and version for each record
// updating its Properties field with the result of each invocation. Record
// properties are supposed to conform to inSchema. After the transformation,
// Record properties conform to outSchema unless a transformation error
// occurred, and in that case, the error is stored in the Record's Err field.
//
// It returns the ErrFunctionNotExist error if the function does not exist, and
// a FunctionExecutionError if the execution fails.
func (fn *function) Call(ctx context.Context, id, version string, inSchema, outSchema types.Type, preserveJSON bool, records []transformers.Record) error {

	name, language, err := parseID(id)
	if err != nil {
		return err
	}

	var executable string
	switch language {
	case state.JavaScript:
		executable = fn.settings.NodeExecutable
	case state.Python:
		executable = fn.settings.PythonExecutable
	default:
		return errors.New("language is not supported")
	}

	if v, _ := strconv.Atoi(version); v <= 0 || version[0] == '+' {
		return fmt.Errorf("invalid version %q", version)
	}
	filename := fn.filename(name, version, language)
	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return transformers.ErrFunctionNotExist
		}
		return err
	}
	payload := make([]byte, 0, 1024)
	payload, err = transformers.Marshal(payload, inSchema, records, language, preserveJSON)
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, executable, filename, string(payload))
	cmd.Env = []string{}
	cmd.Dir = fn.settings.FunctionsDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return transformers.Unmarshal(&stdout, records, outSchema, language, preserveJSON)
}

// Close closes the function.
func (fn *function) Close(ctx context.Context) error {
	return nil
}

// Create creates a new function with the given name, language, and source and
// returns its identifier and version.
func (fn *function) Create(ctx context.Context, name string, language state.Language, source string) (string, string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", "", errors.New("function name is not valid")
	}
	if !fn.SupportLanguage(language) {
		return "", "", errors.New("language is not supported")
	}
	// TODO(Gianluca): on Windows, escape reserved filenames.
	// See https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file?redirectedfrom=MSDN.
	id, err := fn.create(name, "1", language, source)
	if err != nil {
		return "", "", err
	}
	return id, "1", nil
}

// create creates a function with the given name, version, language, and source.
func (fn *function) create(name, version string, language state.Language, source string) (string, error) {
	var ext string
	var fullSource string
	switch language {
	case state.JavaScript:
		ext = "js"
		escapedSource := escapeJavaScriptSourceCode(source)
		fullSource = `
try {
	Function(` + "`" + escapedSource + "`" + `);
} catch (error) {
	process.stdout.write(JSON.stringify({ error: error.toString() }));
	return;
}
const transform = Function('event', ` + "`" + escapedSource + "; return transform(event)`" + `);
const records = [];
const event = Function("return " + process.argv[2])();
` + embed.JavaScriptNormalizeFunc + `
for ( let i = 0; i < event.length; i++ ) {
	try {
		let value = transform(event[i]);
		normalize(value);
		records[i] = { value: value };
	} catch (error) {
		if (error instanceof Error) {
			error = error.toString();
		} else {
			error = "throw error of type " + (typeof error) + ": " + JSON.stringify(error);
		}
		records[i] = { error: error };
	}
}
process.stdout.write(JSON.stringify({ records: records }));`
	case state.Python:
		ext = "py"
		fullSource = embed.PythonNormalizeFunc + "\n\n"
		fullSource += "_SOURCE = '''" + escapePythonSourceCode(source) + "'''\n\n"
		fullSource += `
def main():
	import json
	import sys
	from uuid import UUID
	from decimal import Decimal
	from datetime import datetime, date, time

	try:
		exec(_SOURCE, globals())
	except SyntaxError as ex:
		error = f"SyntaxError: {ex.msg} (line {ex.lineno})"
		print(json.dumps({"error": error}, separators=(",", ":"), default=str))
		return
	except Exception as ex:
		name = type(ex).__name__
		error = f"{name}: {ex}"
		print(json.dumps({"error": error}, separators=(",", ":"), default=str))
		return

	records = []
	for event in eval(sys.argv[1]):
		try:
			value = transform(event)
			_Norm.normalize(value)
		except Exception as ex:
			name = type(ex).__name__
			records.append({"error": f"{name}: {ex}"})
		else:
			records.append({"value": value})
	print(json.dumps({"records": records}, separators=(",", ":"), default=str))

if __name__ == "__main__":
	main()
`
	}
	filename := fn.filename(name, version, language)
	var success bool
	defer func() {
		if !success {
			_ = os.Remove(filename)
		}
	}()
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("function name %q already exist", name)
		}
		dir := filepath.Dir(filename)
		st, err2 := os.Stat(dir)
		if err2 != nil {
			if errors.Is(err2, os.ErrNotExist) {
				return "", fmt.Errorf("directory %q for storing local transformation functions does not exist", dir)
			}
		} else {
			if !st.IsDir() {
				return "", fmt.Errorf("path %q for storing local transformation functions is not a directory", dir)
			}
		}
		return "", fmt.Errorf("cannot create local transformation function: %v", err)
	}
	_, err = f.WriteString(fullSource)
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	success = true
	id := fmt.Sprintf("%s.%s", name, ext)
	return id, nil
}

// Delete deletes the function with the given identifier.
// If a function with the given identifier does not exist, it does nothing.
func (fn *function) Delete(ctx context.Context, id string) error {
	name, language, err := parseID(id)
	if err != nil {
		return err
	}
	var ext string
	switch language {
	case state.JavaScript:
		ext = "js"
	case state.Python:
		ext = "py"
	}
	dir := fn.settings.FunctionsDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory %q for storing local transformation functions does not exist", dir)
		}
		return fmt.Errorf("cannot read files in directory %q storing local transformation functions", dir)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version, ok := versionFromFilename(entry.Name(), name, ext)
		if ok {
			filename := fn.filename(name, strconv.Itoa(version), language)
			err := os.Remove(filename)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("cannot remove file %q of local transformation function: %v", filename, err)
			}
		}
	}
	return nil
}

// SupportLanguage reports whether language is supported as a language.
// It panics if language is not valid.
func (fn *function) SupportLanguage(language state.Language) bool {
	switch language {
	case state.JavaScript:
		return fn.settings.NodeExecutable != ""
	case state.Python:
		return fn.settings.PythonExecutable != ""
	}
	panic("invalid language")
}

// Update updates the source of the function with the given identifier and
// returns a new version, which has a length in the range [1, 128].
// If the function does not exist, it returns the ErrFunctionNotExist error.
func (fn *function) Update(ctx context.Context, id, source string) (string, error) {
	name, language, err := parseID(id)
	if err != nil {
		return "", err
	}
	var ext string
	switch language {
	case state.JavaScript:
		ext = "js"
	case state.Python:
		ext = "py"
	}
	dir := fn.settings.FunctionsDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("directory %q for storing local transformation functions does not exist", dir)
		}
		return "", fmt.Errorf("cannot read files in directory %q storing local transformation functions", dir)
	}
	// Filenames for functions should be like: "<name>_v<version>.<ext>"
	var maxVersion int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		v, ok := versionFromFilename(entry.Name(), name, ext)
		if ok && v > maxVersion {
			maxVersion = v
		}
	}
	if maxVersion == 0 {
		return "", transformers.ErrFunctionNotExist
	}
	if maxVersion == math.MaxInt64 {
		return "", errors.New("too many versions")
	}
	version := strconv.Itoa(maxVersion + 1)
	_, err = fn.create(name, version, language, source)
	if err != nil {
		return "", err
	}
	return version, nil
}

// versionFromFilename returns the version from the filename relative to a
// function with the given name.
//
// For example, if a function is named "meergo-action12345" and the filename is
// "meergo-action12345.v10.py", then 10 and true are returned.
//
// The boolean value reports whether the filename (and thus the returned
// version) is valid for the given name or not.
func versionFromFilename(filename, name, ext string) (int, bool) {
	s, ok := strings.CutPrefix(filename, name+".v")
	if !ok {
		return 0, false
	}
	s, ok = strings.CutSuffix(s, "."+ext)
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

// pythonEscaper is used by escapePythonSourceCode.
//
// Keep this in sync with the code within the Lambda transformer.
var pythonEscaper = strings.NewReplacer(`\`, `\\`, `'''`, `''\'`)

// escapePythonSourceCode escapes the given Python source code so it can be
// safely be put into a triple-quoted Python string literal (where the quote
// character is the single quote, not double) for later evaluation.
//
// Keep this in sync with the code within the Lambda transformer.
func escapePythonSourceCode(src string) string {
	return pythonEscaper.Replace(src)
}

// javaScriptEscaper is used by escapeJavaScriptSourceCode.
//
// Keep this in sync with the code within the Lambda transformer.
var javaScriptEscaper = strings.NewReplacer(`\`, `\\`, "`", "\\`", `$`, `\$`)

// escapeJavaScriptSourceCode escapes the given JavaScript source code so it can
// be safely be put into a single quoted JavaScript string literal for later
// evaluation.
//
// Keep this in sync with the code within the Lambda transformer.
func escapeJavaScriptSourceCode(src string) string {
	return javaScriptEscaper.Replace(src)
}

// filename returns the absolute filename corresponding to the provided function's name, version, and language.
func (fn *function) filename(name, version string, language state.Language) string {
	var ext string
	switch language {
	case state.JavaScript:
		ext = "js"
	case state.Python:
		ext = "py"
	}
	return filepath.Join(fn.settings.FunctionsDir, fmt.Sprintf("%s.v%s.%s", name, version, ext))
}

// parseID parses the provided function identifier and returns the function name
// and its associated language.
func parseID(id string) (name string, language state.Language, err error) {
	var ext string
	name, ext, _ = strings.Cut(id, ".")
	switch ext {
	case "js":
		language = state.JavaScript
	case "py":
		language = state.Python
	default:
		return "", 0, fmt.Errorf("transformers/local: invalid function identifier %q", id)
	}
	return
}
