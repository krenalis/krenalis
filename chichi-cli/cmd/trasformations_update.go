//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"chichi-cli/chichiapis"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:  "update connector_id [filename | -]",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connector, _ := strconv.Atoi(args[0])
		if connector <= 0 {
			log.Fatalf("invalid connector ID %q", args[0])
		}
		source := args[1]
		var transformation []byte
		if source == "-" {
			stdin, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("cannot read from stdin: %s", err)
			}
			transformation = stdin
		} else {
			var err error
			transformation, err = os.ReadFile(source)
			if err != nil {
				log.Fatalf("cannot read from file: %s", err)
			}
		}
		chichiapis.UpdateTransformation(connector, transformation)
	},
}

func init() {
	transformationsCmd.AddCommand(updateCmd)
}
