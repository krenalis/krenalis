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

var workspaceReloadSchemasCmd = &cobra.Command{
	Use:   "reload-schemas <workspace>",
	Short: "Reload the schemas",
	Long:  "Reload the schemas of the data warehouse",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceReloadSchemas(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(workspaceReloadSchemasCmd)
}
