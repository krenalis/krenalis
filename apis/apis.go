//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slices"
)

type APIs struct {
	db             *postgres.DB
	state          *state.State
	events         *events.Events
	scheduler      *scheduler
	eventProcessor *events.Processor
}

var hasBeenCalled bool

type Config struct {
	PostgreSQL PostgreSQLConfig
}

type PostgreSQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// New returns an API instance. It can only be called once.
func New(ctx context.Context, conf *Config) (*APIs, error) {

	if hasBeenCalled {
		return nil, errors.New("apis.New has already been called")
	}
	hasBeenCalled = true

	// Open connection to PostgreSQL.
	ps := conf.PostgreSQL
	db, err := postgres.Open(&postgres.Options{
		Host:     ps.Host,
		Port:     ps.Port,
		Username: ps.Username,
		Password: ps.Password,
		Database: ps.Database,
		Schema:   ps.Schema,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to PostreSQL: %s", err)
	}

	apis := &APIs{db: db}

	// Load the state.
	apis.state, err = state.Load(ctx, db)
	if err != nil {
		return nil, err
	}

	// Listen to state changes.
	apis.state.AddListener(apis.onAddConnection)
	apis.state.AddListener(apis.onElectLeader)
	apis.state.AddListener(apis.onExecuteAction)
	apis.state.AddListener(apis.onSetAction)

	apis.events, err = events.New(ctx, db, apis.state)

	// Keep the state updated.
	apis.state.Keep()

	return apis, nil
}

// Account returns the account with identifier id.
//
// It returns an errors.NoFound error if the account does not exist.
func (apis *APIs) Account(id int) (*Account, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid account identifier", id)
	}
	acc, ok := apis.state.Account(id)
	if !ok {
		return nil, errors.NotFound("account %d does not exist", id)
	}
	account := Account{
		db:            apis.db,
		eventObserver: apis.events.Observer(),
		state:         apis.state,
		account:       acc,
		ID:            acc.ID,
		Name:          acc.Name,
		Email:         acc.Email,
		InternalIPs:   slices.Clone(acc.InternalIPs),
	}
	return &account, nil
}

// Accounts returns a list of Account, in the given order, describing all
// accounts but starting from first and up to limit. first must be >= 0 and
// limit must be > 0.
func (apis *APIs) Accounts(order AccountSort, first, limit int) ([]*Account, error) {
	if order != SortByName && order != SortByEmail {
		return nil, errors.BadRequest("order %d is not valid", int(order))
	}
	if limit <= 0 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}
	if first < 0 {
		return nil, errors.BadRequest("first %d is not valid", first)
	}
	accounts := apis.state.Accounts()
	count := len(accounts)
	if first >= count {
		return []*Account{}, nil
	}
	if first+limit > count {
		limit = count - first
	}
	sort.Slice(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		switch order {
		case SortByName:
			return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
		case SortByEmail:
			return a.Email < b.Email || a.Email == b.Email && a.ID < b.ID
		}
		return false
	})
	accounts = accounts[first : first+limit]
	accs := make([]*Account, len(accounts))
	for i, account := range accounts {
		accs[i] = &Account{
			ID:          account.ID,
			Name:        account.Name,
			Email:       account.Email,
			InternalIPs: slices.Clone(account.InternalIPs),
		}
	}
	return accs, nil
}

// AuthenticateAccount authenticates an account given its email and password.
//
// It returns an errors.UnprocessableError error with code
// AuthenticationFailed, if the authentication fails.
func (apis *APIs) AuthenticateAccount(email, password string) (int, error) {
	if !emailRegExp.MatchString(email) {
		return 0, errors.BadRequest("email is not valid")
	}
	if len(password) < 8 {
		return 0, errors.BadRequest("password is not valid")
	}
	var id int
	var hashedPassword []byte
	err := apis.db.QueryRow(context.Background(), "SELECT id, password FROM accounts WHERE email = $1", email).Scan(&id, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.Unprocessable(AuthenticationFailed, "authentication has failed")
		}
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return 0, errors.Unprocessable(AuthenticationFailed, "authentication has failed")
	}
	return id, nil
}

