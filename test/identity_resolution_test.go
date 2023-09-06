//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

// TestIdentityResolution tests the identity resolution by importing users and
// retrieving the users from the APIs.
//
// This works by importing users through a CSV file, which is created (or
// updated) every time an user is imported, then it's loaded into Chichi by
// running the import action on the CSV.
func TestIdentityResolution(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a storage where the the CSV files (containing the incoming users)
	// will be created.
	storageDir, err := os.MkdirTemp("", "chichi-test-identity-resolution")
	if err != nil {
		t.Fatal(err)
	}
	removeTempDirectory := false
	defer func() {
		if removeTempDirectory {
			err := os.RemoveAll(storageDir)
			if err != nil {
				t.Logf("cannot remove the temporary directory used by the storage: %s", err)
			}
		} else {
			t.Logf("the temporary directory for the storage %q has been kept for troubleshooting the test", storageDir)
		}
	}()

	csvFilename := "users.csv"
	csvAbsPath := filepath.Join(storageDir, csvFilename)

	// Create the Filesystem connection.
	fsID := c.AddSourceFilesystem(storageDir)

	// Create the CSV connection.
	csvID := c.AddSourceCSV(fsID)

	allProps := []string{"dummy_id", "Email"}
	identifiers := []string{"dummy_id", "Email"}

	// Generate and add an action to the CSV for importing the users.
	inSchemaProps := make([]types.Property, len(allProps))
	outSchemaProps := make([]types.Property, len(allProps))
	mapping := map[string]string{}
	for i, out := range allProps {
		in := fmt.Sprintf("column%d", i+1)
		inSchemaProps[i] = types.Property{Name: in, Type: types.Text()}
		outSchemaProps[i] = types.Property{Name: out, Type: types.Text()}
		mapping[out] = in
	}
	importUsersActionID := c.AddAction(csvID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name":        "Import users from CSV on Filesystem",
			"Path":        "users.csv",
			"InSchema":    types.Object(inSchemaProps),
			"OutSchema":   types.Object(outSchemaProps),
			"Identifiers": identifiers,
			"Mapping":     mapping,
		},
	})

	// Define a function "expectUsers" which checks if the expected users match
	// with the users on the data warehouse.
	expectUsers := func(expected []irProps) {

		// Retrieve the users from the APIs and convert their format.
		rawUsers := c.Users(allProps, 0, 1000)["users"].([]any)
		gotUsers := make([]irProps, len(rawUsers))
		for i := range rawUsers {
			u := map[string]string{}
			for j, p := range allProps {
				v := rawUsers[i].([]any)[j].(string)
				if v != "" {
					u[p] = v
				}
			}
			gotUsers[i] = u
		}

		// Check if the users are equal to the expected or not.
		if !reflect.DeepEqual(expected, gotUsers) {
			t.Fatalf("expecting: %v, got: %v", expected, gotUsers)
		}
		t.Logf("users: %v", gotUsers)
	}

	// Define a function "importUser" which imports the user into the data
	// warehouse.
	importUser := func(props irProps) {

		// Create a CSV file with the user.
		t.Logf("importing user %v", props)
		csvContent := createCSVContent(allProps, props)
		err := os.WriteFile(csvAbsPath, csvContent, 0755)
		if err != nil {
			log.Fatalf("cannot write the incoming user to the CSV file: %s", err)
		}

		// Import the users in the CSV.
		c.ExecuteAction(csvID, importUsersActionID, true)
		c.WaitActionsToFinish(csvID)

	}

	// -------------------------------------------------------------------------

	// Add the tests on the identity resolution here.

	expectUsers([]irProps{})

	importUser(irProps{"Email": "a@b"})
	expectUsers([]irProps{
		{"Email": "a@b"},
	})

	importUser(irProps{"Email": "c@d"})
	expectUsers([]irProps{
		{"Email": "a@b"},
		{"Email": "c@d"},
	})

	importUser(irProps{"dummy_id": "AAA", "Email": "a@b"})
	expectUsers([]irProps{
		{"Email": "a@b"},
		{"Email": "c@d"},
		{"dummy_id": "AAA", "Email": "a@b"},
	})

	importUser(irProps{"dummy_id": "AAA", "Email": "e@f"})
	expectUsers([]irProps{
		{"Email": "a@b"},
		{"Email": "c@d"},
		{"dummy_id": "AAA", "Email": "e@f"},
	})

	// TODO(Gianluca): see the issue
	// https://github.com/open2b/chichi/issues/254.
	//
	// importUser(irProps{"dummy_id": "AAA"})
	// expectUsers([]irProps{
	// 	{"Email": "a@b"},
	// 	{"Email": "c@d"},
	// 	{"dummy_id": "AAA", "Email": "e@f"},
	// })

	// -------------------------------------------------------------------------

	// The test completed successfully, so the temporary directory for the
	// storage can be removed.
	removeTempDirectory = true

}

type irProps map[string]string

// createCSVContent creates the content of a CSV file with all the given
// properties as headers and the user properties in the first row.
//
// If a property specified in allProps is not passed in the userProps, then it
// is left empty in the CSV.
func createCSVContent(allProps []string, userProps irProps) []byte {
	b := &bytes.Buffer{}
	firstRow := &bytes.Buffer{}
	for i, p := range allProps {
		if i > 0 {
			b.WriteByte(',')
			firstRow.WriteByte(',')
		}
		b.WriteString(p)
		if v, ok := userProps[p]; ok {
			firstRow.WriteString(v)
		}
	}
	b.WriteByte('\n')
	firstRow.WriteByte('\n')
	_, err := io.Copy(b, firstRow)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}
