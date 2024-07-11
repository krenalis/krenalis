//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var workspaceInitWarehouseCmd = &cobra.Command{
	Use:   "init-warehouse <workspace>",
	Short: "Initialize the data warehouse",
	Long:  "Initialize the connected data warehouse by creating the supporting tables.",
	Run: func(cmd *cobra.Command, args []string) {
		meergoapis.WorkspaceInitWarehouse(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(workspaceInitWarehouseCmd)
}
