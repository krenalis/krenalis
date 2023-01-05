//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceReloadSchemaCmd = &cobra.Command{
	Use:   "reload-schema <workspace>",
	Short: "Reload the schema",
	Long:  "Reload the schema of the data warehouse",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceReloadSchema()
	},
}

func init() {
	rootCmd.AddCommand(workspaceReloadSchemaCmd)
}
