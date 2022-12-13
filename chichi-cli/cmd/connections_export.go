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

var exportCmd = &cobra.Command{
	Use:   "export connector_id",
	Short: "export data to connection",
	Long:  `export data to connection`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connector, _ := strconv.Atoi(args[0])
		if connector <= 0 {
			log.Fatalf("invalid connector ID %q", args[0])
		}
		chichiapis.ExportUsersToConnection(connector)
	},
}

func init() {
	connectionsCmd.AddCommand(exportCmd)
}
