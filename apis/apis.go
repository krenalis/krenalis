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
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/encoding"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events/collector"
	"github.com/open2b/chichi/apis/events/dispatcher"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/statistics"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/apis/transformers/lambda"
	"github.com/open2b/chichi/apis/transformers/local"
	"github.com/open2b/chichi/apis/transformers/mappings"
	"github.com/open2b/chichi/telemetry"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	TransformationFailed   errors.Code = "TransformationFailed"
	InvitationTokenExpired errors.Code = "InvitationTokenExpired"
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
	statistics *statistics.Collector
	events     struct {
		collector  *collector.Collector
		observer   *collector.Observer
		dispatcher *dispatcher.Dispatcher
	}
	functionTransformer transformers.Function
	mu                  sync.Mutex // for the scheduler field
	scheduler           *scheduler
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
	PostgreSQL  PostgreSQLConfig
	Transformer any // must be a LambdaConfig or LocalConfig value
	SMTP        SMTPConfig
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

	var smtp *SMTPConfig
	if conf.SMTP.Host != "" {
		smtp = &conf.SMTP
	}

	apis := &APIs{db: db, smtp: smtp}

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

	// Init the datastore.
	apis.datastore = datastore.New(apis.state)

	// Init the connectors.
	apis.connectors = connectors.New(db, apis.state)

	// Init the statistics.
	apis.statistics = statistics.New(db)

	// Init the events.
	apis.events.dispatcher, err = dispatcher.New(db, apis.state, apis.functionTransformer, apis.connectors)
	if err != nil {
		apis.datastore.Close()
		apis.state.Close()
		return nil, err
	}
	apis.events.collector, err = collector.New(db, apis.state, apis.datastore, apis.functionTransformer, apis.events.dispatcher)
	if err != nil {
		apis.events.dispatcher.Close()
		apis.datastore.Close()
		apis.state.Close()
		return nil, err
	}
	apis.events.observer = apis.events.collector.Observer()

	// Keep the state updated.
	apis.state.Keep()

	apis.close.ctx, apis.close.cancelCtx = context.WithCancel(context.Background())

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
		return errors.BadRequest(err.Error())
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
	// Close event dispatcher, statistics, datastore and state.
	apis.events.dispatcher.Close()
	apis.statistics.Close()
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
		SendingMode:            (*SendingMode)(c.SendingMode),
		HasSheets:              c.HasSheets,
		HasUI:                  c.HasUI,
		ExternalIDLabel:        c.ExternalIDLabel,
		SuggestedDisplayedID:   c.SuggestedDisplayedID,
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
			ID:                     c.ID,
			Name:                   c.Name,
			SourceDescription:      c.SourceDescription,
			DestinationDescription: c.DestinationDescription,
			TermForUsers:           c.TermForUsers,
			TermForGroups:          c.TermForGroups,
			Type:                   ConnectorType(c.Type),
			SendingMode:            (*SendingMode)(c.SendingMode),
			HasSheets:              c.HasSheets,
			HasUI:                  c.HasUI,
			ExternalIDLabel:        c.ExternalIDLabel,
			SuggestedDisplayedID:   c.SuggestedDisplayedID,
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
	sort.Slice(connectors, func(i, j int) bool {
		a, b := connectors[i], connectors[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
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
	var properties []types.Path
	for _, expression := range expressions {
		if expression.Value == "" {
			return nil, errors.BadRequest("expression value is empty")
		}
		if !expression.Type.Valid() {
			return nil, errors.BadRequest("expression type is not valid")
		}
		exp, err := mappings.Compile(expression.Value, schema, expression.Type, false, true, nil)
		if err != nil {
			return nil, errors.BadRequest("expression is not valid: %w", err)
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
		for path, expr := range transformation.Mapping {
			outPath, err := types.ParsePropertyPath(path)
			if err != nil {
				return nil, errors.BadRequest("output mapped property %q is not valid", path)
			}
			p, err := outSchema.PropertyByPath(outPath)
			if err != nil {
				err := err.(types.PathNotExistError)
				return nil, errors.BadRequest("output mapped property %s not found in output schema", err.Path)
			}
			_, err = mappings.Compile(expr, inSchema, p.Type, p.Required, p.Nullable, nil)
			if err != nil {
				return nil, errors.BadRequest("invalid expression mapped to %s: %s", path, err)
			}
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
		if err, ok := err.(ValidationError); ok {
			return nil, errors.Unprocessable(TransformationFailed, err.Error())
		}
		return nil, err
	}

	return encoding.Marshal(outSchema, value)
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

func isValidInvitationToken(token string) bool {
	if len(token) != 44 {
		return false
	}
	_, err := base64.URLEncoding.DecodeString(token)
	return err == nil
}

// Workspace represents a workspace.
type Workspace struct {
	apis                *APIs
	organization        *Organization
	store               *datastore.Store
	workspace           *state.Workspace
	ID                  int
	Name                string
	UsersSchema         types.Type
	Identifiers         []string
	PrivacyRegion       PrivacyRegion
	DisplayedProperties DisplayedProperties
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

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

// DisplayedProperties represents the displayed properties.
type DisplayedProperties struct {
	Image       string
	FirstName   string
	LastName    string
	Information string
}
