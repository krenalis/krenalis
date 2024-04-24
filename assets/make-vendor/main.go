//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/exp/maps"
)

func main() {

	var explicit bool
	flag.BoolVar(&explicit, "x", false, "print each step explicitly")
	flag.Parse()

	err := makeVendor(explicit)
	if err != nil {
		panic(err)
	}
}

func makeVendor(explicit bool) error {

	printX := func(format string, args ...any) {
		if !explicit {
			return
		}
		fmt.Printf(format, args...)
	}

	root, err := moveToModuleRoot()
	if err != nil {
		return err
	}

	entryPoint := filepath.Join(root, "assets", "src", "index.jsx")
	nodeModulesDir := filepath.Join(root, "assets", "node_modules") + string(os.PathSeparator)

	// Create the out directory used by esbuild.
	outDir, err := os.MkdirTemp("", "chichi-ui-make-vendor-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(outDir)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: could not remove the temporary directory %q: %s", outDir, err)
		}
		printX("temporary directory %q removed\n", outDir)
	}()
	printX("created the temporary directory %q\n", outDir)

	var resolve = newResolveFile()

	plugin := api.Plugin{
		Name: "resolver",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^.*$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if args.Path == entryPoint {
						return api.OnResolveResult{}, nil
					}
					var key string
					if dir, ok := strings.CutPrefix(args.ResolveDir, nodeModulesDir); ok {
						key = pathKey(dir, args.Path)
					} else if isPackagePath(args.Path) {
						key = path.Clean(args.Path)
					}
					if key != "" {
						if resolve.Contains(key) {
							return api.OnResolveResult{}, nil
						}
					}
					printX("resolve: Path: %q, Importer: %s, Namespace: %s, ResolveDir: %q, Kind: %d\n",
						args.Path, args.Importer, args.Namespace, args.ResolveDir, args.Kind)
					if key == "" {
						printX("\t--- skip")
						return api.OnResolveResult{}, nil
					}
					printX("\t--- resolve %q\n", args.Path)
					resolve.AddImport(key, "", false)
					result := build.Resolve(args.Path, api.ResolveOptions{
						Importer:   args.Importer,
						Namespace:  args.Namespace,
						Kind:       args.Kind,
						ResolveDir: args.ResolveDir,
					})
					if len(result.Errors) > 0 {
						printX("\t--- errors %#v\n", result.Errors)
						return api.OnResolveResult{Errors: result.Errors}, nil
					}
					value, ok := strings.CutPrefix(result.Path, nodeModulesDir)
					if !ok {
						return api.OnResolveResult{}, fmt.Errorf("esbuild has resolved %s to a path that is not in 'node_packages' directory: %s", key, result.Path)
					}
					value = filepath.ToSlash(value)
					resolve.AddImport(key, value, result.SideEffects)
					printX("\t--- %q --> %s\n", key, value)
					res := api.OnResolveResult{
						Path:        result.Path,
						External:    result.External,
						Namespace:   result.Namespace,
						SideEffects: api.SideEffectsFalse,
					}
					if result.SideEffects {
						res.SideEffects = api.SideEffectsTrue
					}
					return res, nil
				})
		},
	}

	// Run esbuild.
	printX("running esbuild...\n")
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{entryPoint},
		Format:            api.FormatESModule,
		JSX:               api.JSXAutomatic,
		LegalComments:     api.LegalCommentsNone,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            outDir,
		Plugins:           []api.Plugin{plugin},
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             true,
	})
	if result.Errors != nil {
		msg := "cannot generate UI assets when making vendor:"
		for _, err := range result.Errors {
			if len(result.Errors) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		return errors.New(msg)
	}
	if result.Warnings != nil {
		msg := "cannot generate UI assets when making vendor:"
		for _, err := range result.Warnings {
			if len(result.Warnings) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		return errors.New(msg)
	}
	printX("esbuild execution completed\n")

	// Copy the resolved files from the "node_modules" directory to "vendor".
	paths := resolve.ResolvedPaths()
	printX("preparing writing of %d file(s) to 'assets/vendor'...\n", len(paths))
	err = os.RemoveAll("./assets/vendor")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot remove the current 'assets/vendor' directory: %s", err)
	}
	printX("current directory 'assets/vendor' removed\n")
	packages := map[string]struct{}{}
	for _, name := range paths {
		src := filepath.Join("assets/node_modules", name)
		dst := filepath.Join("assets/vendor", name)
		err := os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}
		err = copyFile(dst, src)
		if err != nil {
			return err
		}
		i := 0
		if strings.HasPrefix(name, "@") {
			i = strings.Index(name, "/") + 1
		}
		for {
			if p := strings.Index(name[i:], "/"); p != -1 {
				i += p
				packages[name[:i]] = struct{}{}
			}
			if p := strings.Index(name[i:], "/node_modules/"); p != -1 {
				i += p + len("/node_modules/")
				continue
			}
			break
		}
	}

	// Copy 'package.json' and 'LICENSE' files.
	for dir := range packages {
		src := filepath.Join("assets/node_modules", dir, "package.json")
		dst := filepath.Join("assets/vendor", dir, "package.json")
		err = copyPackageFile(dst, src)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		for _, license := range []string{"LICENSE", "license", "License"} {
			src = filepath.Join("assets/node_modules", dir, license)
			dst = filepath.Join("assets/vendor", dir, strings.ToUpper(license))
			err = copyFile(dst, src)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}

	// Save the 'resolve.json' file.
	b, _ := resolve.MarshalJSON()
	err = os.WriteFile("assets/vendor/resolve.json", b, 0644)
	if err != nil {
		return err
	}

	return nil
}

