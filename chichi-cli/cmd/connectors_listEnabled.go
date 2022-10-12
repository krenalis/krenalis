//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var listEnabledCmd = &cobra.Command{
	Use: "list-enabled",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.ListEnabledConnectors()
	},
}

func init() {
	connectorsCmd.AddCommand(listEnabledCmd)
}
