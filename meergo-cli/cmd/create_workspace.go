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
	"os"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var createWorkspace = &cobra.Command{
	Use:   "create-workspace <warehouse-name> <warehouse-settings-file>",
	Short: "Create a workspace",
	Long: "Create a workspace with an associated data warehouse.\n\n" +
		"<warehouse-name>          is the name of the data warehouse and can be PostgreSQL or Snowflake\n" +
		"<warehouse-settings-file> is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		whName := args[0]
		whSettingsFile := args[1]
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatal(err)
		}
		privacyRegion, err := cmd.Flags().GetString("privacy-region")
		if err != nil {
			log.Fatal(err)
		}
		settings, err := os.ReadFile(whSettingsFile)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Validate(settings); err != nil {
			log.Fatalf("content of file %q is not JSON valid: %s", whSettingsFile, err)
		}
		id := meergoapis.CreateWorkspace(name, meergoapis.PrivacyRegion(privacyRegion), whName, settings)
		fmt.Printf("Created workspace with ID: %d\n", id)
	},
}

func init() {
	createWorkspace.Flags().StringP("name", "n", "Workspace", "the name of the workspace")
	createWorkspace.Flags().StringP("privacy-region", "r", "", "privacy region")
	rootCmd.AddCommand(createWorkspace)
}
