//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var workspaceDisconnectRedisCmd = &cobra.Command{
	Use:   "disconnect-redis",
	Short: "Disconnect the Redis database",
	Long:  "Disconnect the Redis database of the workspace",
	Run: func(cmd *cobra.Command, args []string) {
		chichiapis.WorkspaceDisconnectRedis()
	},
}

func init() {
	rootCmd.AddCommand(workspaceDisconnectRedisCmd)
}
