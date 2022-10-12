//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use: "users",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("users called")
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
}
