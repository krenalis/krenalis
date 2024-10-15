//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var workspaceCanChangeWarehouseSettings = &cobra.Command{
	Use:   "can-change-warehouse-settings <file>",
	Short: "Check if can change warehouse settings of the workspace",
	Long: "Check if can change warehouse settings\n\n" +
		"<file> is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(settings) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		meergoapis.CanChangeWarehouseSettings(workspace(cmd), settings)
		fmt.Println("Yes, the settings can be changed with the ones specified.")
	},
}

func init() {
	rootCmd.AddCommand(workspaceCanChangeWarehouseSettings)
}
