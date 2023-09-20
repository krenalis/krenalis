//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"encoding/json"
	"log"
	"os"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceConnectWarehouseCmd = &cobra.Command{
	Use:   "connect-warehouse <type> <file>",
	Short: "Connect the workspace to a data warehouse",
	Long: "Connect the workspace to a data warehouse.\n\n" +
		"<type> is the data warehouse type and can be ClickHouse, PostgreSQL or Snowflake,\n" +
		"<file> is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		typ := args[0]
		filename := args[1]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(settings) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		chichiapis.WorkspaceConnectWarehouse(workspace(cmd), typ, settings)
	},
}

func init() {
	rootCmd.AddCommand(workspaceConnectWarehouseCmd)
}
