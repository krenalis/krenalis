// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"bufio"
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

func main() {

	cliOptions := parseCli()

	if cliOptions.justTestAdmin {
		runGoTestAdmin()
		os.Exit(0)
	}

	start := time.Now()

	// Find modules and packages in the current working directory, then ensure
	// that the script has been launched with the correct working directory.
	modules, packages := findModulesPackages(".")
	for _, mod := range []string{"."} { // just some random modules in the repository (currently we have just one).
		if !slices.Contains(modules, mod) {
			fatal("module %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", mod)
		}
	}
	for _, pkg := range []string{"core", "connectors", "warehouses"} { // just some random top-level packages in the repository.
		if !slices.Contains(packages, pkg) {
			fatal("package %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", pkg)
		}
	}

	// Get the current working directory.
	repo, err := os.Getwd()
	if err != nil {
		fatal("cannot read the current working directory: %s", err)
	}

	// Tidy modules.
	fmt.Println("Tidy modules")
	for _, module := range modules {
		removeGoSum(repo, module)
		NewCmd("go", "mod", "tidy").InDir(repo, module).Run()
	}

	// Go-fix modules.
	fmt.Println("Go-fixing modules")
	for _, module := range modules {
		// Call the command 3 times, because some fixes are applied in
		// subsequent calls. 3 is a reasonable number to most likely perform all
		// the necessary fixes.
		for range 3 {
			NewCmd("go", "fix", "./...").InDir(repo, module).Run()
		}
	}

	// Format modules.
	fmt.Println("Format modules")
	for _, module := range modules {
		NewCmd("go", "fmt", "./...").InDir(repo, module).Run()
	}

	// Running 'go vet' on every module.
	fmt.Println("Running 'go vet' on every module")
	for _, module := range modules {
		NewCmd("go", "vet", "./...").InDir(repo, module).Run()
	}

	// Update the Go vendor.
	NewCmd("go", "mod", "vendor").InDir(repo).Run()

	// Run checks and do operations on the Admin.
	fmt.Println("Run checks and do operations on the Admin")
	NewCmd("npm", "ci").InDir(repo, "admin").Run()

	// TODO(Gianluca): this is a workaround for this npm bug:
	// https://github.com/npm/cli/issues/8690#issuecomment-3463552492.
	//
	// See https://github.com/krenalis/krenalis/issues/2164.
	err = removePeerLines("admin/package-lock.json")
	if err != nil {
		fatal("cannot remove peer lines from 'admin/package-lock.json': %s", err)
	}

	NewCmd("npm", "run", "prettier").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "minify-snippet").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "typecheck").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "makevendor").InDir(repo, "admin").Run()

	// Validate the Docker Compose files.
	fmt.Println("Validate Docker Compose files")
	NewCmd("docker", "compose", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run()                           // compose.yaml with overriding (default).
	NewCmd("docker", "compose", "-f", "compose.yaml", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run()     // compose.yaml without overriding.
	NewCmd("docker", "compose", "-f", "compose.dev.yaml", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run() // compose.dev.yaml.

	// Run Go tests.
	if runGoTests := !cliOptions.noGoTest; runGoTests {
		fmt.Println("Run Go tests")
		args := []string{
			"test",
			"-count",
			"1",
			"-failfast",
			"-v",
			// It is important to specify a timeout, because otherwise 'go test'
			// has a default timeout of 10 minutes (see 'go help testflag'),
			// which may not be sufficient to run all the tests inside "/test".
			"--timeout=2h",
		}
		if cliOptions.short {
			args = append(args, "-short")
		}
		if cliOptions.noSnowflakeTests {
			// Skip all tests and subtests that have "snowflake" in the name
			// (case insensitive).
			args = append(args, "-skip", "snowflake")
		}
		for _, pkg := range packages {
			if cliOptions.noConnectorTests && strings.HasPrefix(pkg, "connectors"+string(os.PathSeparator)) {
				continue // skip this package.
			}
			NewCmd("go", args...).InDir(repo, pkg).Run()
		}
	}

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

type cliOptions struct {
	justTestAdmin    bool
	noConnectorTests bool
	noSnowflakeTests bool
	noGoTest         bool
	short            bool
}

// TODO(Gianluca): this function is a workaround for this npm bug:
// https://github.com/npm/cli/issues/8690#issuecomment-3463552492.
//
// See https://github.com/krenalis/krenalis/issues/2164.
func removePeerLines(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"peer": true`) {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	for i := 0; i < len(lines)-1; i++ {
		trimmed := strings.TrimSpace(lines[i])
		nextTrimmed := strings.TrimSpace(lines[i+1])
		if strings.HasSuffix(trimmed, ",") &&
			(strings.HasPrefix(nextTrimmed, "}") || strings.HasPrefix(nextTrimmed, "]")) {
			commaIdx := strings.LastIndex(lines[i], ",")
			lines[i] = lines[i][:commaIdx]
		}
	}

	output := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filepath, []byte(output), 0644)
}

func parseCli() cliOptions {

	var justTestAdmin bool
	var noGoTest bool
	var printHelp bool
	var short bool
	var noConnectorTests bool
	var noSnowflakeTests bool

	const reducedTestSetWarning = "WARNING: this option reduces the set of tests performed," +
		" so some parts of the software and/or changes made may not be validated " +
		"when running the script with this option"

	flag.BoolVar(&justTestAdmin, "just-test-admin", false, "just run the Go tests on the Admin console. "+
		reducedTestSetWarning)
	flag.BoolVar(&noConnectorTests, "no-connector-tests", false, "do not run 'go test' within the 'connectors' directory. "+reducedTestSetWarning)
	flag.BoolVar(&noSnowflakeTests, "no-snowflake-tests", false, "skip Go tests that contain 'snowflake' in their name (case insensitive). "+reducedTestSetWarning)
	flag.BoolVar(&noGoTest, "no-go-test", false, "do not run 'go test' at all."+
		" Useful when you just want to run vendor generation commands, various asset related commands, etc... "+
		reducedTestSetWarning)
	flag.BoolVar(&short, "short", false, "pass the '-short' flag to 'go test', reducing the tests set. "+reducedTestSetWarning)
	flag.BoolVar(&printHelp, "help", false, "print help message and exit")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of the 'commit' command:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) > 0 {
		fmt.Fprintf(flag.CommandLine.Output(), "Unexpected command line parameters: %s\n", strings.Join(flag.Args(), " "))
		flag.Usage()
		os.Exit(1)
	}

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Mutually exclusive flags.
	mutualExclusive := func(flag1, flag2 bool, name1, name2 string) {
		if flag1 && flag2 {
			fmt.Fprintf(flag.CommandLine.Output(), "CLI error: flag '%s' cannot be used in conjunction with flag '%s'\n", name1, name2)
			flag.Usage()
			os.Exit(1)
		}
	}

	// Flags incompatible with '--just-test-admin'.
	mutualExclusive(justTestAdmin, noConnectorTests, "-just-test-admin", "-no-connector-tests")
	mutualExclusive(justTestAdmin, noGoTest, "-just-test-admin", "-no-go-test")
	mutualExclusive(justTestAdmin, short, "-just-test-admin", "-short")
	// Flags incompatible with '--no-go-test'.
	mutualExclusive(noGoTest, short, "-no-go-test", "-short")
	mutualExclusive(noGoTest, noConnectorTests, "-no-go-test", "-no-connector-tests")
	mutualExclusive(noGoTest, noSnowflakeTests, "-no-go-test", "-no-snowflake-tests")

	return cliOptions{
		justTestAdmin:    justTestAdmin,
		noConnectorTests: noConnectorTests,
		noSnowflakeTests: noSnowflakeTests,
		noGoTest:         noGoTest,
		short:            short,
	}
}

// findModulesPackages finds the Go modules and packages within the given dir.
// Both the modules and packages are sorted in alphabetical order.
// This function skips directories named "vendor".
func findModulesPackages(dir string) (modules, packages []string) {
	err := filepath.Walk(dir, func(path string, fi fs.FileInfo, err error) error {
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
	return modules, packages
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
	Dir           string
	AdditionalEnv []string
}

func NewCmd(name string, args ...string) *Cmd {
	return &Cmd{Name: name, Args: args}
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
	goCmd.Env = append(os.Environ(), cmd.AdditionalEnv...)
	goCmd.Dir = cmd.Dir
	logCmd(cmd.Dir, strings.Join(append([]string{cmd.Name}, cmd.Args...), " "))
	err := goCmd.Run()
	if err != nil {
		// Stdout and Stderr have already been printed.
		fatal("command %q failed (%s)", cmd.Name, err)
	}
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Fatal error: "+msg+"\n", args...)
	os.Exit(1)
}

func removeGoSum(repo, module string) {
	logCmd(filepath.Join(repo, module), "rm go.sum")
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatal("cannot remove 'go.sum': %s", err)
	}
}

// runGoTestAdmin runs Admin tests via go test.
func runGoTestAdmin() {
	start := time.Now()
	args := []string{"test", "-run", "^TestAdmin$", "github.com/krenalis/krenalis/test", "-count", "1", "-v"}
	NewCmd("go", args...).Run()
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		fatal("admin test took too little time (< 2 seconds). There is probably a problem" +
			" with the execution of such tests")
	}
}