type resolveFile struct {
	imports     map[string]string
	sideEffects map[string]bool
	mu          sync.Mutex
}

func newResolveFile() *resolveFile {
	return &resolveFile{
		imports:     map[string]string{},
		sideEffects: map[string]bool{},
	}
}

func (f *resolveFile) AddImport(path, value string, sideEffects bool) {
	f.mu.Lock()
	f.imports[path] = value
	f.sideEffects[path] = sideEffects
	f.mu.Unlock()
}

func (f *resolveFile) Contains(path string) bool {
	f.mu.Lock()
	_, ok := f.imports[path]
	f.mu.Unlock()
	return ok
}

func (f *resolveFile) ResolvedPaths() []string {
	values := maps.Values(f.imports)
	slices.Sort(values)
	return values
}

func (f *resolveFile) MarshalJSON() ([]byte, error) {
	f.mu.Lock()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	b.WriteString("{")
	paths := maps.Keys(f.imports)
	slices.Sort(paths)
	for i, name := range paths {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n\t")
		_ = enc.Encode(name)
		b.Truncate(b.Len() - 1)
		b.WriteString(": ")
		value := f.imports[name]
		if f.sideEffects[name] {
			value += ";true"
		} else {
			value += ";false"
		}
		_ = enc.Encode(value)
		b.Truncate(b.Len() - 1)
	}
	b.WriteString("\n}")
	f.mu.Unlock()
	return b.Bytes(), nil
}

func pathKey(dir, name string) string {
	dir = filepath.ToSlash(dir)
	if isPackagePath(name) {
		i := strings.LastIndex(dir, "/node_modules/")
		if i != -1 {
			dir = dir[:i]
		}
	}
	return path.Join(dir, name)
}

func isPackagePath(name string) bool {
	return name != "." && name != ".." && !strings.HasPrefix(name, "./") && !strings.HasPrefix(name, "../") && !strings.HasPrefix(name, "/")
}

func copyFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

type PackageFile struct {
	Name        any `json:"name,omitempty"`
	Version     any `json:"version,omitempty"`
	Author      any `json:"author,omitempty"`
	License     any `json:"license,omitempty"`
	Type        any `json:"type,omitempty"`
	Main        any `json:"main,omitempty"`
	Browser     any `json:"browser,omitempty"`
	Module      any `json:"module,omitempty"`
	Imports     any `json:"imports,omitempty"`
	Exports     any `json:"exports,omitempty"`
	SideEffects any `json:"sideEffects,omitempty"`
}

func copyPackageFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	file := PackageFile{}
	err = json.Unmarshal(data, &file)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func moveToModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		st, err := os.Stat("go.mod")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if !errors.Is(err, os.ErrNotExist) && st.Mode().IsRegular() {
			return cwd, nil
		}
		err = os.Chdir("..")
		if err != nil {
			return "", err
		}
		parent, err := os.Getwd()
		if err != nil {
			return "", err
		}
		if parent == cwd {
			return "", errors.New("go.mod file not found in current directory or any parent directory")
		}
		cwd = parent
	}
}
