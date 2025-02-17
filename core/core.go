//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/events/collector"
	"github.com/meergo/meergo/core/events/dispatcher"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/core/transformers/lambda"
	"github.com/meergo/meergo/core/transformers/local"
	"github.com/meergo/meergo/core/transformers/mappings"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// validationError is the interface implemented by validation errors.
type validationError interface {
	error
	PropertyPath() string
}

type Core struct {
	db         *postgres.DB
	state      *state.State
	datastore  *datastore.Datastore
	connectors *connectors.Connectors
	metrics    *metrics.Collector
	events     struct {
		collector      *collector.Collector
		observer       *collector.Observer
		dispatcher     *dispatcher.Dispatcher
		operationStore events.OperationStore
	}
	transformerProvider transformers.Provider
	actionCleaner       *actionCleaner
	actionScheduler     *actionScheduler
	smtp                *SMTPConfig
	close               struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		sync.WaitGroup
	}
	closed atomic.Bool
}

var hasBeenCalled bool

type Config struct {
	PostgreSQL      PostgreSQLConfig
	Transformer     any // must be a LambdaConfig or LocalConfig value
	SMTP            SMTPConfig
	ConnectorsOAuth map[string]*state.ConnectorOAuth
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

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type ExpressionToBeExtracted struct {
	Value string     `json:"value"`
	Type  types.Type `json:"type"`
}

// New returns a *Core instance. It can only be called once.
func New(conf *Config) (*Core, error) {

	if hasBeenCalled {
		return nil, errors.New("core.New has already been called")
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

	// Ping the PostgreSQL connection to test if it works.
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = db.Ping(pingCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to PostgreSQL: %s", err)
	}

	var smtp *SMTPConfig
	if conf.SMTP.Host != "" {
		smtp = &conf.SMTP
	}

	core := &Core{db: db, smtp: smtp}

	// Create a transformer.
	switch c := conf.Transformer.(type) {
	case LambdaConfig:
		core.transformerProvider = lambda.New(lambda.Settings(c))
	case LocalConfig:
		core.transformerProvider = local.New(local.Settings(c))
	case nil:
	default:
		return nil, errors.New("invalid transformer")
	}

	// Instantiate the state.
	core.state, err = state.New(db, conf.ConnectorsOAuth)
	if err != nil {
		return nil, err
	}

	// Init the event operation store.
	core.events.operationStore = events.NewPostgreStore(db)

	// Init the metrics.
	core.metrics = metrics.New(db, core.state)

	// Init the datastore.
	core.datastore = datastore.New(core.state)

	// Init the connectors.
	core.connectors = connectors.New(db, core.state)

	// Init the events.
	core.events.dispatcher, err = dispatcher.New(db, core.state, core.events.operationStore, core.transformerProvider, core.connectors, core.metrics)
	if err != nil {
		core.datastore.Close()
		core.state.Close()
		return nil, err
	}
	core.events.collector, err = collector.New(db, core.state, core.datastore, core.events.operationStore,
		core.transformerProvider, core.events.dispatcher, core.metrics)
	if err != nil {
		core.events.dispatcher.Close()
		core.datastore.Close()
		core.state.Close()
		return nil, err
	}
	core.events.observer = core.events.collector.Observer()

	// Create the action cleaner.
	core.actionCleaner = newActionCleaner(core.state, core.datastore)

	// Create the action scheduler.
	core.actionScheduler = newActionScheduler(core)

	core.close.ctx, core.close.cancelCtx = context.WithCancel(context.Background())

	// Listen to state changes.
	core.state.Freeze()
	core.state.AddListener(core.onDeleteAction)
	core.state.AddListener(core.onExecuteAction)
	core.state.Unfreeze()

	return core, nil
}

// AcceptInvitation accepts the invitation with the given invitation token. It
// sets the member's name and password and removes its token. name's length must
// be in range [1, 60]. password's length must be at least 8 character long.
//
// If an invitation with the given token does not exist, it returns a
// NotFoundError error. If the token is expired it returns an
// error.UnprocessableError error with code InvitationTokenExpired.
func (core *Core) AcceptInvitation(ctx context.Context, token string, name string, password string) error {
	core.mustBeOpen()
	if !isValidInvitationToken(token) {
		return errors.NotFound("invitation token %q does not exist", token)
	}
	m := MemberToSet{
		Name:     name,
		Password: password,
	}
	err := validateMemberToSet(m, false, true)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = core.state.Transaction(ctx, func(tx *state.Tx) error {
		var id int
		var createdAt time.Time
		err := core.db.QueryRow(ctx, "SELECT id, created_at FROM members WHERE invitation_token = $1", token).Scan(&id, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("invitation token %q does not exist", token)
			}
			return err
		}
		if isInvitationTokenExpired(createdAt) {
			return errors.Unprocessable(InvitationTokenExpired, "invitation token is expired")
		}
		_, err = core.db.Exec(ctx, "UPDATE members SET name = $1, password = $2, invitation_token = '' WHERE id = $3",
			name, string(pass), id)
		return err
	})
	return err
}

// AddOrganization adds a new organization and returns its identifier.
// name cannot be empty and cannot be longer than 45 runes.
func (core *Core) AddOrganization(ctx context.Context, name string) (int, error) {
	core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 45); err != nil {
		return 0, errors.BadRequest("%s", err)
	}
	var id int
	err := core.db.QueryRow(ctx, "INSERT INTO organizations (name) VALUES ($1)").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// APIKey returns the organization and workspace identifiers associated with the
// provided API key token. If the API key is not restricted to a workspace, the
// workspace identifier will be 0. The boolean return value indicates whether
// the token exists.
func (core *Core) APIKey(token string) (int, int, bool) {
	key, ok := core.state.APIKeyByToken(token)
	if !ok {
		return 0, 0, false
	}
	return key.Organization, key.Workspace, true
}

// Close closes the Core. When Close is called, no other calls to Core's methods
// should be in progress and no other shall be made.
// It panics if it has already been called.
func (core *Core) Close() {
	if core.closed.Swap(true) {
		panic("core already closed")
	}
	// Cancel the execution of actions initiated via API.
	core.close.cancelCtx()
	// Close the action scheduler.
	core.actionScheduler.Close()
	// Wait for the completion of actions initiated via API.
	core.close.Wait()
	// Close the action cleaner.
	core.actionCleaner.Close(context.Background())
	// Close event collector, event dispatcher, metrics, datastore, and state.
	core.events.collector.Close()
	core.events.dispatcher.Close()
	core.metrics.Close(context.Background())
	core.datastore.Close()
	core.state.Close()
}

// Connector returns the connector with the provided name.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (core *Core) Connector(name string) (*Connector, error) {
	core.mustBeOpen()
	c, ok := core.state.Connector(name)
	if !ok {
		return nil, errors.NotFound("connector %q does not exist", name)
	}
	connector := Connector{
		core:            core,
		connector:       c,
		Name:            c.Name,
		Type:            ConnectorType(c.Type),
		IdentityIDLabel: c.IdentityIDLabel,
		HasSheets:       c.HasSheets,
		FileExtension:   c.FileExtension,
		RequiresAuth:    c.OAuth != nil,
		TermForUsers:    c.TermForUsers,
		TermForGroups:   c.TermForGroups,
		Icon:            c.Icon,
	}
	if c.SourceTargets != 0 {
		connector.AsSource = &SourceConnector{
			Description: c.SourceDescription,
			Targets:     stateToCoreTargets(c.SourceTargets),
			HasSettings: c.HasSourceSettings,
			SampleQuery: c.SampleQuery,
			WebhooksPer: WebhooksPer(c.WebhooksPer),
		}
	}
	if c.DestinationTargets != 0 {
		connector.AsDestination = &DestinationConnector{
			Description: c.DestinationDescription,
			Targets:     stateToCoreTargets(c.DestinationTargets),
			HasSettings: c.HasDestinationSettings,
			SendingMode: (*SendingMode)(c.SendingMode),
		}
	}
	if connector.TermForUsers == "" {
		connector.TermForUsers = "users"
	}
	if connector.TermForGroups == "" {
		connector.TermForGroups = "groups"
	}
	return &connector, nil
}

