//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"chichi/apis/connectors"
	"chichi/apis/datastore"
	"chichi/apis/encoding"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/apis/transformers/lambda"
	"chichi/apis/transformers/local"
	"chichi/apis/transformers/mappings"
	"chichi/connector/types"
	"chichi/telemetry"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const TransformationFailed errors.Code = "TransformationFailed"

type APIs struct {
	db                  *postgres.DB
	state               *state.State
	datastore           *datastore.Datastore
	connectors          *connectors.Connectors
	events              *events.Events
	functionTransformer transformers.Function
	mu                  sync.Mutex // for the scheduler field
	scheduler           *scheduler
	eventProcessor      *events.Processor
	close               struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		sync.WaitGroup
	}
	closed atomic.Bool
}

var hasBeenCalled bool

type Config struct {
	PostgreSQL  PostgreSQLConfig
	Transformer any // must be a LambdaConfig or LocalConfig value
}

type PostgreSQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

type LambdaConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Role            string
	Node            struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeExecutable   string
	PythonExecutable string
	FunctionsDir     string
}

type ExpressionToBeExtracted struct {
	Value string
	Type  types.Type
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

	// Create a transformer.
	switch c := conf.Transformer.(type) {
	case LambdaConfig:
		apis.functionTransformer = lambda.New(lambda.Settings(c))
	case LocalConfig:
		apis.functionTransformer = local.New(local.Settings(c))
	case nil:
	default:
		return nil, errors.New("invalid transformer")
	}

	// Instantiate the state.
	apis.state, err = state.New(db)
	if err != nil {
		return nil, err
	}

	// Listen to state changes.
	apis.state.AddListener(apis.onDeleteAction)
	apis.state.AddListener(apis.onElectLeader)
	apis.state.AddListener(apis.onExecuteAction)

	// Load the state.
	err = apis.state.Load()
	if err != nil {
		return nil, err
	}

	// Init the datastore.
	apis.datastore = datastore.New(apis.state)

	// Init the connectors.
	apis.connectors = connectors.New(db, apis.state)

	apis.events, err = events.New(db, apis.state, apis.datastore, apis.functionTransformer, apis.connectors)
	if err != nil {
		apis.datastore.Close()
		apis.state.Close()
		return nil, err
	}

	apis.close.ctx, apis.close.cancelCtx = context.WithCancel(context.Background())

	return apis, nil
}

// Account returns the account with identifier id.
//
// It returns an errors.NoFound error if the account does not exist.
func (apis *APIs) Account(ctx context.Context, id int) (*Account, error) {
	apis.mustBeOpen()
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
		apis:        apis,
		account:     acc,
		ID:          acc.ID,
		Name:        acc.Name,
		Email:       acc.Email,
		InternalIPs: slices.Clone(acc.InternalIPs),
	}
	return &account, nil
}

