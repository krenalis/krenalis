//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"bufio"
	jsonstd "encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/meergo-cli/meergoapis"
	"github.com/meergo/meergo/types"

	"github.com/spf13/cobra"
)

var workspaceChangeUserSchema = &cobra.Command{
	Use:   "change-user-schema <file>",
	Short: "Change the user schema",
	Long: "Change the user schema.\n\n" +
		"<file> is a JSON file containing the new user schema",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		assumeYes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			log.Fatal(err)
		}
		filename := args[0]
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Validate(content); err != nil {
			log.Fatalf("content of file %q is not JSON valid: %s", filename, err)
		}
		var req struct {
			Schema  types.Type
			RePaths map[string]any
		}
		err = jsonstd.Unmarshal(content, &req)
		if err != nil {
			log.Fatalf("cannot unmarshal content of JSON file %q: %s", filename, err)
		}
		if !req.Schema.Valid() {
			log.Fatalf("field 'Schema' in JSON must refer to a valid schema")
		}
		queries := meergoapis.WorkspaceChangeUserSchemaQueries(workspace(cmd), req.Schema, req.RePaths)
		if len(queries) == 0 {
			fmt.Printf("It looks like the 'users' table schema on the data warehouse"+
				" already match with the schema indicated in %q,"+
				" so there are no queries to execute, exiting\n", filename)
			return
		}
		fmt.Print("These queries will be executed:\n\n")
		for _, query := range queries {
			fmt.Printf("%s\n\n", query)
		}
		if assumeYes {
			// Just go on.
		} else {
			if !askUserConfirmation("Are you sure you want to proceed?") {
				log.Fatalf("exiting")
			}
		}
		meergoapis.WorkspaceChangeUserSchema(workspace(cmd), req.Schema, req.RePaths)
		fmt.Print("Done!\n")
	},
}

func askUserConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", s)
		resp, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp == "y" || resp == "yes" {
			return true
		}
		if resp == "n" || resp == "no" {
			return false
		}
	}
}

func init() {
	_ = workspaceChangeUserSchema.Flags().Bool("yes", false, "Assume \"yes\" instead of asking for confirmation before changing schema")
	rootCmd.AddCommand(workspaceChangeUserSchema)
}
