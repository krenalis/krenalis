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

var dataSourcesCmd = &cobra.Command{
	Use:   "data-sources",
	Short: "Interact with data sources",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.ListDataSources()
	},
}

func init() {
	rootCmd.AddCommand(dataSourcesCmd)
}
