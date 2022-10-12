//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"log"
	"strconv"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var propertiesCmd = &cobra.Command{
	Use:   "properties connector_id",
	Short: "List connector properties",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connector, _ := strconv.Atoi(args[0])
		if connector <= 0 {
			log.Fatalf("invalid connector ID %q", args[0])
		}
		chichiapis.ListConnectorProperties(connector)
	},
}

func init() {
	connectorsCmd.AddCommand(propertiesCmd)
}
