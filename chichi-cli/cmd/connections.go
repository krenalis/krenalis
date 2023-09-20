//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Interact with connections",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.ListConnections(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(connectionsCmd)
}
