//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// This program builds and compresses the assets, storing them in a directory
// named 'meergo-assets' within the current working directory. If a directory
// with the same name already exists, it will be deleted.
package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/meergo/meergo/json"

	"github.com/andybalholm/brotli"
	"github.com/evanw/esbuild/pkg/api"
)

//go:embed package.json public/index.html all:src tsconfig.json all:node_modules_vendor node_modules_vendor/resolve.json
var assetsFS embed.FS

// Path to the Shoelace icons within the "node_modules" directory.
const shoelaceIconsPath = "@shoelace-style/shoelace/dist/assets/icons"

// Destination directory for the Shoelace icons in "meergo-assets".
const shoelaceIconsDir = "shoelace/dist/assets/icons"

func main() {
	err := buildAssets()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(" ✔ The asset files have been generated.")
}

var generatedFiles = []string{"index.html", "index.js", "index.js.map", "index.css", "index.css.map"}

// buildAssets builds and the assets.
func buildAssets() error {

	// Create the directory used to build the assets.
	buildDir, err := os.MkdirTemp("", "meergo-assets-build")
	if err != nil {
		return fmt.Errorf("cannot create a temporary directory: %s", err)
	}
	defer func() {
		err = os.Chdir("..")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: cannot chdir to directory %q: %s\n", filepath.Join(buildDir, ".."), err)
		}
		if err = os.RemoveAll(buildDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: cannot remove temporary directory %q: %s\n", buildDir, err)
		}
	}()

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %s", err)
	}

	err = os.Chdir(buildDir)
	if err != nil {
		return fmt.Errorf("cannot chdir to build directory: %s", err)
	}
	buildDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %s", err)
	}
	buildDir += string(os.PathSeparator)

	uiDir := buildDir + "ui" + string(os.PathSeparator)
	outDir := buildDir + "out" + string(os.PathSeparator)
	dstDir := buildDir + "dst" + string(os.PathSeparator)

	for _, dir := range []string{outDir, dstDir} {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return fmt.Errorf("cannot create directory %q: %s", dir, err)
		}
	}

	// Copy the UI's assets into the uiDir directory.
	err = os.CopyFS(uiDir, assetsFS)
	if err != nil {
		return fmt.Errorf("cannot copy assets: %s", err)
	}

	// Read the 'resolve.json' file.
	resolve, err := readResolveFile()
	if err != nil {
		return fmt.Errorf("cannot read the resolve file: %s", err)
	}

	// Bundle the UI's assets.
	err = build(outDir, uiDir, resolve)
	if err != nil {
		return err
	}

	// Verify that all assets are been generated.
	for _, file := range []string{"index.js", "index.js.map", "index.css", "index.css.map"} {
		st, err := os.Stat(outDir + file)
		if err != nil {
			return fmt.Errorf("cannot stat file %q: %s", outDir+file, err)
		}
		if st.Size() == 0 {
			return fmt.Errorf("bundled file %q is empty", file)
		}
	}

	// Copy the "index.html" file.
	data, _ := assetsFS.ReadFile("public/index.html")
	err = os.WriteFile(outDir+"index.html", data, 0666)
	if err != nil {
		return err
	}

	// Copy the Shoelace icons.
	shoelaceIconsFS, err := fs.Sub(assetsFS, path.Join("node_modules_vendor", shoelaceIconsPath))
	if err != nil {
		return fmt.Errorf("cannot create an fs.FS for the Shoelace icons directory: %s", err)
	}
	err = os.CopyFS(outDir+shoelaceIconsDir, shoelaceIconsFS)
	if err != nil {
		return fmt.Errorf("cannot copy Shoelace icons: %s", err)
	}

	// Read the Shoelace icons names and append them to the generatedFiles slice.
	shoelaceIconsNames, err := fs.Glob(shoelaceIconsFS, "*")
	if err != nil {
		return fmt.Errorf("cannot glob Shoelace icons: %s", err)
	}
	for _, name := range shoelaceIconsNames {
		generatedFiles = append(generatedFiles, path.Join(shoelaceIconsDir, name))
	}
	err = os.MkdirAll(path.Join(dstDir, shoelaceIconsDir), 0o777)
	if err != nil {
		return fmt.Errorf("cannot create the Shoelace icons directory into the destination dir: %s", err)
	}

	// Compress the UI's assets.
	var in, out *os.File
	var bw *brotli.Writer
	defer func() {
		if in != nil {
			_ = in.Close()
		}
		if bw != nil {
			_ = bw.Close()
		}
		if out != nil {
			_ = out.Close()
		}
	}()
	for _, name := range generatedFiles {

		srcPath := outDir + name
		dstPath := dstDir + name + ".br"

		// Compress the file.
		in, err = os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("cannot open file %q: %s", srcPath, err)
		}

		out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("cannot create file %q: %s", srcPath+".br", err)
		}

		bw := brotli.NewWriter(out)

		_, err = io.Copy(bw, in)
		if err != nil {
			return fmt.Errorf("cannot compress file %q as %q: %s", srcPath, dstPath, err)
		}

		_ = in.Close()
		in = nil

		err = bw.Close()
		if err != nil {
			return fmt.Errorf("cannot compress file %q as %q: %s", srcPath, dstPath, err)
		}
		bw = nil

		err = out.Close()
		if err != nil {
			return fmt.Errorf("cannot close file %q: %s", dstPath, err)
		}
		out = nil

		// Verify the compressed file.
		in, err = os.Open(dstPath)
		if err != nil {
			return fmt.Errorf("cannot open file %q: %s", dstPath+".br", err)
		}
		br := brotli.NewReader(in)
		_, err = io.Copy(io.Discard, br)
		if err != nil {
			return fmt.Errorf("cannot read file %q: %s", dstPath, err)
		}
		_ = in.Close()
		in = nil

	}

	// Copy the assets to the destination.
	for _, file := range generatedFiles {
		err = os.Remove(filepath.Join(root, "meergo-assets", file+".br"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	err = os.CopyFS(filepath.Join(root, "meergo-assets"), os.DirFS(dstDir))
	if err != nil {
		return err
	}

	return nil
}

// build builds the assets located in the assetsDir directory and saves them
// into the outDir directory. If resolve is not nil, it will be used to resolve
// the paths.
func build(outDir, assetsDir string, resolve map[string]string) error {

	var err error
	assetsDir, err = filepath.Abs(assetsDir)
	if err != nil {
		return err
	}

	entryPoint := filepath.Join(assetsDir, "src", "index.tsx")
	vendorDir := filepath.Join(assetsDir, "node_modules_vendor") + string(os.PathSeparator)

	var plugins []api.Plugin
	if resolve != nil {
		plugin := api.Plugin{
			Name: "resolve_from_vendor",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `.*`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						if args.Kind == api.ResolveEntryPoint {
							return api.OnResolveResult{}, nil
						}
						if strings.HasPrefix(args.Path, "data:") {
							return api.OnResolveResult{}, nil
						}
						var key string
						if dir, ok := strings.CutPrefix(args.ResolveDir, vendorDir); ok {
							key = pathKey(dir, args.Path)
						} else if isPackagePath(args.Path) {
							key = path.Clean(args.Path)
						} else {
							return api.OnResolveResult{}, nil
						}
						value, ok := resolve[key]
						if !ok {
							return api.OnResolveResult{}, fmt.Errorf("vendor does not contain the key %q (imported as %q from %q)", key, args.Path, args.ResolveDir)
						}
						var sideEffect api.SideEffects
						if value, ok = strings.CutSuffix(value, ";true"); ok {
							sideEffect = api.SideEffectsTrue
						} else {
							value = strings.TrimSuffix(value, ";false")
							sideEffect = api.SideEffectsFalse
						}
						value = filepath.ToSlash(value)
						res := api.OnResolveResult{
							Path:        filepath.Join(vendorDir, value),
							Namespace:   "file",
							SideEffects: sideEffect,
						}
						return res, nil
					})
			},
		}
		plugins = []api.Plugin{plugin}
	}

	// Bundle the assets.
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
		Plugins:           plugins,
		Sourcemap:         api.SourceMapLinked,
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             true,
	})
	if result.Errors != nil {
		var msg string
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

	return nil
}

// readResolveFile reads the 'resolve.json' file.
func readResolveFile() (map[string]string, error) {
	data, _ := assetsFS.ReadFile("node_modules_vendor/resolve.json")
	resolve := map[string]string{}
	err := json.Unmarshal(data, &resolve)
	if err != nil {
		return nil, err
	}
	return resolve, nil
}

// pathKey returns the key to use in the resolve.json file, relative to the
// given name when imported from the directory dir.
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

// isPackagePath reports whether name is a package path.
func isPackagePath(name string) bool {
	return name != "." && name != ".." && !strings.HasPrefix(name, "./") &&
		!strings.HasPrefix(name, "../") && !strings.HasPrefix(name, "/")
}
