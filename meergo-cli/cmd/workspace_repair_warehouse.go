//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var repairWarehouse = &cobra.Command{
	Use:   "repair-warehouse <workspace>",
	Short: "Repair the warehouse of the workspace",
	Run: func(cmd *cobra.Command, args []string) {
		meergoapis.RepairWarehouse(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(repairWarehouse)
}
