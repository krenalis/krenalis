//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"chichi/server"
)

func buildChichi(ctx context.Context, setts *server.Settings) error {

	repo, err := filepath.Abs("../")
	if err != nil {
		return err
	}
	_, err = os.Stat(filepath.Join(repo, "go.work"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("file 'go.work' not found, cannot determine root directory where to build Chichi")
		}
		return err
	}

	chichiDir := filepath.Join(repo, "test", "chichi-executable-for-tests")
	err = os.Mkdir(chichiDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	// Build Chichi.
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "osusergo,netgo", "-trimpath", "-o", filepath.Join(chichiDir, chichiExecFilename()))
	cmd.Dir = repo
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("cannot build Chichi: %s", err)
	}

	// Copy the certificates.
	for _, cert := range []string{"cert.pem", "key.pem"} {
		src := cert // the certificate under the "test" directory.
		dst := filepath.Join(chichiDir, cert)
		err = copyFile(dst, src)
		if err != nil {
			abs, err2 := filepath.Abs(src)
			if err2 != nil {
				return err2
			}
			return fmt.Errorf("cannot read HTTPS certificate %s: %s", abs, err)
		}
	}

	// Write the configuration.
	conf := &bytes.Buffer{}
	conf.WriteString("[Main]\n")
	conf.WriteString("Host=" + setts.Main.Host + "\n")
	conf.WriteString("[PostgreSQL]\n")
	conf.WriteString("Host=" + setts.PostgreSQL.Host + "\n")
	conf.WriteString("Port=" + strconv.Itoa(setts.PostgreSQL.Port) + "\n")
	conf.WriteString("Username=" + setts.PostgreSQL.Username + "\n")
	conf.WriteString("Password=" + setts.PostgreSQL.Password + "\n")
	conf.WriteString("Database=" + setts.PostgreSQL.Database + "\n")
	conf.WriteString("Schema=" + setts.PostgreSQL.Schema + "\n")
	appIniPath := filepath.Join(chichiDir, "app.ini")
	err = os.WriteFile(appIniPath, conf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func launchChichi(ctx context.Context) error {
	repo, err := filepath.Abs("../")
	if err != nil {
		return err
	}
	chichiDir := filepath.Join(repo, "test", "chichi-executable-for-tests")
	cmd := exec.CommandContext(ctx, "./"+chichiExecFilename())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = chichiDir
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

func chichiExecFilename() string {
	if runtime.GOOS == "windows" {
		return "chichi.exe"
	}
	return "chichi"
}
