//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

var transformationsShowCmd = &cobra.Command{
	Use:  "list <data source ID>",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		data_source, _ := strconv.Atoi(args[0])
		if data_source <= 0 {
			log.Fatalf("invalid data source ID %q", args[0])
		}
		chichiapis.GetTransformations(data_source)
	},
}

func init() {
	transformationsCmd.AddCommand(transformationsShowCmd)
}
