//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"log"
	"os"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var workspaceChangeWarehouseSettings = &cobra.Command{
	Use:   "change-warehouse-settings <mode> <file>",
	Short: "Change the warehouse settings of the workspace",
	Long: "Change the warehouse settings of the workspace\n\n" +
		"<file> is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		mode := args[0]
		filename := args[1]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Validate(settings); err != nil {
			log.Fatalf("content of file %q is not JSON valid: %s", filename, err)
		}
		meergoapis.ChangeWarehouseSettings(workspace(cmd), mode, settings)
	},
}

func init() {
	rootCmd.AddCommand(workspaceChangeWarehouseSettings)
}