// Connectors returns the connectors.
func (core *Core) Connectors() []*Connector {
	core.mustBeOpen()
	cc := core.state.Connectors()
	connectors := make([]*Connector, len(cc))
	for i, c := range cc {
		connector := Connector{
			core:            core,
			connector:       c,
			Name:            c.Name,
			Type:            ConnectorType(c.Type),
			IdentityIDLabel: c.IdentityIDLabel,
			HasSheets:       c.HasSheets,
			FileExtension:   c.FileExtension,
			RequiresAuth:    c.OAuth != nil,
			TermForUsers:    c.TermForUsers,
			TermForGroups:   c.TermForGroups,
			Icon:            c.Icon,
		}
		if c.SourceTargets != 0 {
			connector.AsSource = &SourceConnector{
				Description: c.SourceDescription,
				Targets:     stateToCoreTargets(c.SourceTargets),
				HasSettings: c.HasSourceSettings,
				SampleQuery: c.SampleQuery,
				WebhooksPer: WebhooksPer(c.WebhooksPer),
			}
		}
		if c.DestinationTargets != 0 {
			connector.AsDestination = &DestinationConnector{
				Description: c.DestinationDescription,
				Targets:     stateToCoreTargets(c.DestinationTargets),
				HasSettings: c.HasDestinationSettings,
				SendingMode: (*SendingMode)(c.SendingMode),
			}
		}
		if connector.TermForUsers == "" {
			connector.TermForUsers = "users"
		}
		if connector.TermForGroups == "" {
			connector.TermForGroups = "groups"
		}
		connectors[i] = &connector
	}
	slices.SortFunc(connectors, func(a, b *Connector) int {
		if a.Name < b.Name {
			return -1
		}
		return 1
	})
	return connectors
}

