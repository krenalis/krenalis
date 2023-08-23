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
	"net/http"
	"os"
	"slices"
	"sort"
	"sync"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/httpclient"
	"chichi/apis/mappings/mapexp"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector/types"
	"chichi/telemetry"

	"golang.org/x/crypto/bcrypt"
)

type APIs struct {
	db             *postgres.DB
	state          *state.State
	datastore      *datastore.Datastore
	http           *httpclient.HTTP
	events         *events.Events
	scheduler      *scheduler
	schedulerMu    sync.Mutex
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

type ExpressionToBeExtracted struct {
	Value    string
	Type     types.Type
	Nullable bool
}

// New returns an *APIs instance. It can only be called once.
func New(conf *Config) (*APIs, error) {

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
		return nil, fmt.Errorf("cannot connect to PostgreSQL: %s", err)
	}

	apis := &APIs{db: db}

	// Set the HTTP client.
	apis.http = httpclient.New(db, apis.state, http.DefaultTransport)
	apis.http.SetTrace(os.Stdout)

	// Instantiate the state.
	apis.state, err = state.New(db)
	if err != nil {
		return nil, err
	}

	// Listen to state changes.
	apis.state.AddListener(apis.onElectLeader)
	apis.state.AddListener(apis.onExecuteAction)

	// Load the state.
	err = apis.state.Load()
	if err != nil {
		return nil, err
	}

	// Init the datastore.
	apis.datastore = datastore.New(apis.state)

	apis.events, err = events.New(db, apis.state, apis.datastore, apis.http)
	if err != nil {
		return nil, err
	}

	return apis, nil
}

