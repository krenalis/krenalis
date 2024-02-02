//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceRunIdentityResolution = &cobra.Command{
	Use:   "run-workspace-identity-resolution <workspace>",
	Short: "Run the Workspace Identity Resolution",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceRunIdentityResolution(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(workspaceRunIdentityResolution)
}
