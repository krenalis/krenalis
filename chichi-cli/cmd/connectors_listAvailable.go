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

var listAvailableCmd = &cobra.Command{
	Use: "list-available",
}

func init() {
	connectorsCmd.AddCommand(listAvailableCmd)
}