// Account returns the account with identifier id.
//
// It returns an errors.NoFound error if the account does not exist.
func (apis *APIs) Account(ctx context.Context, id int) (*Account, error) {
	_, t := telemetry.TraceSpan(ctx, "apis.Account", "account_id", id)
	defer t.End()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid account identifier", id)
	}
	acc, ok := apis.state.Account(id)
	if !ok {
		return nil, errors.NotFound("account %d does not exist", id)
	}
	account := Account{
		db:            apis.db,
		datastore:     apis.datastore,
		eventObserver: apis.events.Observer(),
		state:         apis.state,
		http:          apis.http,
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
func (apis *APIs) Accounts(ctx context.Context, order AccountSort, first, limit int) ([]*Account, error) {
	_, s := telemetry.TraceSpan(ctx, "apis.Connectors")
	defer s.End()
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
func (apis *APIs) AuthenticateAccount(ctx context.Context, email, password string) (int, error) {
	_, t := telemetry.TraceSpan(ctx, "apis.Connectors")
	defer t.End()
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

// Close closes the APIs.
func (apis *APIs) Close() {
	// Close scheduler.
	apis.schedulerMu.Lock()
	if apis.scheduler != nil {
		apis.scheduler.Close()
		apis.scheduler = nil
	}
	apis.schedulerMu.Unlock()
	// Close events, datastore and state.
	apis.events.Close()
	apis.datastore.Close()
	apis.state.Close()
}

// Connector returns the connector with identifier id.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (apis *APIs) Connector(ctx context.Context, id int) (*Connector, error) {
	_, t := telemetry.TraceSpan(ctx, "apis.Connector", "id", id)
	defer t.End()
	c, ok := apis.state.Connector(id)
	if !ok {
		return nil, errors.NotFound("connector %d does not exist", id)
	}
	connector := Connector{
		connector:              c,
		http:                   apis.http,
		ID:                     c.ID,
		Name:                   c.Name,
		SourceDescription:      c.SourceDescription,
		DestinationDescription: c.DestinationDescription,
		Type:                   ConnectorType(c.Type),
		HasSheets:              c.HasSheets,
		HasSettings:            c.HasSettings,
		Icon:                   c.Icon,
		FileExtension:          c.FileExtension,
		SampleQuery:            c.SampleQuery,
		WebhooksPer:            WebhooksPer(c.WebhooksPer),
		OAuth:                  c.OAuth != nil,
	}
	return &connector, nil
}

// Connectors returns the collectors.
func (apis *APIs) Connectors(ctx context.Context) []*Connector {
	_, s := telemetry.TraceSpan(ctx, "apis.Connectors")
	defer s.End()
	cc := apis.state.Connectors()
	connectors := make([]*Connector, len(cc))
	for i, c := range cc {
		connector := Connector{
			connector:              c,
			http:                   apis.http,
			ID:                     c.ID,
			Name:                   c.Name,
			SourceDescription:      c.SourceDescription,
			DestinationDescription: c.DestinationDescription,
			Type:                   ConnectorType(c.Type),
			HasSheets:              c.HasSheets,
			HasSettings:            c.HasSettings,
			Icon:                   c.Icon,
			FileExtension:          c.FileExtension,
			SampleQuery:            c.SampleQuery,
			WebhooksPer:            WebhooksPer(c.WebhooksPer),
			OAuth:                  c.OAuth != nil,
		}
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
func (apis *APIs) CreateAccount(ctx context.Context, email, password string) (int, error) {
	_, t := telemetry.TraceSpan(ctx, "apis.CreateAccount")
	defer t.End()
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
func (apis *APIs) CountAccounts(ctx context.Context) int {
	_, s := telemetry.TraceSpan(ctx, "apis.CountAccounts")
	defer s.End()
	return len(apis.state.Accounts())
}

// onElectLeader is called when a leader is elected.
func (apis *APIs) onElectLeader(n state.ElectLeader) {
	if apis.state.IsLeader() {
		apis.schedulerMu.Lock()
		apis.scheduler = newScheduler(apis.db, apis.state, apis.datastore, apis.http)
		apis.schedulerMu.Unlock()
		return
	}
	apis.schedulerMu.Lock()
	if apis.scheduler != nil {
		apis.scheduler.Close()
		apis.scheduler = nil
	}
	apis.schedulerMu.Unlock()
}

// onExecuteAction is called when an action is executed.
func (apis *APIs) onExecuteAction(n state.ExecuteAction) {
	if !apis.state.IsLeader() {
		return
	}
	action, _ := apis.state.Action(n.Action)
	c := action.Connection()
	store := apis.datastore.Store(c.Workspace().ID)
	connection := &Connection{db: apis.db, connection: c, store: store, http: apis.http}
	a := &Action{db: apis.db, action: action, connection: connection}
	go a.exec()
}

// validateExpression validates an expression.
func (apis *APIs) validateExpression(expression string, schema types.Type, dtType types.Type, dtNullable bool) string {
	_, err := mapexp.Compile(expression, schema, dtType, dtNullable)
	if err != nil {
		return err.Error()
	}
	return ""
}

// expressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (apis *APIs) expressionsProperties(expressions []ExpressionToBeExtracted, schema types.Type) ([]string, error) {
	var properties []types.Path
	for _, expression := range expressions {
		exp, err := mapexp.Compile(expression.Value, schema, expression.Type, expression.Nullable)
		if err != nil {
			return nil, err
		}
		expressionProperties := exp.Properties()
		properties = append(properties, expressionProperties...)
	}
	// Remove duplicated properties.
	m := map[string]types.Path{}
	for _, prop := range properties {
		m[prop.String()] = prop
	}
	uniqueProperties := []string{}
	for s := range m {
		uniqueProperties = append(uniqueProperties, s)
	}
	return uniqueProperties, nil
}

// Workspace represents a workspace.
type Workspace struct {
	account              *Account
	db                   *postgres.DB
	state                *state.State
	store                *datastore.Store
	http                 *httpclient.HTTP
	eventObserver        *events.Observer
	workspace            *state.Workspace
	ID                   int
	Name                 string
	AnonymousIdentifiers AnonymousIdentifiers
	PrivacyRegion        PrivacyRegion
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

// AnonymousIdentifiers represents the anonymous identifiers of a workspace.
type AnonymousIdentifiers struct {
	Priority []string
	Mapping  map[string]string
}

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)
