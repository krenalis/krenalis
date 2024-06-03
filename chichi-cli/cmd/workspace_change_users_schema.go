//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/open2b/chichi/chichi-cli/chichiapis"
	"github.com/open2b/chichi/types"

	"github.com/spf13/cobra"
)

var workspaceChangeUsersSchema = &cobra.Command{
	Use:  "change-users-schema <file>",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(content) {
			log.Fatalf("content of file %q is not JSON valid", filename)
		}
		var req struct {
			Schema  types.Type
			RePaths map[string]any
		}
		err = json.Unmarshal(content, &req)
		if err != nil {
			log.Fatalf("cannot unmarshal content of JSON file %q: %s", filename, err)
		}
		if !req.Schema.Valid() {
			log.Fatalf("field 'Schema' in JSON must refer to a valid schema")
		}
		queries := chichiapis.WorkspaceChangeUsersSchemaQueries(workspace(cmd), req.Schema, req.RePaths)
		if len(queries) == 0 {
			fmt.Printf("It looks like the 'users' / 'user_identities' schemas"+
				" on the data warehouse already match with the schema indicated in %q,"+
				" so there are no queries to execute, exiting\n", filename)
			return
		}
		fmt.Print("These queries will be executed:\n\n")
		for _, query := range queries {
			fmt.Printf("%s\n\n", query)
		}
		if !askUserConfirmation("Are you sure you want to proceed?") {
			log.Fatalf("exiting")
		}
		chichiapis.WorkspaceChangeUsersSchema(workspace(cmd), req.Schema, req.RePaths)
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
	rootCmd.AddCommand(workspaceChangeUsersSchema)
}
