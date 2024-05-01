//go:generate go run github.com/open2b/chichi/assets

// To compile:
//
//  1. go generate
//  2. go build
//
// To add your custom connectors or choose what connector to build into Chichi:
//
//  1. Create a new directory: mkdir chichi
//  2. Change into it: cd chichi
//  3. Copy this file into the new directory
//  4. Edit the copied file to add your connectors (optional):
//     import _ "github.com/example/connector"
//  5. Initialize a Go module: go mod init chichi
//  6. Tidy the module: go mod tidy
//  7. Generate the assets: go generate
//  8. Build: go build
//
// Note: Re-execute 'go generate' if you change Chichi module version.
//
// See also https://github.com/open2b/chichi/blob/main/doc/src/getting-started.md
package main

import (
	"embed"

	"github.com/open2b/chichi/cmd"

	// Add your custom connectors here:
	// _ "github.com/example/connector"

	// Imports the standard connectors:
	_ "github.com/open2b/chichi/connectors"
)

//go:embed chichi-assets
var assets embed.FS

func main() {
	cmd.Main(assets)
}
