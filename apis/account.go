//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"regexp"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

var AuthenticationFailed errors.Code = "AuthenticationFailed"

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// Account represents an account.
type Account struct {
	db             *postgres.DB
	eventProcessor *eventProcessor
	state          *state.State
	account        *state.Account
	ID             int
	Name           string
	Email          string
	InternalIPs    []string
}

// Workspace returns the workspace with identifier id of the account.
//
// It returns an errors.NotFound error if the workspace does not exist.
func (this *Account) Workspace(id int) (*Workspace, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid workspace identifier", id)
	}
	ws, ok := this.account.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	return &Workspace{db: this.db, eventProcessor: this.eventProcessor, state: this.state, workspace: ws}, nil
}

// DeprecatedProperty returns an instance of DeprecatedProperties which operates
// on the given property.
func (this *Account) DeprecatedProperty(property int) *DeprecatedProperties {
	properties := &DeprecatedProperties{
		Account: this,
		id:      property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
