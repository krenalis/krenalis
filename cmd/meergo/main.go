//go:generate go run github.com/meergo/meergo/assets

// To compile:
//
//  1. go generate
//  2. go build
//
// To add your custom connectors or choose what connector to build into Meergo:
//
//  1. Create a new directory: mkdir meergo
//  2. Change into it: cd meergo
//  3. Copy this file into the new directory
//  4. Edit the copied file to add your connectors (optional):
//     import _ "github.com/example/connector"
//  5. Initialize a Go module: go mod init meergo
//  6. Tidy the module: go mod tidy
//  7. Generate the assets: go generate
//  8. Build: go build
//
// Note: Re-execute 'go generate' if you change Meergo module version.
//
// See also https://github.com/meergo/meergo/blob/main/doc/src/getting-started.md
package main

import (
	"embed"

	"github.com/meergo/meergo/cmd"

	// Add your custom connectors and data warehouses here:
	// _ "github.com/example/connector"
	// _ "github.com/example/warehouse"

	// Imports the standard connectors:
	_ "github.com/meergo/meergo/connectors"

	// Imports the standard data warehouses:
	_ "github.com/meergo/meergo/warehouses"
)

//go:embed meergo-assets
var assets embed.FS

func main() {
	cmd.Main(assets)
}