// CountOrganizations returns the total number of organizations.
func (core *Core) CountOrganizations(ctx context.Context) int {
	core.mustBeOpen()
	return len(core.state.Organizations())
}

// ExpressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (core *Core) ExpressionsProperties(expressions []ExpressionToBeExtracted, schema types.Type) ([]string, error) {
	core.mustBeOpen()
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.BadRequest("schema is non an object")
	}
	var properties []string
	for _, expression := range expressions {
		if expression.Value == "" {
			return nil, errors.BadRequest("expression value is empty")
		}
		if !expression.Type.Valid() {
			return nil, errors.BadRequest("expression type is not valid")
		}
		exp, err := mappings.Compile(expression.Value, schema, expression.Type, nil)
		if err != nil {
			return nil, errors.BadRequest("expression is not valid: %w", err)
		}
		expressionProperties := exp.Properties()
		properties = append(properties, expressionProperties...)
	}
	// Remove duplicated properties.
	m := map[string]string{}
	for _, prop := range properties {
		m[prop] = prop
	}
	uniqueProperties := []string{}
	for s := range m {
		uniqueProperties = append(uniqueProperties, s)
	}
	return uniqueProperties, nil
}

// MemberInvitation returns the organization's name and email of the member
// invited with the given invitation token. If an invitation with the given
// token does not exist, it returns a NotFoundError error. If the token is
// expired it returns an errors.UnprocessableError error with code
// InvitationTokenExpired.
func (core *Core) MemberInvitation(ctx context.Context, token string) (string, string, error) {
	core.mustBeOpen()
	if !isValidInvitationToken(token) {
		return "", "", errors.NotFound("invitation token %q does not exist", token)
	}
	var organizationID int
	var email string
	var createdAt time.Time
	err := core.db.QueryRow(ctx, "SELECT organization, email, created_at FROM members WHERE invitation_token = $1", token).Scan(&organizationID, &email, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", errors.NotFound("invitation token %q does not exist", token)
		}
		return "", "", err
	}
	if isInvitationTokenExpired(createdAt) {
		return "", "", errors.Unprocessable(InvitationTokenExpired, "invitation token is expired")
	}
	organization, ok := core.state.Organization(organizationID)
	if !ok {
		return "", "", errors.NotFound("invitation token %q does not exist", token)
	}
	return organization.Name, email, nil
}

// Organization returns the organization with identifier id.
//
// It returns an errors.NotFound error if the organization does not exist.
func (core *Core) Organization(ctx context.Context, id int) (*Organization, error) {
	core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid organization identifier", id)
	}
	org, ok := core.state.Organization(id)
	if !ok {
		return nil, errors.NotFound("organization %d does not exist", id)
	}
	organization := Organization{
		core:         core,
		organization: org,
		ID:           org.ID,
		Name:         org.Name,
	}
	return &organization, nil
}

// Organizations returns the organizations, in the given order, describing all
// organizations but starting from first and up to limit. first must be >= 0 and
// limit must be > 0.
func (core *Core) Organizations(ctx context.Context, order OrganizationSort, first, limit int) ([]*Organization, error) {
	core.mustBeOpen()
	if order != SortByName {
		return nil, errors.BadRequest("order %d is not valid", int(order))
	}
	if limit <= 0 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}
	if first < 0 {
		return nil, errors.BadRequest("first %d is not valid", first)
	}
	organizations := core.state.Organizations()
	count := len(organizations)
	if first >= count {
		return []*Organization{}, nil
	}
	if first+limit > count {
		limit = count - first
	}
	sort.Slice(organizations, func(i, j int) bool {
		a, b := organizations[i], organizations[j]
		switch order {
		case SortByName:
			return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
		}
		return false
	})
	organizations = organizations[first : first+limit]
	orgs := make([]*Organization, len(organizations))
	for i, organization := range organizations {
		orgs[i] = &Organization{
			ID:   organization.ID,
			Name: organization.Name,
		}
	}
	return orgs, nil
}

