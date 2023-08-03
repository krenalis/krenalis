//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"log"
	"os"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspacePingRedisCmd = &cobra.Command{
	Use:   "ping-redis <file>",
	Short: "Ping a Redis database",
	Long: "Ping a Redis database to validate the settings and establish a connection.\n\n" +
		"<file> is a JSON file containing the Redis database settings",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		settings, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		chichiapis.WorkspacePingRedis(settings)
	},
}

func init() {
	rootCmd.AddCommand(workspacePingRedisCmd)
}
