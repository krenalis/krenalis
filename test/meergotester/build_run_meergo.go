//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// buildMeergo builds Meergo.
//
// Meergo is compiled in production mode, so the asset files must already be
// generated in the directory where Meergo is compiled.
func buildMeergo(ctx context.Context, repo, meergoDir string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "osusergo,netgo", "-o", filepath.Join(meergoDir, meergoExecFilename()), "./cmd/meergo")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repo
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cannot build Meergo: %s", err)
	}
	return nil
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
