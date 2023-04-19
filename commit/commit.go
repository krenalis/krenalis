//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {

	start := time.Now()

	// Find modules in this repository.
	var modules []string
	err := filepath.Walk(".", func(path string, _ fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		return nil
	})
	sort.Strings(modules)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the command has been executed correctly basing on modules which
	// certainly should be found.
	for _, mod := range []string{".", "chichi-cli"} {
		found := false
		for _, mod2 := range modules {
			if mod2 == mod {
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("module %q not found, maybe you ran this script incorrectly"+
				"or this script is out-of-date", mod)
		}
	}

	// Get the cwd.
	repo, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Re-create the 'go.sum' files in the repository.
	for _, module := range modules {
		removeGoSum(repo, module)
		cmd("go", []string{"mod", "tidy"}, repo, module)
	}

	// Call command(s) on every module.
	for _, module := range modules {
		cmd("go", []string{"fmt", "./..."}, repo, module)
		cmd("go", []string{"vet", "./..."}, repo, module)
		cmd("go", []string{"test", "./..."}, repo, module)
	}

	// Call command(s) on the workspace.
	cmd("go", []string{"work", "sync"}, repo, ".")

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

func cmd(name string, arg []string, repo, moduleDir string) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = filepath.Join(repo, moduleDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logCmd(moduleDir, strings.Join(append([]string{name}, arg...), " "))
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func removeGoSum(repo, module string) {
	logCmd(module, "rm go.sum")
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}
}

func logCmd(dir, cmd string) {
	fmt.Printf("%-30s%s\n", dir, cmd)
}
