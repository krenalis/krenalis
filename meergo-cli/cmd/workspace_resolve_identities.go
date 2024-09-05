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

var resolveIdentities = &cobra.Command{
	Use:   "resolve-identities <workspace>",
	Short: "Resolve the identities of the workspace",
	Run: func(cmd *cobra.Command, args []string) {
		meergoapis.ResolveIdentities(workspace(cmd))
	},
}

func init() {
	rootCmd.AddCommand(resolveIdentities)
}
