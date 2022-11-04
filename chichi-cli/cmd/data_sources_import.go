//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"log"
	"strconv"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import connector_id",
	Short: "Import data from a data source",
	Long:  `Import data from a data source, starting from the last import.`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connector, _ := strconv.Atoi(args[0])
		if connector <= 0 {
			log.Fatalf("invalid connector ID %q", args[0])
		}
		chichiapis.ImportUsersFromDataSource(connector, false)
	},
}

func init() {
	dataSourcesCmd.AddCommand(importCmd)
}
