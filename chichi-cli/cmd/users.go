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

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List the users' Golden Records",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.ListUsers()
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
}