// Accounts returns a list of Account, in the given order, describing all
// accounts but starting from first and up to limit. first must be >= 0 and
// limit must be > 0.
func (apis *APIs) Accounts(ctx context.Context, order AccountSort, first, limit int) ([]*Account, error) {
	apis.mustBeOpen()
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
	apis.mustBeOpen()
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
	err := apis.db.QueryRow(ctx, "SELECT id, password FROM accounts WHERE email = $1", email).Scan(&id, &hashedPassword)
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
// It panics if it has already been called.
func (apis *APIs) Close() {
	if apis.closed.Swap(true) {
		panic("apis already closed")
	}
	// Cancel the execution of actions initiated via API.
	apis.close.cancelCtx()
	// Close scheduler.
	apis.mu.Lock()
	if apis.scheduler != nil {
		apis.scheduler.Close()
		apis.scheduler = nil
	}
	apis.mu.Unlock()
	// Wait for the completion of actions initiated via API.
	apis.close.Wait()
	// Close events, datastore and state.
	apis.events.Close()
	apis.datastore.Close()
	apis.state.Close()
}

// Connector returns the connector with identifier id.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (apis *APIs) Connector(ctx context.Context, id int) (*Connector, error) {
	apis.mustBeOpen()
	_, t := telemetry.TraceSpan(ctx, "apis.Connector", "id", id)
	defer t.End()
	c, ok := apis.state.Connector(id)
	if !ok {
		return nil, errors.NotFound("connector %d does not exist", id)
	}
	connector := Connector{
		apis:                   apis,
		connector:              c,
		ID:                     c.ID,
		Name:                   c.Name,
		SourceDescription:      c.SourceDescription,
		DestinationDescription: c.DestinationDescription,
		TermForUsers:           c.TermForUsers,
		TermForGroups:          c.TermForGroups,
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
	apis.mustBeOpen()
	_, s := telemetry.TraceSpan(ctx, "apis.Connectors")
	defer s.End()
	cc := apis.state.Connectors()
	connectors := make([]*Connector, len(cc))
	for i, c := range cc {
		connector := Connector{
			apis:                   apis,
			connector:              c,
			ID:                     c.ID,
			Name:                   c.Name,
			SourceDescription:      c.SourceDescription,
			DestinationDescription: c.DestinationDescription,
			TermForUsers:           c.TermForUsers,
			TermForGroups:          c.TermForGroups,
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

// CountAccounts returns the total number of accounts.
func (apis *APIs) CountAccounts(ctx context.Context) int {
	apis.mustBeOpen()
	_, s := telemetry.TraceSpan(ctx, "apis.CountAccounts")
	defer s.End()
	return len(apis.state.Accounts())
}

// CreateAccount a new account given its email and password and returns its
// identifier.
func (apis *APIs) CreateAccount(ctx context.Context, email, password string) (int, error) {
	apis.mustBeOpen()
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
	err = apis.db.QueryRow(ctx, "INSERT INTO accounts (email, password) VALUES ($1, $2)",
		email, string(hashedPassword)).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, err
}

// ExpressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (apis *APIs) ExpressionsProperties(expressions []ExpressionToBeExtracted, schema types.Type) ([]string, error) {
	apis.mustBeOpen()
	var properties []types.Path
	for _, expression := range expressions {
		exp, err := mappings.Compile(expression.Value, schema, expression.Type, false, true, nil)
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

// ServeEvents serves the events sent via HTTP.
func (apis *APIs) ServeEvents(w http.ResponseWriter, r *http.Request) {
	apis.mustBeOpen()
	apis.events.ServeHTTP(w, r)
}

// TransformationLanguages returns the supported transformation languages.
// Possible returned languages are "JavaScript" and "Python".
func (apis *APIs) TransformationLanguages() []string {
	if apis.functionTransformer == nil {
		return []string{}
	}
	languages := make([]string, 0, 2)
	if apis.functionTransformer.SupportLanguage(state.JavaScript) {
		languages = append(languages, "JavaScript")
	}
	if apis.functionTransformer.SupportLanguage(state.Python) {
		languages = append(languages, "Python")
	}
	return languages
}

// TransformData transforms data using a mapping or a function transformation
// and returns the transformed data. inSchema is the schema of data, and
// outSchema is the schema of the transformed data. Only one of mapping and
// transformation must be non-nil.
//
// It returns an errors.UnprocessableError error with code:
//   - LanguageNotSupported, if the transformation language is not supported.
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
func (apis *APIs) TransformData(ctx context.Context, data []byte, inSchema, outSchema types.Type, transformation Transformation) ([]byte, error) {

	apis.mustBeOpen()

	// Validate the parameters.
	if !inSchema.Valid() {
		return nil, errors.BadRequest("input schema is not valid")
	}
	if !outSchema.Valid() {
		return nil, errors.BadRequest("output schema is not valid")
	}
	if transformation.Mapping != nil && transformation.Function != nil {
		return nil, errors.BadRequest("mapping and function transformations cannot both be present")
	}
	switch {
	case transformation.Mapping != nil:
		var inPaths []types.Path
		var outPaths []types.Path
		for path, expr := range transformation.Mapping {
			outPath, err := types.ParsePropertyPath(path)
			if err != nil {
				return nil, errors.BadRequest("output mapped property %q is not valid", path)
			}
			outPaths = append(outPaths, outPath)
			p, err := outSchema.PropertyByPath(outPath)
			if err != nil {
				err := err.(types.PathNotExistError)
				return nil, errors.BadRequest("output mapped property %s not found in output schema", err.Path)
			}
			expr, err := mappings.Compile(expr, inSchema, p.Type, p.Required, p.Nullable, nil)
			if err != nil {
				return nil, errors.BadRequest("invalid expression mapped to %s: %s", path, err)
			}
			inPaths = append(inPaths, expr.Properties()...)
		}
		if props := unmappedProperties(inSchema, inPaths); props != nil {
			return nil, errors.BadRequest("input schema contains unmapped properties: %s", strings.Join(props, ", "))
		}
		if props := unmappedProperties(outSchema, outPaths); props != nil {
			return nil, errors.BadRequest("output schema contains unmapped properties: %s", strings.Join(props, ", "))
		}
	case transformation.Function != nil:
		if transformation.Function.Source == "" {
			return nil, errors.BadRequest("transformation source is empty")
		}
		tr := apis.functionTransformer
		switch transformation.Function.Language {
		case "JavaScript":
			if tr == nil || !tr.SupportLanguage(state.JavaScript) {
				return nil, errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language  is not supported")
			}
		case "Python":
			if tr == nil || !tr.SupportLanguage(state.Python) {
				return nil, errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
			}
		case "":
			return nil, errors.BadRequest("transformation language is empty")
		default:
			return nil, errors.BadRequest("transformation language %q is not valid", transformation.Function.Language)
		}
	default:
		return nil, errors.BadRequest("mapping (or transformation) is required")
	}
	value, err := encoding.Unmarshal(bytes.NewReader(data), "data", inSchema)
	if err != nil {
		return nil, errors.BadRequest("data does not validate against the input schema: %w", err)
	}

	// Create a temporary transformer.
	var transformer transformers.Function
	var function *state.TransformationFunction
	if transformation.Function != nil {
		function = &state.TransformationFunction{
			Source:  transformation.Function.Source,
			Version: "1", // no matter the version, it will be overwritten by the temporary transformation.
		}
		name := "temp-" + uuid.NewString()
		switch transformation.Function.Language {
		case "JavaScript":
			name += ".js"
			function.Language = state.JavaScript
		case "Python":
			name += ".py"
			function.Language = state.Python
		}
		transformer = newTemporaryTransformer(name, transformation.Function.Source, apis.functionTransformer)
	}

	// Transform the data.
	action := 1 // no matter the action, it will be overwritten by the temporary transformation.
	tr := state.Transformation{
		Mapping:  transformation.Mapping,
		Function: function,
	}
	m, err := transformers.New(inSchema, outSchema, tr, action, transformer, nil)
	if err != nil {
		return nil, err
	}
	value, err = m.Transform(ctx, value)
	if err != nil {
		if err, ok := err.(transformers.FunctionExecutionError); ok {
			return nil, errors.Unprocessable(TransformationFailed, err.Error())
		}
		return nil, err
	}

	return encoding.Marshal(outSchema, value)
}

// ValidateExpression validates an expression. properties represents the allowed
// properties in the expression. typ is the type of the expression, required
// indicates whether a value for that property is required, and nullable
// indicates whether it can be nullable.
func (apis *APIs) ValidateExpression(expression string, properties []types.Property, typ types.Type, required, nullable bool) string {
	apis.mustBeOpen()
	_, err := mappings.Compile(expression, types.Object(properties), typ, required, nullable, nil)
	if err != nil {
		return err.Error()
	}
	return ""
}

// mustBeOpen panics if apis has been closed.
func (apis *APIs) mustBeOpen() {
	if apis.closed.Load() {
		panic("apis is closed")
	}
}

// onElectLeader is called when a new leader is elected.
func (apis *APIs) onElectLeader(n state.ElectLeader) {
	if apis.state.IsLeader() {
		s := newScheduler(apis)
		apis.mu.Lock()
		apis.scheduler = s
		apis.mu.Unlock()
		return
	}
	apis.mu.Lock()
	s := apis.scheduler
	apis.scheduler = nil
	apis.mu.Unlock()
	if s != nil {
		go s.Shutdown(apis.close.ctx)
	}
}

// onDeleteAction is called when an action is deleted.
func (apis *APIs) onDeleteAction(n state.DeleteAction) {
	if apis.state.IsLeader() && apis.functionTransformer != nil {
		go func() {
			for _, language := range [...]state.Language{state.JavaScript, state.Python} {
				if apis.functionTransformer.SupportLanguage(language) {
					name := transformationFunctionName(n.ID, language)
					err := apis.functionTransformer.Delete(apis.close.ctx, name)
					if err != nil {
						slog.Debug("cannot delete transformer function", "name", name, "err", err)
					}
				}
			}
		}()
	}
}

// onExecuteAction is called when an action is executed.
func (apis *APIs) onExecuteAction(n state.ExecuteAction) {
	if !apis.state.IsLeader() {
		return
	}
	action, _ := apis.state.Action(n.Action)
	c := action.Connection()
	store := apis.datastore.Store(c.Workspace().ID)
	connection := &Connection{apis: apis, store: store, connection: c}
	a := &Action{apis: apis, action: action, connection: connection}
	apis.close.Add(1)
	go func() {
		defer apis.close.Done()
		a.exec(apis.close.ctx)
	}()
}

// Workspace represents a workspace.
type Workspace struct {
	apis                 *APIs
	account              *Account
	store                *datastore.Store
	workspace            *state.Workspace
	ID                   int
	Name                 string
	Identifiers          []string
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
