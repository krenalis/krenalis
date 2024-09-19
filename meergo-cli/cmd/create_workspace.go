//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"fmt"
	"log"

	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var createWorkspace = &cobra.Command{
	Use:   "create-workspace",
	Short: "Create a workspace",
	Run: func(cmd *cobra.Command, args []string) {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatal(err)
		}
		privacyRegion, err := cmd.Flags().GetString("privacy-region")
		if err != nil {
			log.Fatal(err)
		}
		id := meergoapis.CreateWorkspace(name, meergoapis.PrivacyRegion(privacyRegion))
		fmt.Printf("Created workspace with ID: %d\n", id)
	},
}

func init() {
	createWorkspace.Flags().StringP("name", "n", "Workspace", "the name of the workspace")
	createWorkspace.Flags().StringP("privacy-region", "r", "", "privacy region")
	rootCmd.AddCommand(createWorkspace)
}
