//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceInitWarehouseCmd = &cobra.Command{
	Use:   "init-warehouse <workspace>",
	Short: "Initialize the data warehouse",
	Long:  "Initialize the connected data warehouse by creating the supporting tables.",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceInitWarehouse()
	},
}

func init() {
	rootCmd.AddCommand(workspaceInitWarehouseCmd)
}
