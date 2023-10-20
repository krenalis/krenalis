//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"regexp"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

var AuthenticationFailed errors.Code = "AuthenticationFailed"

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// Account represents an account.
type Account struct {
	apis        *APIs
	account     *state.Account
	ID          int
	Name        string
	Email       string
	InternalIPs []string
}

// AddWorkspace adds a workspace with the given name and privacy region, and
// returns its identifier. name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the account does not exist
// anymore.
func (this *Account) AddWorkspace(ctx context.Context, name string, region PrivacyRegion) (int, error) {

	this.apis.mustBeOpen()

	if name == "" || utf8.RuneCountInString(name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", name)
	}
	switch region {
	case PrivacyRegionNotSpecified, PrivacyRegionEurope:
	default:
		return 0, errors.BadRequest("privacy region is not valid")
	}

	n := state.AddWorkspace{
		Account:       this.account.ID,
		Name:          name,
		PrivacyRegion: state.PrivacyRegion(region),
	}

	// Generate the identifier.
	var err error
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO workspaces (id, account, name, privacy_region) VALUES ($1, $2, $3, $4)",
			n.ID, n.Account, n.Name, n.PrivacyRegion)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "workspaces_keys_account_fkey" {
					return errors.NotFound("account %d does not exist", n.Account)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// Workspace returns the workspace with identifier id of the account.
//
// It returns an errors.NotFound error if the workspace does not exist.
func (this *Account) Workspace(id int) (*Workspace, error) {
	this.apis.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid workspace identifier", id)
	}
	ws, ok := this.account.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	workspace := Workspace{
		apis:                 this.apis,
		account:              this,
		store:                this.apis.datastore.Store(id),
		workspace:            ws,
		ID:                   ws.ID,
		Name:                 ws.Name,
		Identifiers:          ws.Identifiers,
		AnonymousIdentifiers: AnonymousIdentifiers(ws.AnonymousIdentifiers),
		PrivacyRegion:        PrivacyRegion(ws.PrivacyRegion),
	}
	return &workspace, nil
}

// Workspaces returns the workspaces of the account.
func (this *Account) Workspaces() []*Workspace {
	this.apis.mustBeOpen()
	workspaces := this.account.Workspaces()
	infos := make([]*Workspace, len(workspaces))
	for i, ws := range workspaces {
		workspace := Workspace{
			apis:                 this.apis,
			account:              this,
			store:                this.apis.datastore.Store(ws.ID),
			workspace:            ws,
			ID:                   ws.ID,
			Name:                 ws.Name,
			Identifiers:          ws.Identifiers,
			AnonymousIdentifiers: AnonymousIdentifiers(ws.AnonymousIdentifiers),
			PrivacyRegion:        PrivacyRegion(ws.PrivacyRegion),
		}
		infos[i] = &workspace
	}
	return infos
}

var bigMaxInt32 = big.NewInt(math.MaxInt32)

// generateRandomID generates a random identifier in [1, maxInt32].
func generateRandomID() (int, error) {
	n, err := rand.Int(rand.Reader, bigMaxInt32)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}
