//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var runIdentityResolution = &cobra.Command{
	Use:   "run-identity-resolution <workspace>",
	Short: "Run the Identity Resolution",
	Run: func(cmd *cobra.Command, args []string) {
		meergoapis.RunIdentityResolution(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(runIdentityResolution)
}
