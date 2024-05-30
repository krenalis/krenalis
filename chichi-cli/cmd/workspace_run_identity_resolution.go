//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"github.com/open2b/chichi/chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var runIdentityResolution = &cobra.Command{
	Use:   "run-identity-resolution <workspace>",
	Short: "Run the Identity Resolution",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.RunIdentityResolution(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(runIdentityResolution)
}
