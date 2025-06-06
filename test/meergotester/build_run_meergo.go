//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildMeergo builds Meergo.
func buildMeergo(t *testing.T, repo, meergoDir string) {

	// Create a temporary directory.
	tmpdir, err := os.MkdirTemp("", "meergo-build-for-tests-*")
	if err != nil {
		t.Fatalf("%s", err)
	}

	// Copy the 'main.go' file, which is the entry point for Meergo.
	err = copyFile(
		filepath.Join(repo, "cmd", "meergo", "main.go"),
		filepath.Join(tmpdir, "main.go"),
	)
	if err != nil {
		t.Fatalf("%s", err)
	}

	// Initialize the Go module.
	execCmd(t, tmpdir, "go", "mod", "init", "meergo")

	// Edit the go.mod so that the local Meergo sources are used, both for Go
	// and for the assets.
	execCmd(t, tmpdir, "go", "mod", "edit", "-replace", "github.com/meergo/meergo="+repo)

	// Edit the go.mod as a workaround for
	// https://github.com/meergo/meergo/issues/1558.
	execCmd(t, tmpdir, "go", "mod", "edit", "-replace", "github.com/ClickHouse/clickhouse-go/v2=github.com/open2b/clickhouse-go/v2@v2.35.0-fix2")

	// Copy the file with the connectors and warehouse imports, replacing the
	// package name "meergotester" with "main".
	testImports, err := os.ReadFile(filepath.Join(repo, "test", "meergotester", "test_imports.go"))
	if err != nil {
		t.Fatalf("%s", err)
	}
	testImports = bytes.Replace(testImports, []byte(`package meergotester`), []byte(`package main`), 1)
	err = os.WriteFile(filepath.Join(tmpdir, "test_imports.go"), testImports, 0644)
	if err != nil {
		t.Fatalf("%s", err)
	}

	// Run 'go mod tidy'.
	execCmd(t, tmpdir, "go", "mod", "tidy")

	// Generate the assets.
	execCmd(t, tmpdir, "go", "generate")

	// Build Meergo, putting the output into the meergoDir, where it will be
	// executed by the tests.
	execCmd(t, tmpdir, "go", "build", "-o", filepath.Join(meergoDir, meergoExecFilename()))

}

func execCmd(t *testing.T, dir string, command ...string) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("executing command '%s' within %s", strings.Join(command, " "), dir)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("command execution failed: %s", err)
	}
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

// generateAssets generates the assets necessary for the compilation and
// execution of Meergo in production mode, which is the mode used by the tests.
func generateAssets(ctx context.Context, repo string) error {
	cmd := exec.CommandContext(ctx, "go", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Join(repo, "cmd", "meergo")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command 'go generate' executed in directory '%s' failed: %s", cmd.Dir, err)
	}
	return nil
}

func launchMeergo(ctx context.Context, env []string) error {
	repo, err := filepath.Abs("../")
	if err != nil {
		return err
	}
	meergoDir := filepath.Join(repo, "test", "meergo-executable-for-tests")
	cmd := exec.CommandContext(ctx, "./"+meergoExecFilename())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = meergoDir
	cmd.Env = env
	return cmd.Run()
}

func meergoExecFilename() string {
	if runtime.GOOS == "windows" {
		return "meergo.exe"
	}
	return "meergo"
}
