//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "chichi-cli",
}

func init() {

	// Prevent Cobra from creating a default 'completion' command.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Exit when the parsing of a flag fails.
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		log.Fatal(err)
		return nil
	})

	// Determine the workspace in this way:
	//
	//  * if the "--workspace" flag is passed, then use its value.
	//  * otherwise, if the environment variable CHICHI_CLI_WORKSPACE is set,
	//    use its value.
	//  * otherwise use 1.
	//
	// Exit with error if the value for "--workspace" or for
	// CHICHI_CLI_WORKSPACE is not valid.
	chichiCliWs := os.Getenv("CHICHI_CLI_WORKSPACE")
	if chichiCliWs == "" {
		chichiCliWs = "1"
	}
	defaultWs, err := strconv.Atoi(chichiCliWs)
	if err != nil {
		log.Fatalf("invalid value for CHICHI_CLI_WORKSPACE: %s", err)
	}

	rootCmd.PersistentFlags().IntP("workspace", "w", defaultWs, "Workspace. Defaults to CHICHI_CLI_WORKSPACE, if set, otherwise to 1")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func workspace(cmd *cobra.Command) int {
	ws, err := cmd.Flags().GetInt("workspace")
	if err != nil {
		log.Fatal(err)
	}
	return ws
}
