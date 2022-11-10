//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"io"
	"log"
	"os"
	"strconv"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:  "update connection_id [filename | -]",
	Args: cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		connection, _ := strconv.Atoi(args[0])
		if connection <= 0 {
			log.Fatalf("invalid connection ID %q", args[0])
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
		chichiapis.UpdateTransformation(connection, transformation)
	},
}

func init() {
	transformationsCmd.AddCommand(updateCmd)
}
