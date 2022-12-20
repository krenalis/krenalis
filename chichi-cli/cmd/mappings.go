//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"github.com/spf13/cobra"
)

var mappingsCmd = &cobra.Command{
	Use:   "mappings",
	Short: "Interact with mappings of connections",
}

func init() {
	rootCmd.AddCommand(mappingsCmd)
}