// Connector returns the connector with identifier id.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (apis *APIs) Connector(id int) (*Connector, error) {
	c, ok := apis.state.Connector(id)
	if !ok {
		return nil, errors.NotFound("connector %d does not exist", id)
	}
	connector := Connector{
		connector:   c,
		ID:          c.ID,
		Name:        c.Name,
		Type:        ConnectorType(c.Type),
		HasSettings: c.HasSettings,
		LogoURL:     c.LogoURL,
		WebhooksPer: WebhooksPer(c.WebhooksPer),
	}
	if c.OAuth != nil {
		connector.OAuth = &ConnectorOAuth{}
		*connector.OAuth = ConnectorOAuth(*c.OAuth)
	}
	connector.SourceDescription = c.SourceDescription
	connector.DestinationDescription = c.DestinationDescription
	return &connector, nil
}

// Connectors returns the collectors.
func (apis *APIs) Connectors() []*Connector {
	cc := apis.state.Connectors()
	connectors := make([]*Connector, len(cc))
	for i, c := range cc {
		connector := Connector{
			connector:   c,
			ID:          c.ID,
			Name:        c.Name,
			Type:        ConnectorType(c.Type),
			HasSettings: c.HasSettings,
			LogoURL:     c.LogoURL,
			WebhooksPer: WebhooksPer(c.WebhooksPer),
		}
		if c.OAuth != nil {
			connector.OAuth = &ConnectorOAuth{}
			*connector.OAuth = ConnectorOAuth(*c.OAuth)
		}
		connector.SourceDescription = c.SourceDescription
		connector.DestinationDescription = c.DestinationDescription
		connectors[i] = &connector
	}
	sort.Slice(connectors, func(i, j int) bool {
		a, b := connectors[i], connectors[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
	})
	return connectors
}

// CreateAccount a new account given its email and password and returns its
// identifier.
func (apis *APIs) CreateAccount(email, password string) (int, error) {
	if !emailRegExp.MatchString(email) {
		return 0, errors.BadRequest("email is not valid")
	}
	if len(password) < 8 {
		return 0, errors.BadRequest("password is not valid")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	var id int
	err = apis.db.QueryRow(context.Background(), "INSERT INTO accounts (email, password) VALUES ($1, $2)",
		email, string(hashedPassword)).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, err
}

// CountAccounts returns the total number of accounts.
func (apis *APIs) CountAccounts() int {
	return len(apis.state.Accounts())
}

// onAddConnection is called when a connection is added.
func (apis *APIs) onAddConnection(n state.AddConnectionNotification) {
	if !apis.state.IsLeader() {
		return
	}
	connection, _ := apis.state.Connection(n.ID)
	connector := connection.Connector()
	if connector.Type != state.AppType {
		return
	}
	if connection.Role == state.SourceRole && connector.Targets.Contains(state.UsersTarget) {
		go apis.reloadSchema(connection)
		return
	}
}

// onElectLeader is called when a leader is elected.
func (apis *APIs) onElectLeader(n state.ElectLeaderNotification) {
	if apis.state.IsLeader() {
		apis.scheduler = newScheduler(apis.db, apis.state)
		return
	}
	if apis.scheduler != nil {
		apis.scheduler.stop()
		apis.scheduler = nil
	}
}

// onExecuteAction is called when an action is executed.
func (apis *APIs) onExecuteAction(n state.ExecuteActionNotification) {
	if !apis.state.IsLeader() {
		return
	}
	action, _ := apis.state.Action(n.Action)
	a := &Action{db: apis.db, action: action}
	go a.exec()
}

// onSetAction is called when an action is changed.
func (apis *APIs) onSetAction(n state.SetActionNotification) {
	if !apis.state.IsLeader() {
		return
	}
	if n.Query == "" {
		return
	}
	action, _ := apis.state.Action(n.ID)
	go apis.reloadSchema(action.Connection())
}

func (apis *APIs) reloadSchema(connection *state.Connection) {
	c := &Connection{db: apis.db, connection: connection}
	err := c.reloadUserSchema()
	if err != nil {
		log.Printf("[error] cannot reload user schema for connection %d: %s", c.connection.ID, err)
	}
}

// Workspace represents a workspace.
type Workspace struct {
	db            *postgres.DB
	state         *state.State
	eventObserver *events.Observer
	workspace     *state.Workspace
	ID            int
	Name          string
	PrivacyRegion PrivacyRegion
}

type AccountSort int

const (
	SortByName AccountSort = iota
	SortByEmail
)

func (s AccountSort) String() string {
	switch s {
	case SortByName:
		return "name"
	case SortByEmail:
		return "email"
	}
	panic("invalid account sort")
}

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)
