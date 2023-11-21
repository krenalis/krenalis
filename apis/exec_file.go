//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chichi/apis/mappings"
)

// exportUsersToFile exports the users to the file.
func (this *Action) exportUsersToFile(ctx context.Context) error {

	users, err := this.readUsersFromDataWarehouse(ctx, nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.FilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	// Determine the columns of the exported file from the "users" schema.
	usersSchema, err := this.connection.schema(ctx, "users")
	if err != nil {
		return actionExecutionError{err}
	}
	if !usersSchema.Valid() {
		return actionExecutionError{errors.New("'users' schema not found")}
	}
	columns := usersSchema.Properties()

	// Prepare the users.
	rows := make([][]any, len(users))
	for i, u := range users {
		userSlice := make([]any, len(columns))
		for j, c := range columns {
			userSlice[j] = u.Properties[c.Name]
		}
		rows[i] = userSlice
	}

	// Write the file.
	path, err := replacePlaceholders(this.action.Path, newPathPlaceholderReplacer(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("invalid path: %s", err)
	}
	err = this.file().Write(ctx, path, this.action.Sheet, columns, rows)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %s", err)}
	}

	return nil
}

// newPathPlaceholderReplacer returns a placeholder replacer that replaces the
// following placeholders using time.Now().UTC() as current time.
//
//	${today}  which renders to something like:  2035-10-30
//	${now}    which renders to something like:  2035-10-30-16-33-25
//	${unix}   which renders to something like:  2077374805
//
// These placeholders are case-insensitive, so ${TODAY} is handled like
// ${today}.
func newPathPlaceholderReplacer(t time.Time) func(string) (string, bool) {
	return func(name string) (string, bool) {
		switch strings.ToLower(name) {
		case "today":
			return t.Format(time.DateOnly), true
		case "now":
			return t.Format("2006-01-02-15-04-05"), true
		case "unix":
			return strconv.FormatInt(t.Unix(), 10), true
		}
		return "", false
	}
}
