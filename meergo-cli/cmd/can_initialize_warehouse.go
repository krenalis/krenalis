//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/cobra"
)

var canInitializeWarehouse = &cobra.Command{
	Use:   "can-initialize-warehouse <type> <file>",
	Short: "Check if can initialize a warehouse",
	Long: "Check if can initialize a warehouse.\n\n" +
		"<type>  is the data warehouse type and can be PostgreSQL or Snowflake\n" +
		"<file>  is a JSON file containing the data warehouse settings",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		typ := args[0]
		filename := args[1]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(settings) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		meergoapis.CanInitializeWarehouse(typ, settings)
		fmt.Println("Yes, the data warehouse is initializable.")
	},
}

func init() {
	rootCmd.AddCommand(canInitializeWarehouse)
}
