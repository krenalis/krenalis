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

var propertiesCmd = &cobra.Command{
	Use:   "properties connection_id",
	Short: "List connection properties",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connection, _ := strconv.Atoi(args[0])
		if connection <= 0 {
			log.Fatalf("invalid connection ID %q", args[0])
		}
		chichiapis.ListConnectionsProperties(connection)
	},
}

func init() {
	connectionsCmd.AddCommand(propertiesCmd)
}
