//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2023 Open2b
//

package cmd

import (
	"encoding/json"
	"log"
	"os"

	"chichi-cli/chichiapis"
	"chichi/apis"

	"github.com/spf13/cobra"
)

var workspaceConnectWarehouseCmd = &cobra.Command{
	Use:   "connect-warehouse <file>",
	Short: "Connect a data warehouse to the workspace",
	Long: "Connect a data warehouse to the workspace.\n\n" +
		"<file> must be a JSON file containing an object " +
		"which can be serialized into a apis.PostgreSQLSettings value",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		var setts apis.PostgreSQLSettings
		err = json.NewDecoder(f).Decode(&setts)
		if err != nil {
			log.Fatal(err)
		}
		chichiapis.WorkspaceConnectWarehouse(setts)
	},
}

func init() {
	rootCmd.AddCommand(workspaceConnectWarehouseCmd)
}