// ServeEvents serves the events sent via HTTP.
func (core *Core) ServeEvents(w http.ResponseWriter, r *http.Request) {
	core.mustBeOpen()
	core.events.collector.ServeHTTP(w, r)
}

// DataTransformation represents transformation passed to (*Core).TransformData
// and (*Connection).PreviewSendEvent methods.
type DataTransformation struct {
	Mapping  map[string]string           `json:"mapping,format:emitnull"`
	Function *DataTransformationFunction `json:"function"`
}

// DataTransformationFunction represents transformation function passed to
// (*Core).TransformData and (*Connection).PreviewSendEvent methods.
type DataTransformationFunction struct {
	Source       string   `json:"source"`
	Language     Language `json:"language"`
	PreserveJSON bool     `json:"preserveJSON"`
}

// Purpose represents the purpose of a data transformation.
// It can be "Create" or "Update".
type Purpose string

const (
	Create Purpose = "Create"
	Update Purpose = "Update"
)

// TransformData transforms data using a mapping or a function transformation
// and returns the transformed data. inSchema is the schema of data, and
// outSchema is the schema of the transformed data. Only one of mapping and
// transformation must be non-nil. purpose indicates the purpose of the
// transformation and can be either "Create" or "Update".
//
// It returns an errors.UnprocessableError error with code:
//   - TransformationFailed if the transformation fails due to an error in the
//     executed function.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (core *Core) TransformData(ctx context.Context, data []byte, inSchema, outSchema types.Type, transformation DataTransformation, purpose Purpose) (json.Value, error) {

	core.mustBeOpen()

	// Validate the parameters.
	if !inSchema.Valid() {
		return nil, errors.BadRequest("input schema is not valid")
	}
	if !outSchema.Valid() {
		return nil, errors.BadRequest("output schema is not valid")
	}
	if purpose != Create && purpose != Update {
		return nil, errors.BadRequest(`purpose must be "Create" or "Update"`)
	}
	if transformation.Mapping != nil && transformation.Function != nil {
		return nil, errors.BadRequest("mapping and function transformations cannot both be present")
	}

	action := &state.Action{
		InSchema:  inSchema,
		OutSchema: outSchema,
		Transformation: state.Transformation{
			Mapping: transformation.Mapping,
		},
	}

	// provider is a temporary function transformer provider.
	var provider transformers.Provider

	// Validate the mapping and the transformation.
	switch {
	case transformation.Mapping != nil:
		mapping, err := mappings.New(transformation.Mapping, inSchema, outSchema, false, nil)
		if err != nil {
			return nil, errors.BadRequest("mapping is not valid: %s", err)
		}
		action.Transformation.InPaths = mapping.InPaths()
		action.Transformation.OutPaths = mapping.OutPaths()
	case transformation.Function != nil:
		if transformation.Function.Source == "" {
			return nil, errors.BadRequest("transformation source is empty")
		}
		switch transformation.Function.Language {
		case "JavaScript":
			if core.transformerProvider == nil || !core.transformerProvider.SupportLanguage(state.JavaScript) {
				return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript transformation language  is not supported")
			}
		case "Python":
			if core.transformerProvider == nil || !core.transformerProvider.SupportLanguage(state.Python) {
				return nil, errors.Unprocessable(UnsupportedLanguage, "Python transformation language is not supported")
			}
		case "":
			return nil, errors.BadRequest("transformation language is empty")
		default:
			return nil, errors.BadRequest("transformation language %q is not valid", transformation.Function.Language)
		}
		action.Transformation.Function = &state.TransformationFunction{
			Source:  transformation.Function.Source,
			Version: "1", // no matter the version, it will be overwritten by the temporary transformation.
		}
		name := "temp-" + uuid.NewString()
		switch transformation.Function.Language {
		case "JavaScript":
			name += ".js"
			action.Transformation.Function.Language = state.JavaScript
		case "Python":
			name += ".py"
			action.Transformation.Function.Language = state.Python
		}
		action.Transformation.Function.PreserveJSON = transformation.Function.PreserveJSON
		action.Transformation.InPaths = types.PropertyNames(action.InSchema)
		action.Transformation.OutPaths = types.PropertyNames(action.OutSchema)
		provider = newTempTransformerProvider(name, transformation.Function.Source, core.transformerProvider)
	default:
		return nil, errors.BadRequest("mapping (or transformation) is required")
	}

	properties, err := types.Decode[map[string]any](bytes.NewReader(data), inSchema)
	if err != nil {
		return nil, errors.BadRequest("data does not validate against the input schema: %w", err)
	}

	// Transform the properties.
	transformer, err := transformers.New(action, provider, nil)
	if err != nil {
		return nil, err
	}
	records := []transformers.Record{
		{Purpose: transformers.Create, Properties: properties},
	}
	if purpose == "Update" {
		records[0].Purpose = transformers.Update
	}
	err = transformer.Transform(ctx, records)
	if err != nil {
		if err, ok := err.(transformers.FunctionExecutionError); ok {
			return nil, errors.Unprocessable(TransformationFailed, "%w", err)
		}
		return nil, err
	}
	if err = records[0].Err; err != nil {
		return nil, errors.Unprocessable(TransformationFailed, "%w", err)
	}

	return types.Marshal(records[0].Properties, outSchema)
}

