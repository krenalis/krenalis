//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// The package debugid provides useful functions for working with Sentry Debug
// IDs:
//
// https://docs.sentry.io/platforms/javascript/sourcemaps/troubleshooting_js/debug-ids/
//
// In particular, the functions here provide tools for calculating Debug IDs
// based on file content and for injesting those Debug IDs into the files, in a
// Sentry-compatible format.
package debugid

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
)

// CalculateFileDebugID calculates the Sentry's Debug ID, deterministically,
// from the contents of the file at filename.
//
// "Deterministically" means that if the file contents are the same, the
// resulting Debug ID will also be the same.
//
// This is like [CalculateFileDebugIDFromContent], but this function accepts the
// filename instead of the file content.
func CalculateFileDebugID(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return calculateFileDebugIDFromContent(data), nil
}

// CalculateFileDebugIDFromContent calculates the Sentry's Debug ID,
// deterministically, for a file with the provided content.
//
// "Deterministically" means that if the file contents are the same, the
// resulting Debug ID will also be the same.
//
// This is like [CalculateFileDebugID], but this function accepts the file
// content instead of the filename.
func CalculateFileDebugIDFromContent(content []byte) string {
	return calculateFileDebugIDFromContent(content)
}

// InjectDebugIDIntoFile injects the header and footer referring to the Debug ID
// into the file, in the format required by Sentry.
//
// See
// https://docs.sentry.io/platforms/javascript/sourcemaps/uploading/cli/#3-inject-debug-ids-into-artifacts.
//
// This is like [InjectDebugIDIntoFileContent], but this function injects the
// code directly into the file with the given filename.
func InjectDebugIDIntoFile(filename, debugID string) error {

	// Read the file content and stat.
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	fileContent, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Inject the Debug ID.
	fileContent = injectDebugIDIntoFileContent(fileContent, debugID)

	// Write the file.
	err = os.WriteFile(filename, fileContent, fi.Mode().Perm())
	if err != nil {
		return err
	}

	return nil
}

// InjectDebugIDIntoFileContent injects the header and footer referring to the
// Debug ID into the file, in the format required by Sentry.
//
// See
// https://docs.sentry.io/platforms/javascript/sourcemaps/uploading/cli/#3-inject-debug-ids-into-artifacts.
//
// This is like [InjectDebugIDIntoFile], but this function injects the code on
// the file content and returns a new slice, leaving unaltered the provided
// fileContent, instead of altering a file.
func InjectDebugIDIntoFileContent(fileContent []byte, debugID string) []byte {
	return injectDebugIDIntoFileContent(fileContent, debugID)
}

// InjectDebugIDIntoSourceMap injects the "debug_id", in the format required by
// Sentry, into the source map file with the given filename.
//
// It also fixes the "mappings" key of the map, to handle the "shift" introduced
// by the changes to the original JS file.
//
// See
// https://docs.sentry.io/platforms/javascript/sourcemaps/uploading/cli/#3-inject-debug-ids-into-artifacts.
func InjectDebugIDIntoSourceMap(sourceMapFilename string, debugID string) error {

	// Read the file content and stat.
	fr, err := os.Open(sourceMapFilename)
	if err != nil {
		return err
	}
	defer fr.Close()

	fi, err := fr.Stat()
	if err != nil {
		return err
	}

	var sourceMap map[string]any
	err = json.NewDecoder(fr).Decode(&sourceMap)
	if err != nil {
		return err
	}
	err = fr.Close()
	if err != nil {
		return err
	}

	sourceMap["debug_id"] = debugID
	sourceMap["mappings"] = ";;" + sourceMap["mappings"].(string)

	fw, err := os.OpenFile(sourceMapFilename, os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fw)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	err = enc.Encode(sourceMap)
	if err != nil {
		return err
	}

	return nil
}

func calculateFileDebugIDFromContent(content []byte) string {
	// Calculate the MD5 checksum of the file content.
	checksum := md5.Sum(content)
	// Create an UUID, representing the Debug ID, corresponding to the
	// 128 bits of the MD5 checksum.
	debugID, _ := uuid.FromBytes(checksum[:])
	return debugID.String()
}

const (
	debugIDHeader = "\n" + `!function(){try{var e="undefined"!=typeof window?window:"undefined"!=typeof global?global:"undefined"!=typeof globalThis?globalThis:"undefined"!=typeof self?self:{},n=(new e.Error).stack;n&&(e._sentryDebugIds=e._sentryDebugIds||{},e._sentryDebugIds[n]="{{debugId}}")}catch(e){}}();`
	debugIDFooter = `//# debugId={{debugId}}` + "\n"
)

func injectDebugIDIntoFileContent(fileContent []byte, debugID string) []byte {

	header := strings.ReplaceAll(debugIDHeader, "{{debugId}}", debugID)
	footer := strings.ReplaceAll(debugIDFooter, "{{debugId}}", debugID)

	var b bytes.Buffer
	b.WriteString(header)
	b.WriteByte('\n')
	b.Write(fileContent)
	b.WriteByte('\n')
	b.WriteString(footer)

	return b.Bytes()
}
