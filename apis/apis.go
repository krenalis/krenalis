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
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/encoding"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/events/collector"
	"github.com/meergo/meergo/apis/events/dispatcher"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/apis/transformers"
	"github.com/meergo/meergo/apis/transformers/lambda"
	"github.com/meergo/meergo/apis/transformers/local"
	"github.com/meergo/meergo/apis/transformers/mappings"
	"github.com/meergo/meergo/telemetry"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ValidationError is the interface implemented by validation errors.
type ValidationError interface {
	error
	PropertyPath() string
}

type APIs struct {
	db         *postgres.DB
	state      *state.State
	datastore  *datastore.Datastore
	connectors *connectors.Connectors
	statistics *statistics.Statistics
	events     struct {
		collector  *collector.Collector
		observer   *collector.Observer
		dispatcher *dispatcher.Dispatcher
	}
	transformerProvider transformers.Provider
	actionPurger        *actionPurger
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

	apis := &APIs{db: db, smtp: smtp}

	// Create a transformer.
	switch c := conf.Transformer.(type) {
	case LambdaConfig:
		apis.transformerProvider = lambda.New(lambda.Settings(c))
	case LocalConfig:
		apis.transformerProvider = local.New(local.Settings(c))
	case nil:
	default:
		return nil, errors.New("invalid transformer")
	}

	// Instantiate the state.
	apis.state, err = state.New(db, conf.ConnectorsOAuth)
	if err != nil {
		return nil, err
	}

	// Init the datastore.
	apis.datastore = datastore.New(apis.state)

	// Init the connectors.
	apis.connectors = connectors.New(db, apis.state)

	// Init the statistics.
	apis.statistics = statistics.New(db, apis.state)

	// Init the events.
	apis.events.dispatcher, err = dispatcher.New(db, apis.state, apis.transformerProvider, apis.connectors)
	if err != nil {
		apis.datastore.Close()
		apis.state.Close()
		return nil, err
	}
	apis.events.collector, err = collector.New(db, apis.state, apis.datastore, apis.transformerProvider, apis.events.dispatcher, apis.statistics)
	if err != nil {
		apis.events.dispatcher.Close()
		apis.datastore.Close()
		apis.state.Close()
		return nil, err
	}
	apis.events.observer = apis.events.collector.Observer()

	// Create the action purger.
	apis.actionPurger = newActionPurger(apis.state, apis.datastore)

	// Create the action scheduler.
	apis.actionScheduler = newActionScheduler(apis)

	apis.close.ctx, apis.close.cancelCtx = context.WithCancel(context.Background())

	// Listen to state changes.
	apis.state.Freeze()
	apis.state.AddListener(apis.onDeleteAction)
	apis.state.AddListener(apis.onExecuteAction)
	apis.state.Unfreeze()

	return apis, nil
}

// AcceptInvitation accepts the invitation with the given invitation token. It
// sets the member's name and password and removes its token. name's length must
// be in range [1, 60]. password's length must be at least 8 character long.
//
// If an invitation with the given token does not exist, it returns a
// NotFoundError error. If the token is expired it returns an
// error.UnprocessableError error with code InvitationTokenExpired.
func (apis *APIs) AcceptInvitation(ctx context.Context, token string, name string, password string) error {
	apis.mustBeOpen()
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
	err = apis.state.Transaction(ctx, func(tx *state.Tx) error {
		var id int
		var createdAt time.Time
		err := apis.db.QueryRow(ctx, "SELECT id, created_at FROM members WHERE invitation_token = $1", token).Scan(&id, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("invitation token %q does not exist", token)
			}
			return err
		}
		if isInvitationTokenExpired(createdAt) {
			return errors.Unprocessable(InvitationTokenExpired, "invitation token is expired")
		}
		_, err = apis.db.Exec(ctx, "UPDATE members SET name = $1, password = $2, invitation_token = '' WHERE id = $3",
			name, string(pass), id)
		return err
	})
	return err
}

