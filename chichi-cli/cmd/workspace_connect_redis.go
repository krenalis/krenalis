//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"encoding/json"
	"log"
	"os"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceConnectRedisCmd = &cobra.Command{
	Use:   "connect-redis <file>",
	Short: "Connect the workspace to a Redis database",
	Long: "Connect the workspace to a Redis database.\n\n" +
		"<file> is a JSON file containing the Redis database settings",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(settings) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		chichiapis.WorkspaceConnectRedis(settings)
	},
}

func init() {
	rootCmd.AddCommand(workspaceConnectRedisCmd)
}
