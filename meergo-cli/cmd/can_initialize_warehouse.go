//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
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

var canInitializeWarehouse = &cobra.Command{
	Use:   "can-initialize-warehouse <name> <file>",
	Short: "Check if can initialize a warehouse",
	Long: "Check if can initialize a warehouse.\n\n" +
		"<name>  is the name of the data warehouse and can be PostgreSQL or Snowflake\n" +
		"<file>  is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		typ := args[0]
		filename := args[1]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Validate(settings); err != nil {
			log.Fatalf("content of file %q is not JSON valid: %s", filename, err)
		}
		meergoapis.CanInitializeWarehouse(typ, settings)
		fmt.Println("Yes, the data warehouse is initializable.")
	},
}

func init() {
	rootCmd.AddCommand(canInitializeWarehouse)
}
