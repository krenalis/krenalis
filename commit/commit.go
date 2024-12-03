//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// expectedDenoVersion is the expected Deno version.
//
// Keep in sync with the version within ".github/workflows/main.yml".
const expectedDenoVersion = "2.1.0"

func main() {

	var short bool
	var verbose bool
	var testPackages bool
	flag.BoolVar(&short, "short", false, "pass the '-short' flag to 'go test'")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&testPackages, "pkg", false, "run tests on every single package"+
		" instead of every module (used in conjunction with option '-v', may"+
		" help spotting problems in tests)")
	flag.Parse()

	start := time.Now()

	// Check if the locally installed Deno version is correct.
	checkDenoVersion()

	// Find modules and packages in this repository.
	var modules []string
	var packages []string
	err := filepath.Walk(".", func(path string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() && fi.Name() == "vendor" {
			return filepath.SkipDir
		}
		// Module found.
		if filepath.Base(path) == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		// Package found.
		if filepath.Ext(path) == ".go" {
			dir := filepath.Dir(path)
			if !slices.Contains(packages, dir) {
				packages = append(packages, dir)
			}
		}
		return nil
	})
	if err != nil {
		fatal("cannot read modules and packages in the repository: %s", err)
	}
	slices.Sort(modules)
	slices.Sort(packages)

	// Check if the command has been executed correctly basing on modules which
	// certainly should be found.
	for _, mod := range []string{"."} {
		if !slices.Contains(modules, mod) {
			fatal("module %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", mod)
		}
	}

	// Check if the command has been executed correctly basing on packages which
	// certainly should be found.
	for _, pkg := range []string{".", "core", "connectors", "warehouses"} {
		if !slices.Contains(packages, pkg) {
			fatal("package %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", pkg)
		}
	}

	// Get the cwd.
	repo, err := os.Getwd()
	if err != nil {
		fatal("cannot read the cwd: %s", err)
	}

	fmt.Println("Tidying modules")
	for _, module := range modules {
		removeGoSum(repo, module, verbose)
		NewCmd("go", "mod", "tidy").InDir(repo, module).Run()
	}

	fmt.Println("Formatting modules")
	for _, module := range modules {
		NewCmd("go", "fmt", "./...").InDir(repo, module).Run()
	}

	fmt.Println("Running 'go vet' on every module")
	for _, module := range modules {
		NewCmd("go", "vet", "./...").InDir(repo, module).Run()
	}

	// Test single packages or modules.
	fmt.Println("Running Go tests")
	args := []string{"test", "-count", "1"}
	if short {
		args = append(args, "-short")
	}
	if verbose {
		args = append(args, "-v")
	}
	if testPackages {
		for _, pkg := range packages {
			NewCmd("go", args...).InDir(repo, pkg).Run()
		}
	} else {
		args = append(args, "./...")
		for _, module := range modules {
			NewCmd("go", args...).InDir(repo, module).Run()
		}
	}

	// Update the vendor.
	NewCmd("go", "mod", "vendor").InDir(repo).Run()

	// Run 'npm install' in the 'assets' directory.
	NewCmd("npm", "install").InDir(repo, "assets").Run()

	// Format the files in the 'assets' directory.
	NewCmd("npm", "run", "prettier").InDir(repo, "assets").Run()

	// Minify the JavaScript snippet in the 'assets' directory.
	NewCmd("npm", "run", "minify-snippet").InDir(repo, "assets").Run()

	// Typecheck the Typescript code in the 'assets' directory.
	NewCmd("npm", "run", "typecheck").InDir(repo, "assets").Run()

	// Make the vendor of assets' 'node_modules' directory.
	NewCmd("npm", "run", "make-vendor").InDir(repo, "assets").Run()

	// Format, test and build the files in the 'javascript-sdk' directory.
	NewCmd("npm", "install").InDir(repo, "javascript-sdk").Run()
	NewCmd("deno", "fmt").InDir(repo, "javascript-sdk").Run()
	NewCmd("deno", "task", "build").InDir(repo, "javascript-sdk").Run()

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

func checkDenoVersion() {
	fmt.Println("Checking the Deno version")
	cmd := exec.Command("deno", "--version")
	var stdout bytes.Buffer
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		fatal("cannot execute the command 'deno --version': %s", err)
	}
	firstLine := strings.Split(stdout.String(), "\n")[0]
	parts := strings.Split(firstLine, " ")
	if len(parts) < 2 {
		fatal("unexpected output from 'deno --version': %q", stdout.String())
	}
	version := parts[1]
	if version != expectedDenoVersion {
		fatal(fmt.Sprintf("it is not possible to run the tests because they require Deno %s, but the installed version is Deno %s.\n"+
			"To proceed with the tests, please update the Deno version:\n"+
			"\n\tdeno upgrade --version %s\n\n"+
			"If your intention is to update the tests to use Deno %s instead, please modify the specified version in the 'commit/commit.go' script.\n",
			expectedDenoVersion, version, expectedDenoVersion, version))
	}
	fmt.Printf("Locally installed Deno version is correct: %s\n", version)
}

func logCmd(dir, cmd string) {
	const (
		Reset = "\033[0m"
		Bold  = "\033[1m"
	)
	fmt.Printf("%s%s$ %s%s\n", Bold, dir, cmd, Reset)
}

type Cmd struct {
	Name          string
	Args          []string
	Echo          bool
	Dir           string
	AdditionalEnv []string
}

func NewCmd(name string, args ...string) *Cmd {
	return &Cmd{Name: name, Args: args, Echo: true}
}

func (cmd *Cmd) Silent() *Cmd {
	cmd.Echo = false
	return cmd
}

func (cmd *Cmd) WithEnv(name, value string) *Cmd {
	cmd.AdditionalEnv = append(cmd.AdditionalEnv, name+"="+value)
	return cmd
}

func (cmd *Cmd) InDir(elem ...string) *Cmd {
	cmd.Dir = filepath.Join(elem...)
	return cmd
}

func (cmd *Cmd) Run() {
	goCmd := exec.Command(cmd.Name, cmd.Args...)
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	if cmd.Echo {
		logCmd(cmd.Dir, strings.Join(append([]string{cmd.Name}, cmd.Args...), " "))
	}
	goCmd.Env = append(os.Environ(), cmd.AdditionalEnv...)
	goCmd.Dir = cmd.Dir
	err := goCmd.Run()
	if err != nil {
		fatal("command %q failed (%s)", cmd.Name, err)
	}
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Fatal error: "+msg+"\n", args...)
	os.Exit(1)
}

func removeGoSum(repo, module string, verbose bool) {
	if verbose {
		logCmd(module, "rm go.sum")
	}
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatal("cannot remove 'go.sum': %s", err)
	}
}
