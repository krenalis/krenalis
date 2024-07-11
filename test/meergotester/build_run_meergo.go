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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/meergo/meergo/cmd"
)

// buildMeergo builds Meergo and copies files needed for the execution.
func buildMeergo(repo, meergoDir string, ctx context.Context) error {

	// Build Meergo.
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "osusergo,netgo", "-o", filepath.Join(meergoDir, meergoExecFilename()), "./cmd/meergo")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repo
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cannot build Meergo: %s", err)
	}

	// Copy the certificates.
	for _, cert := range []string{"cert.pem", "key.pem"} {
		src := cert // the certificate under the "test" directory.
		dst := filepath.Join(meergoDir, cert)
		err = copyFile(dst, src)
		if err != nil {
			abs, err2 := filepath.Abs(src)
			if err2 != nil {
				return err2
			}
			return fmt.Errorf("cannot read HTTPS certificate %s: %s", abs, err)
		}
	}

	return nil
}

func writeConfigYAMLFile(meergoDir string, setts *cmd.Settings) error {
	err := validDatabaseNameForTests(setts.PostgreSQL.Database)
	if err != nil {
		return err
	}
	conf, err := yaml.Marshal(setts)
	if err != nil {
		return err
	}
	configYamlPath := filepath.Join(meergoDir, "config.yaml")
	err = os.WriteFile(configYamlPath, conf, 0644)
	return err
}

func launchMeergo(ctx context.Context) error {
	repo, err := filepath.Abs("../")
	if err != nil {
		return err
	}
	meergoDir := filepath.Join(repo, "test", "meergo-executable-for-tests")
	cmd := exec.CommandContext(ctx, "./"+meergoExecFilename())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = meergoDir
	return cmd.Run()
}

func copyFile(dst, src string) error {
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

func meergoExecFilename() string {
	if runtime.GOOS == "windows" {
		return "meergo.exe"
	}
	return "meergo"
}
