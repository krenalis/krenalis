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

	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var workspaceConnectWarehouseCmd = &cobra.Command{
	Use:   "connect-warehouse <type> <file> [--init|--repair]",
	Short: "Connect the workspace to a data warehouse",
	Long: "Connect the workspace to a data warehouse.\n\n" +
		"<type> is the data warehouse type and can be ClickHouse, PostgreSQL or Snowflake,\n" +
		"<file> is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		typ := args[0]
		filename := args[1]
		init, err := cmd.Flags().GetBool("init")
		if err != nil {
			log.Fatal(err)
		}
		repair, err := cmd.Flags().GetBool("repair")
		if err != nil {
			log.Fatal(err)
		}
		if init && repair {
			log.Fatalf("cannot specify both the --init and --repair options")
		}
		behavior := meergoapis.FailOnCheck
		if init {
			behavior = meergoapis.InitializeWarehouse
		} else if repair {
			behavior = meergoapis.RepairWarehouse
		}
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(settings) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		meergoapis.WorkspaceConnectWarehouse(workspace(cmd), typ, settings, behavior)
	},
}

func init() {
	_ = workspaceConnectWarehouseCmd.Flags().BoolP("init", "i", false, "initialize the warehouse when connecting it")
	_ = workspaceConnectWarehouseCmd.Flags().BoolP("repair", "r", false, "repair the warehouse when connecting it")
	rootCmd.AddCommand(workspaceConnectWarehouseCmd)
}
