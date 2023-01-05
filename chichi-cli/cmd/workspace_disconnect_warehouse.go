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

var workspaceDisconnectWarehouseCmd = &cobra.Command{
	Use:   "disconnect-warehouse",
	Short: "Disconnect the data warehouse",
	Long:  "Disconnect the data warehouse of the workspace",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceDisconnectWarehouse()
	},
}

func init() {
	rootCmd.AddCommand(workspaceDisconnectWarehouseCmd)
}
