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
	"chichi/apis/events"
	"chichi/apis/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"

	"github.com/redis/go-redis/v9"
)

var AuthenticationFailed errors.Code = "AuthenticationFailed"

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// Account represents an account.
type Account struct {
	db            *postgres.DB
	redis         *redis.Client
	eventObserver *events.Observer
	state         *state.State
	http          *httpclient.HTTP
	account       *state.Account
	ID            int
	Name          string
	Email         string
	InternalIPs   []string
}

// Warehouse represents data warehouse settings. It is used with AddWorkspace.
type Warehouse struct {
	Type     WarehouseType
	Settings []byte
}

// AddWorkspace adds a workspace with the given name and data warehouse, if not
// nil, and returns the identifier. name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the account does not exist anymore.
// It returns an errors.UnprocessableError error with code
//   - ConnectionFailed, if the connection to the data warehouse fails.
//   - InvalidSettings, if the warehouse settings are not valid.
func (this *Account) AddWorkspace(name string, warehouse *Warehouse) (int, error) {

	if name == "" || utf8.RuneCountInString(name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", name)
	}

	if warehouse != nil {
		warehouse, err := openWarehouse(warehouse.Type, warehouse.Settings)
		if err != nil {
			return 0, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
		}
		err = warehouse.Ping(context.Background())
		if err != nil {
			return 0, errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
		}
	}

	n := state.AddWorkspaceNotification{
		Account: this.account.ID,
		Name:    name,
	}
	if warehouse != nil {
		n.Warehouse.Type = state.WarehouseType(warehouse.Type)
		n.Warehouse.Settings = warehouse.Settings
	}

	// Generate the identifier.
	var err error
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var err error
		if n.Warehouse.Settings == nil {
			_, err = tx.Exec(ctx, "INSERT INTO workspaces (id, account, name) VALUES ($1, $2, $3)", n.ID, n.Account, n.Name)
		} else {
			_, err = tx.Exec(ctx, "INSERT INTO workspaces (id, account, name, warehouse_type, warehouse_settings)"+
				" VALUES ($1, $2, $3, $4, $5)", n.ID, n.Account, n.Name, n.Warehouse.Type, string(n.Warehouse.Settings))
		}
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "workspaces_keys_account_fkey" {
					return errors.NotFound("account %d does not exist", n.Account)
				}
			}
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
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid workspace identifier", id)
	}
	ws, ok := this.account.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	workspace := Workspace{
		db:                   this.db,
		redis:                this.redis,
		state:                this.state,
		http:                 this.http,
		eventObserver:        this.eventObserver,
		workspace:            ws,
		ID:                   ws.ID,
		Name:                 ws.Name,
		AnonymousIdentifiers: AnonymousIdentifiers(ws.AnonymousIdentifiers),
		PrivacyRegion:        PrivacyRegion(ws.PrivacyRegion),
	}
	return &workspace, nil
}

// Workspaces returns the workspaces of the account.
func (this *Account) Workspaces() []*Workspace {
	workspaces := this.account.Workspaces()
	infos := make([]*Workspace, len(workspaces))
	for i, ws := range workspaces {
		workspace := Workspace{
			db:                   this.db,
			redis:                this.redis,
			state:                this.state,
			http:                 this.http,
			eventObserver:        this.eventObserver,
			workspace:            ws,
			ID:                   ws.ID,
			Name:                 ws.Name,
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
