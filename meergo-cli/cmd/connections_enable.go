//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"log"
	"strconv"

	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <connection>",
	Short: "enable a connection",
	Long:  "enable a connection",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connection, _ := strconv.Atoi(args[0])
		if connection <= 0 {
			log.Fatalf("invalid connection Id %q", args[0])
		}
		meergoapis.EnableConnection(workspace(cmd), connection)
	},
}

func init() {
	connectionsCmd.AddCommand(enableCmd)
}
