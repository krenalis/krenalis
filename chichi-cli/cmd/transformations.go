//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package cmd

import (
	"github.com/spf13/cobra"
)

var transformationsCmd = &cobra.Command{
	Use:   "transformations",
	Short: "Interact with transformation functions",
}

func init() {
	rootCmd.AddCommand(transformationsCmd)
}