// AddOrganization adds a new organization and returns its identifier.
// name cannot be empty and cannot be longer than 45 runes.
func (apis *APIs) AddOrganization(ctx context.Context, name string) (int, error) {
	apis.mustBeOpen()
	_, t := telemetry.TraceSpan(ctx, "apis.AddOrganization")
	if name == "" {
		return 0, errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(name) {
		return 0, errors.BadRequest("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(name); n > 45 {
		return 0, errors.BadRequest("name is longer than 45 runes")
	}
	defer t.End()
	var id int
	err := apis.db.QueryRow(ctx, "INSERT INTO organizations (name) VALUES ($1)").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Close closes the APIs. When Close is called, no other calls to API's methods
// should be in progress and no other shall be made.
// It panics if it has already been called.
func (apis *APIs) Close() {
	if apis.closed.Swap(true) {
		panic("apis already closed")
	}
	// Cancel the execution of actions initiated via API.
	apis.close.cancelCtx()
	// Close the action scheduler.
	apis.actionScheduler.Close()
	// Wait for the completion of actions initiated via API.
	apis.close.Wait()
	// Close the action purger.
	apis.actionPurger.Close(context.Background())
	// Close event dispatcher, statistics, datastore and state.
	apis.events.dispatcher.Close()
	apis.statistics.Close(context.Background())
	apis.datastore.Close()
	apis.state.Close()
}

// Connector returns the connector with the provided name.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (apis *APIs) Connector(ctx context.Context, name string) (*Connector, error) {
	apis.mustBeOpen()
	_, t := telemetry.TraceSpan(ctx, "apis.Connector", "name", name)
	defer t.End()
	c, ok := apis.state.Connector(name)
	if !ok {
		return nil, errors.NotFound("connector %q does not exist", name)
	}
	connector := Connector{
		apis:                   apis,
		connector:              c,
		Name:                   c.Name,
		SourceDescription:      c.SourceDescription,
		DestinationDescription: c.DestinationDescription,
		TermForUsers:           c.TermForUsers,
		TermForGroups:          c.TermForGroups,
		Type:                   ConnectorType(c.Type),
		SendingMode:            (*SendingMode)(c.SendingMode),
		HasSheets:              c.HasSheets,
		HasUI:                  c.HasUI,
		IdentityIDLabel:        c.IdentityIDLabel,
		Icon:                   c.Icon,
		FileExtension:          c.FileExtension,
		SampleQuery:            c.SampleQuery,
		WebhooksPer:            WebhooksPer(c.WebhooksPer),
		OAuth:                  c.OAuth != nil,
	}
	if connector.TermForUsers == "" {
		connector.TermForUsers = "users"
	}
	if connector.TermForGroups == "" {
		connector.TermForGroups = "groups"
	}
	connector.Targets.Users = c.Targets.Contains(state.Users)
	connector.Targets.Groups = c.Targets.Contains(state.Groups)
	connector.Targets.Events = c.Targets.Contains(state.Events)
	return &connector, nil
}

// Connectors returns the connectors.
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
			Name:                   c.Name,
			SourceDescription:      c.SourceDescription,
			DestinationDescription: c.DestinationDescription,
			TermForUsers:           c.TermForUsers,
			TermForGroups:          c.TermForGroups,
			Type:                   ConnectorType(c.Type),
			SendingMode:            (*SendingMode)(c.SendingMode),
			HasSheets:              c.HasSheets,
			HasUI:                  c.HasUI,
			IdentityIDLabel:        c.IdentityIDLabel,
			Icon:                   c.Icon,
			FileExtension:          c.FileExtension,
			SampleQuery:            c.SampleQuery,
			WebhooksPer:            WebhooksPer(c.WebhooksPer),
			OAuth:                  c.OAuth != nil,
		}
		if connector.TermForUsers == "" {
			connector.TermForUsers = "users"
		}
		if connector.TermForGroups == "" {
			connector.TermForGroups = "groups"
		}
		connector.Targets.Users = c.Targets.Contains(state.Users)
		connector.Targets.Groups = c.Targets.Contains(state.Groups)
		connector.Targets.Events = c.Targets.Contains(state.Events)
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
func (apis *APIs) CountOrganizations(ctx context.Context) int {
	apis.mustBeOpen()
	_, s := telemetry.TraceSpan(ctx, "apis.CountOrganizations")
	defer s.End()
	return len(apis.state.Organizations())
}

// ExpressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (apis *APIs) ExpressionsProperties(expressions []ExpressionToBeExtracted, schema types.Type) ([]string, error) {
	apis.mustBeOpen()
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
func (apis *APIs) MemberInvitation(ctx context.Context, token string) (string, string, error) {
	apis.mustBeOpen()
	if !isValidInvitationToken(token) {
		return "", "", errors.NotFound("invitation token %q does not exist", token)
	}
	var organizationID int
	var email string
	var createdAt time.Time
	err := apis.db.QueryRow(ctx, "SELECT organization, email, created_at FROM members WHERE invitation_token = $1", token).Scan(&organizationID, &email, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", errors.NotFound("invitation token %q does not exist", token)
		}
		return "", "", err
	}
	if isInvitationTokenExpired(createdAt) {
		return "", "", errors.Unprocessable(InvitationTokenExpired, "invitation token is expired")
	}
	organization, ok := apis.state.Organization(organizationID)
	if !ok {
		return "", "", errors.NotFound("invitation token %q does not exist", token)
	}
	return organization.Name, email, nil
}

// Organization returns the organization with identifier id.
//
// It returns an errors.NotFound error if the organization does not exist.
func (apis *APIs) Organization(ctx context.Context, id int) (*Organization, error) {
	apis.mustBeOpen()
	_, t := telemetry.TraceSpan(ctx, "apis.Organization", "organization_id", id)
	defer t.End()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid organization identifier", id)
	}
	org, ok := apis.state.Organization(id)
	if !ok {
		return nil, errors.NotFound("organization %d does not exist", id)
	}
	organization := Organization{
		apis:         apis,
		organization: org,
		ID:           org.ID,
		Name:         org.Name,
	}
	return &organization, nil
}

