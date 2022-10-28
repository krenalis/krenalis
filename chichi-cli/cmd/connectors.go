//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use: "connectors",
	Run: func(cmd *cobra.Command, args []string) {
		panic("TODO: not implemented")
	},
}

func init() {
	rootCmd.AddCommand(connectorsCmd)
}