// TransformationLanguages returns the supported transformation languages.
// Possible returned languages are "JavaScript" and "Python".
func (core *Core) TransformationLanguages() []string {
	if core.transformerProvider == nil {
		return []string{}
	}
	languages := make([]string, 0, 2)
	if core.transformerProvider.SupportLanguage(state.JavaScript) {
		languages = append(languages, "JavaScript")
	}
	if core.transformerProvider.SupportLanguage(state.Python) {
		languages = append(languages, "Python")
	}
	return languages
}

// ValidateExpression validates an expression. properties represents the allowed
// properties in the expression, and typ is the type of the expression.
//
// The returned string explains why the expression is not valid. It is empty if
// the expression is valid.
func (core *Core) ValidateExpression(expression string, properties []types.Property, typ types.Type) (string, error) {
	core.mustBeOpen()
	schema, err := types.ObjectOf(properties)
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}
	_, err = mappings.Compile(expression, schema, typ, nil)
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}

// WarehouseType represents a warehouse type.
type WarehouseType struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// WarehouseTypes returns the warehouse types.
func (core *Core) WarehouseTypes() []WarehouseType {
	core.mustBeOpen()
	types := core.state.WarehouseTypes()
	warehouseTypes := make([]WarehouseType, len(types))
	for i, t := range types {
		warehouseTypes[i] = WarehouseType{
			Name: t.Name,
			Icon: t.Icon,
		}
	}
	return warehouseTypes
}

// mustBeOpen panics if core has been closed.
func (core *Core) mustBeOpen() {
	if core.closed.Load() {
		panic("core is closed")
	}
}

// onDeleteAction is called when an action is deleted.
func (core *Core) onDeleteAction(n state.DeleteAction) {
	if core.state.IsLeader() && core.transformerProvider != nil {
		go func() {
			for _, language := range [...]state.Language{state.JavaScript, state.Python} {
				if core.transformerProvider.SupportLanguage(language) {
					name := util.TransformationFunctionName(n.ID, language)
					err := core.transformerProvider.Delete(core.close.ctx, name)
					if err != nil {
						slog.Debug("cannot delete transformer function", "name", name, "err", err)
					}
				}
			}
		}()
	}
}

// onExecuteAction is called when an action is executed.
func (core *Core) onExecuteAction(n state.ExecuteAction) {
	if !core.state.IsLeader() {
		return
	}
	action, _ := core.state.Action(n.Action)
	c := action.Connection()
	store := core.datastore.Store(c.Workspace().ID)
	connection := &Connection{core: core, store: store, connection: c}
	a := &Action{core: core, action: action, connection: connection}
	core.close.Add(1)
	go func() {
		defer core.close.Done()
		a.exec(core.close.ctx)
	}()
}

func isValidInvitationToken(token string) bool {
	if len(token) != 44 {
		return false
	}
	_, err := base64.URLEncoding.DecodeString(token)
	return err == nil
}

func stateToCoreTargets(targets state.ConnectorTargets) []Target {
	ts := []Target{}
	if targets.Contains(state.Users) {
		ts = append(ts, Users)
	}
	if targets.Contains(state.Groups) {
		ts = append(ts, Groups)
	}
	if targets.Contains(state.Events) {
		ts = append(ts, Events)
	}
	return ts
}

type OrganizationSort int

const (
	SortByName OrganizationSort = iota
)

func (s OrganizationSort) String() string {
	switch s {
	case SortByName:
		return "name"
	}
	panic("invalid organization sort")
}
