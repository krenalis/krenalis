// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/krenalis/krenalis/test/snowflaketester"
)

func main() {

	testEnv, err := snowflaketester.CreateTestEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := testEnv.Teardown()
		if err != nil {
			log.Printf("cannot teardown database: %s", err)
		}
	}()

	stdinReader := bufio.NewReader(os.Stdin)

	for {

		fmt.Printf("The parameters to access your Snowflake testing schema: \n\n")
		fmt.Printf("    Account:   %s\n", testEnv.Settings().Account)
		fmt.Printf("    User:      %s\n", testEnv.Settings().User)
		fmt.Printf("    Password:  ***\n")
		fmt.Printf("    Database:  %s\n", testEnv.Settings().Database)
		fmt.Printf("    Role:      %s\n", testEnv.Settings().Role)
		fmt.Printf("    Schema:    %s\n", testEnv.Settings().Schema)
		fmt.Printf("    Warehouse: %s\n", testEnv.Settings().Warehouse)
		fmt.Printf("\nType 'teardown' and press ENTER to teardown schema and quit: ")

		input, err := stdinReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.Trim(input, "\n")
		if input == "teardown" {
			break
		}

	}

}