// Organizations returns the organizations, in the given order, describing all
// organizations but starting from first and up to limit. first must be >= 0 and
// limit must be > 0.
func (apis *APIs) Organizations(ctx context.Context, order OrganizationSort, first, limit int) ([]*Organization, error) {
	apis.mustBeOpen()
	_, s := telemetry.TraceSpan(ctx, "apis.Connectors")
	defer s.End()
	if order != SortByName {
		return nil, errors.BadRequest("order %d is not valid", int(order))
	}
	if limit <= 0 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}
	if first < 0 {
		return nil, errors.BadRequest("first %d is not valid", first)
	}
	organizations := apis.state.Organizations()
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
func (apis *APIs) ServeEvents(w http.ResponseWriter, r *http.Request) {
	apis.mustBeOpen()
	apis.events.collector.ServeHTTP(w, r)
}

// DataTransformation represents transformation passed to (*APIs).TransformData
// and (*Connection).PreviewSendEvent methods.
type DataTransformation struct {
	Mapping  map[string]string
	Function *DataTransformationFunction
}

// DataTransformationFunction represents transformation function passed to
// (*APIs).TransformData and (*Connection).PreviewSendEvent methods.
type DataTransformationFunction struct {
	Source       string
	Language     Language
	PreserveJSON bool
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
func (apis *APIs) TransformData(ctx context.Context, data []byte, inSchema, outSchema types.Type, transformation DataTransformation, purpose Purpose) ([]byte, error) {

	apis.mustBeOpen()

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
		mapping, err := mappings.New(transformation.Mapping, inSchema, outSchema, nil)
		if err != nil {
			return nil, errors.BadRequest("mapping is not valid: %s", err)
		}
		action.Transformation.InProperties = mapping.InProperties()
		action.Transformation.OutProperties = mapping.OutProperties()
	case transformation.Function != nil:
		if transformation.Function.Source == "" {
			return nil, errors.BadRequest("transformation source is empty")
		}
		switch transformation.Function.Language {
		case "JavaScript":
			if apis.transformerProvider == nil || !apis.transformerProvider.SupportLanguage(state.JavaScript) {
				return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript transformation language  is not supported")
			}
		case "Python":
			if apis.transformerProvider == nil || !apis.transformerProvider.SupportLanguage(state.Python) {
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
		action.Transformation.InProperties = types.PropertyNames(action.InSchema)
		action.Transformation.OutProperties = types.PropertyNames(action.OutSchema)
		provider = newTempTransformerProvider(name, transformation.Function.Source, apis.transformerProvider)
	default:
		return nil, errors.BadRequest("mapping (or transformation) is required")
	}

	properties, err := encoding.Unmarshal(bytes.NewReader(data), inSchema)
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

	return encoding.Marshal(outSchema, records[0].Properties)
}

// TransformationLanguages returns the supported transformation languages.
// Possible returned languages are "JavaScript" and "Python".
func (apis *APIs) TransformationLanguages() []string {
	if apis.transformerProvider == nil {
		return []string{}
	}
	languages := make([]string, 0, 2)
	if apis.transformerProvider.SupportLanguage(state.JavaScript) {
		languages = append(languages, "JavaScript")
	}
	if apis.transformerProvider.SupportLanguage(state.Python) {
		languages = append(languages, "Python")
	}
	return languages
}

// ValidateExpression validates an expression. properties represents the allowed
// properties in the expression, and typ is the type of the expression.
//
// The returned string explains why the expression is not valid. It is empty if
// the expression is valid.
func (apis *APIs) ValidateExpression(expression string, properties []types.Property, typ types.Type) (string, error) {
	apis.mustBeOpen()
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

// mustBeOpen panics if apis has been closed.
func (apis *APIs) mustBeOpen() {
	if apis.closed.Load() {
		panic("apis is closed")
	}
}

// onDeleteAction is called when an action is deleted.
func (apis *APIs) onDeleteAction(n state.DeleteAction) func() {
	if apis.state.IsLeader() && apis.transformerProvider != nil {
		go func() {
			for _, language := range [...]state.Language{state.JavaScript, state.Python} {
				if apis.transformerProvider.SupportLanguage(language) {
					name := transformationFunctionName(n.ID, language)
					err := apis.transformerProvider.Delete(apis.close.ctx, name)
					if err != nil {
						slog.Debug("cannot delete transformer function", "name", name, "err", err)
					}
				}
			}
		}()
	}
	return nil
}

// onExecuteAction is called when an action is executed.
func (apis *APIs) onExecuteAction(n state.ExecuteAction) func() {
	if !apis.state.IsLeader() {
		return nil
	}
	return func() {
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
}

func containsNUL(s string) bool {
	return strings.ContainsRune(s, '\x00')
}

func isValidInvitationToken(token string) bool {
	if len(token) != 44 {
		return false
	}
	_, err := base64.URLEncoding.DecodeString(token)
	return err == nil
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
