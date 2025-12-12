// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/collector"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	dbpkg "github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/initdb"
	coremetrics "github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/core/internal/transformers/lambda"
	"github.com/meergo/meergo/core/internal/transformers/local"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Core struct {
	db                *dbpkg.DB
	dbPoolMetrics     *dbPoolMetrics
	state             *state.State
	datastore         *datastore.Datastore
	connections       *connections.Connections
	metrics           *coremetrics.Collector
	collector         *collector.Collector
	functionProvider  transformers.FunctionProvider
	pipelineCleaner   *pipelineCleaner
	pipelineScheduler *pipelineScheduler
	memberEmailFrom   string
	smtp              *SMTPConfig
	close             struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		sync.WaitGroup
	}
	closed atomic.Bool

	// mcp holds an instance of a warehouses.Warehouse for every workspace, and
	// it is needed by the MCP (Model Context Protocol) server.
	//
	// If a workspace does not have MCP settings configured, the map has a key
	// and the value is nil.
	mcp   map[int]warehouses.Warehouse
	mcpMu sync.Mutex
}

var hasBeenCalled bool

type Config struct {
	DB                     DBConfig
	FunctionProvider       any // must be a LambdaConfig or LocalConfig value
	MaxMindDBPath          string
	MemberEmailFrom        string
	SMTP                   SMTPConfig
	OAuthCredentials       map[string]*OAuthCredentials
	SentryTelemetryLevel   TelemetryLevel
	DatabaseInitialization struct {
		// InitIfEmpty controls whether the PostgreSQL database should be
		// initialized in case it is empty.
		InitIfEmpty bool
		// InitDockerMember controls whether a member specific for Docker
		// scenarios is initialized. Requires InitIfEmpty to be true.
		InitDockerMember bool
	}
}

type DBConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	Schema         string
	MaxConnections int32 // values less than 2 are treated as 2.
}

// OAuthCredentials represents the OAuth client credentials for a connector.
type OAuthCredentials struct {
	ClientID     string
	ClientSecret string
}

type LambdaConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Role            string
	NodeJS          struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeJSExecutable string
	PythonExecutable string
	FunctionsDir     string
	SudoUser         string
	DoasUser         string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

// TelemetryLevel represent the Sentry telemetry level.
type TelemetryLevel string

const (
	TelemetryLevelNone   TelemetryLevel = "none"
	TelemetryLevelErrors TelemetryLevel = "errors"
	TelemetryLevelStats  TelemetryLevel = "stats"
	TelemetryLevelAll    TelemetryLevel = "all"
)

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
	ps := conf.DB
	db, err := dbpkg.Open(&dbpkg.Options{
		Host:           ps.Host,
		Port:           ps.Port,
		Username:       ps.Username,
		Password:       ps.Password,
		Database:       ps.Database,
		Schema:         ps.Schema,
		MaxConnections: max(2, ps.MaxConnections),
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

	// Initializes the PostgreSQL database if it is empty and the option to
	// initialize it is provided.
	dbInitCtx := context.Background()
	if conf.DatabaseInitialization.InitIfEmpty {
		isEmpty, err := initdb.DatabaseIsEmpty(dbInitCtx, db)
		if err != nil {
			return nil, fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
		}
		if isEmpty {
			slog.Info("the PostgreSQL database is empty, so the database will be initialized...")
			// Initialize the PostgreSQL database in a transaction, so if it is
			// fails, there is no need to manually empty the database.
			err := db.Transaction(dbInitCtx, func(tx *dbpkg.Tx) error {
				err := initdb.Initialize(dbInitCtx, tx)
				if err != nil {
					return fmt.Errorf("cannot initialize PostgreSQL database: %s", err)
				}
				slog.Info("PostgreSQL database initialized correctly")
				// Also initialize the Docker member, if requested.
				if conf.DatabaseInitialization.InitDockerMember {
					slog.Info("initializing Docker member...")
					err := initdb.InitializeDockerMember(dbInitCtx, tx)
					if err != nil {
						return fmt.Errorf("cannot initialize the Docker member: %s", err)
					}
					slog.Info("Docker member initialized")
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			slog.Info("the PostgreSQL database is not empty, so it won't be initialized")
		}
	}

	var smtp *SMTPConfig
	if conf.SMTP.Host != "" {
		smtp = &conf.SMTP
	}

	core := &Core{
		db:              db,
		dbPoolMetrics:   registerDBPoolMetrics(db),
		memberEmailFrom: conf.MemberEmailFrom,
		smtp:            smtp,
	}

	// Create a function provider.
	switch p := conf.FunctionProvider.(type) {
	case LambdaConfig:
		core.functionProvider = lambda.New(lambda.Settings(p))
	case LocalConfig:
		core.functionProvider = local.New(local.Settings(p))
	case nil:
	default:
		return nil, errors.New("invalid function provider")
	}

	// Instantiate the state.
	sendStats := conf.SentryTelemetryLevel == TelemetryLevelAll ||
		conf.SentryTelemetryLevel == TelemetryLevelStats
	var connectorsOAuth map[string]*state.OAuthCredentials
	if conf.OAuthCredentials != nil {
		connectorsOAuth = map[string]*state.OAuthCredentials{}
		for name, oAuth := range conf.OAuthCredentials {
			connectorsOAuth[name] = &state.OAuthCredentials{
				ClientID:     oAuth.ClientID,
				ClientSecret: oAuth.ClientSecret,
			}
		}
	}
	core.state, err = state.New(db, connectorsOAuth, sendStats)
	if err != nil {
		return nil, err
	}

	// Add the Meergo installation ID tag to Sentry.
	if conf.SentryTelemetryLevel != TelemetryLevelNone {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("meergo_installation_id", core.InstallationID())
		})
	}

	// Init the metrics.
	core.metrics = coremetrics.New(db, core.state)

	// Init the datastore.
	core.datastore = datastore.New(core.state)

	// Init the connections.
	core.connections = connections.New(core.state)

	// Init the event collector.
	core.collector, err = collector.New(db, core.state, core.datastore, core.connections, core.functionProvider, core.metrics, conf.MaxMindDBPath)
	if err != nil {
		core.datastore.Close()
		core.state.Close()
		return nil, err
	}

	// Create the pipeline cleaner.
	core.pipelineCleaner = newPipelineCleaner(core, core.functionProvider)

	// Create the pipeline scheduler.
	core.pipelineScheduler = newPipelineScheduler(core)

	core.close.ctx, core.close.cancelCtx = context.WithCancel(context.Background())

	// Instantiate a warehouses.Warehouse, used by the MCP server, for every workspace.
	core.mcp = map[int]warehouses.Warehouse{}
	for _, ws := range core.state.Workspaces() {
		var wh warehouses.Warehouse
		if ws.Warehouse.MCPSettings != nil {
			wh, _ = getMCPWarehouseInstance(ws.Warehouse.Platform, ws.Warehouse.MCPSettings)
		}
		core.mcp[ws.ID] = wh
	}

	// Listen to state changes.
	core.state.Freeze()
	core.state.AddListener(core.onCreateWorkspace)
	core.state.AddListener(core.onDeleteWorkspace)
	core.state.AddListener(core.onElectLeader)
	core.state.AddListener(core.onExecutePipeline)
	core.state.AddListener(core.onStartAlterProfileSchema)
	core.state.AddListener(core.onStartIdentityResolution)
	core.state.AddListener(core.onUpdateWarehouse)
	core.state.Unfreeze()

	// Try to start pending pipeline runs.
	for _, pipeline := range core.state.Pipelines() {
		if exe, ok := pipeline.Run(); ok {
			if _, ok := exe.Node(); !ok {
				core.tryStartPipelineRun(pipeline.ID)
			}
		}
	}

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
	if !isValidMemberToken(token) {
		return errors.NotFound("invitation token %q does not exist", token)
	}
	m := MemberToSet{
		Name:     name,
		Password: password,
	}
	err := validateMemberToSet(m, true, false, true)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
		var id int
		var createdAt time.Time
		err := tx.QueryRow(ctx, "SELECT id, created_at FROM members WHERE invitation_token = $1", token).Scan(&id, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NotFound("invitation token %q does not exist", token)
			}
			return nil, err
		}
		if isInvitationTokenExpired(createdAt) {
			return nil, errors.Unprocessable(InvitationTokenExpired, "invitation token is expired")
		}
		_, err = tx.Exec(ctx, "UPDATE members SET name = $1, password = $2, invitation_token = '' WHERE id = $3",
			name, string(pass), id)
		return nil, err
	})
	return err
}

// AddOrganization adds a new organization and returns its identifier.
// name cannot be empty and cannot be longer than 45 runes.
func (core *Core) AddOrganization(ctx context.Context, name string) (uuid.UUID, error) {
	core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 45); err != nil {
		return uuid.Nil, errors.BadRequest("%s", err)
	}
	var id uuid.UUID
	err := core.db.QueryRow(ctx, "INSERT INTO organizations (name) VALUES ($1) RETURNING id", name).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// AccessKey returns the organization and workspace identifiers associated with
// the provided access key token and type. If the access key is not restricted
// to a workspace, the workspace identifier will be 0. The boolean return value
// indicates whether the token exists.
func (core *Core) AccessKey(token string, typ AccessKeyType) (uuid.UUID, int, bool) {
	key, ok := core.state.AccessKeyByToken(token)
	if !ok || key.Type != state.AccessKeyType(typ) {
		return uuid.Nil, 0, false
	}
	return key.Organization, key.Workspace, true
}

// CanSendMemberPasswordReset returns whether it is possible to send the reset
// password email.
func (core *Core) CanSendMemberPasswordReset() bool {
	if core.smtp == nil || core.memberEmailFrom == "" {
		return false
	}
	return true
}

// ChangeMemberPasswordByToken changes the password of a member with the given
// reset password token. password's length must be at least 8 character long.
//
// If a reset password request with the given token does not exist or if the
// token is expired, it returns a NotFoundError error.
func (core *Core) ChangeMemberPasswordByToken(ctx context.Context, token string, password string) error {
	core.mustBeOpen()
	if !isValidMemberToken(token) {
		return errors.NotFound("reset password token %q does not exist or is expired", token)
	}
	m := MemberToSet{
		Password: password,
	}
	err := validateMemberToSet(m, false, false, true)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
		var id int
		var createdAt time.Time
		err := tx.QueryRow(ctx, "SELECT id, reset_password_token_created_at FROM members WHERE reset_password_token = $1", token).Scan(&id, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NotFound("reset password token %q does not exist or is expired", token)
			}
			return nil, err
		}
		if isResetPasswordTokenExpired(createdAt) {
			return nil, errors.NotFound("reset password token %q does not exist or is expired", token)
		}
		_, err = tx.Exec(ctx, "UPDATE members SET password = $1, reset_password_token = '', reset_password_token_created_at = NULL WHERE id = $2",
			string(pass), id)
		return nil, err
	})
	return err
}

// Close closes the Core. When Close is called, no other calls to Core's methods
// should be in progress and no other shall be made.
// It panics if it has already been called.
func (core *Core) Close() {
	if core.closed.Swap(true) {
		panic("core already closed")
	}
	// Cancel pipeline runs initiated via the API.
	core.close.cancelCtx()
	// Close the pipeline scheduler.
	core.pipelineScheduler.Close()
	// Wait for the completion of pipelines initiated via API.
	core.close.Wait()
	// Close the pipeline cleaner.
	core.pipelineCleaner.Close(context.Background())
	// Close the MCP warehouse connections.
	core.mcpMu.Lock()
	for _, mcp := range core.mcp {
		if mcp != nil {
			err := mcp.Close()
			if err != nil {
				slog.Warn("cannot close MCP warehouse connection", "err", err)
			}
		}
	}
	core.mcpMu.Unlock()
	// Close event collector, metrics, datastore, and state.
	core.collector.Close()
	core.metrics.Close(context.Background())
	core.datastore.Close()
	core.state.Close()
	// Unregister the database connection pool metrics.
	core.dbPoolMetrics.Unregister()
	// Close PostgreSQL connections.
	core.db.Close()
}

// Connector returns the connector with the provided code.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (core *Core) Connector(code string) (*Connector, error) {
	core.mustBeOpen()
	c, ok := core.state.Connector(code)
	if !ok {
		return nil, errors.NotFound("connector %q does not exist", code)
	}
	connector := Connector{
		core:          core,
		connector:     c,
		Code:          c.Code,
		Label:         c.Label,
		Type:          ConnectorType(c.Type),
		Categories:    categoryBitmaskToCategoryNames(c.Categories),
		HasSheets:     c.HasSheets,
		FileExtension: c.FileExtension,
		Terms:         ConnectorTerms(c.Terms),
		Strategies:    c.Strategies,
	}
	if c.SourceTargets != 0 {
		connector.AsSource = &SourceConnector{
			Targets:     stateToCoreTargets(c.SourceTargets),
			HasSettings: c.HasSourceSettings,
			SampleQuery: c.SampleQuery,
			WebhooksPer: WebhooksPer(c.WebhooksPer),
			Summary:     c.Documentation.Source.Summary,
		}
	}
	if c.DestinationTargets != 0 {
		connector.AsDestination = &DestinationConnector{
			Targets:     stateToCoreTargets(c.DestinationTargets),
			HasSettings: c.HasDestinationSettings,
			SendingMode: (*SendingMode)(c.SendingMode),
			Summary:     c.Documentation.Destination.Summary,
		}
	}
	if c.OAuth != nil {
		connector.OAuth = &ConnectorOAuth{
			Configured:        c.OAuth.ClientID != "" && c.OAuth.ClientSecret != "",
			Disallow127_0_0_1: c.OAuth.Disallow127_0_0_1,
			DisallowLocalhost: c.OAuth.DisallowLocalhost,
		}
	}
	return &connector, nil
}

// ConnectorDocumentation represents the documentation of a connector.
type ConnectorDocumentation struct {
	Source      ConnectorRoleDocumentation
	Destination ConnectorRoleDocumentation
}

// ConnectorRoleDocumentation represents the documentation of a connector
// relative to a role.
type ConnectorRoleDocumentation struct {
	Summary  string
	Overview string
}

// ConnectorDocumentation returns the documentation of the connector with the
// provided code.
//
// It returns an errors.NotFoundError error if the connector does not exist.
func (core *Core) ConnectorDocumentation(code string) (*ConnectorDocumentation, error) {
	core.mustBeOpen()
	c, ok := core.state.Connector(code)
	if !ok {
		return nil, errors.NotFound("connector %q does not exist", code)
	}
	doc := ConnectorDocumentation{
		Source:      ConnectorRoleDocumentation(c.Documentation.Source),
		Destination: ConnectorRoleDocumentation(c.Documentation.Destination),
	}
	return &doc, nil
}

// Connectors returns the connectors.
func (core *Core) Connectors() []*Connector {
	core.mustBeOpen()
	cc := core.state.Connectors()
	connectors := make([]*Connector, len(cc))
	for i, c := range cc {
		connector := Connector{
			core:          core,
			connector:     c,
			Code:          c.Code,
			Label:         c.Label,
			Type:          ConnectorType(c.Type),
			Categories:    categoryBitmaskToCategoryNames(c.Categories),
			HasSheets:     c.HasSheets,
			FileExtension: c.FileExtension,
			Terms:         ConnectorTerms(c.Terms),
			Strategies:    c.Strategies,
		}
		if c.SourceTargets != 0 {
			connector.AsSource = &SourceConnector{
				Targets:     stateToCoreTargets(c.SourceTargets),
				HasSettings: c.HasSourceSettings,
				SampleQuery: c.SampleQuery,
				WebhooksPer: WebhooksPer(c.WebhooksPer),
				Summary:     c.Documentation.Source.Summary,
			}
		}
		if c.DestinationTargets != 0 {
			connector.AsDestination = &DestinationConnector{
				Targets:     stateToCoreTargets(c.DestinationTargets),
				HasSettings: c.HasDestinationSettings,
				SendingMode: (*SendingMode)(c.SendingMode),
				Summary:     c.Documentation.Destination.Summary,
			}
		}
		if c.OAuth != nil {
			if c.OAuth != nil {
				connector.OAuth = &ConnectorOAuth{
					Configured:        c.OAuth.ClientID != "" && c.OAuth.ClientSecret != "",
					Disallow127_0_0_1: c.OAuth.Disallow127_0_0_1,
					DisallowLocalhost: c.OAuth.DisallowLocalhost,
				}
			}
		}
		connectors[i] = &connector
	}
	slices.SortFunc(connectors, func(a, b *Connector) int {
		if a.Code < b.Code {
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

// InstallationID returns the installation ID.
func (core *Core) InstallationID() string {
	return core.state.InstallationID()
}

// EncryptionKey returns the encryption key. It is 64 bytes long.
func (core *Core) EncryptionKey() []byte {
	return core.state.EncryptionKey()
}

// ExpressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (core *Core) ExpressionsProperties(expressions []ExpressionToBeExtracted, schema types.Type) ([]string, error) {
	core.mustBeOpen()
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.BadRequest("schema is not an object")
	}
	var properties []string
	for _, expression := range expressions {
		if expression.Value == "" {
			return nil, errors.BadRequest("expression value is empty")
		}
		if !expression.Type.Valid() {
			return nil, errors.BadRequest("expression type is not valid")
		}
		_, props, err := mappings.Compile(expression.Value, schema, expression.Type)
		if err != nil {
			return nil, errors.BadRequest("expression is not valid: %w", err)
		}
		properties = append(properties, props...)
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
	if !isValidMemberToken(token) {
		return "", "", errors.NotFound("invitation token %q does not exist", token)
	}
	var organizationID uuid.UUID
	var email string
	var createdAt time.Time
	err := core.db.QueryRow(ctx, "SELECT organization, email, created_at FROM members WHERE invitation_token = $1", token).Scan(&organizationID, &email, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", errors.NotFound("invitation token %q does not exist", token)
		}
		return "", "", err
	}
	// At this point an invited member with the given token exists,
	// and createdAt is the timestamp when the invitation was sent.
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
func (core *Core) Organization(id uuid.UUID) (*Organization, error) {
	core.mustBeOpen()
	if id == uuid.Nil {
		return nil, errors.BadRequest("identifier is not a valid organization identifier")
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
func (core *Core) Organizations(order OrganizationSort, first, limit int) ([]*Organization, error) {
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
			return a.Name < b.Name || a.Name == b.Name && bytes.Compare(a.ID[:], b.ID[:]) < 0
		}
		return false
	})
	organizations = organizations[first : first+limit]
	orgs := make([]*Organization, len(organizations))
	for i, organization := range organizations {
		orgs[i] = &Organization{
			core:         core,
			organization: organization,
			ID:           organization.ID,
			Name:         organization.Name,
		}
	}
	return orgs, nil
}

// ServeEvents serves the events sent via HTTP.
func (core *Core) ServeEvents(w http.ResponseWriter, r *http.Request) {
	core.mustBeOpen()
	core.collector.ServeHTTP(w, r)
}

// ValidateMemberPasswordResetToken validates the given password reset token.
//
// If a password reset request with the given password reset token does not
// exist or if the token is expired, it returns a NotFoundError error.
func (core *Core) ValidateMemberPasswordResetToken(ctx context.Context, token string) error {
	core.mustBeOpen()
	if !isValidMemberToken(token) {
		return errors.NotFound("reset password token %q does not exist or is expired", token)
	}
	var organizationID uuid.UUID
	var createdAt time.Time
	err := core.db.QueryRow(ctx, "SELECT organization, reset_password_token_created_at FROM members WHERE reset_password_token = $1", token).Scan(&organizationID, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NotFound("reset password token %q does not exist or is expired", token)
		}
		return err
	}
	if isResetPasswordTokenExpired(createdAt) {
		return errors.NotFound("reset password token %q does not exist or is expired", token)
	}
	_, ok := core.state.Organization(organizationID)
	if !ok {
		return errors.NotFound("reset password token %q does not exist or is expired", token)
	}
	return nil
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
// It can be "Import", "Create", or "Update".
type Purpose string

const (
	Import Purpose = "Import"
	Create Purpose = "Create"
	Update Purpose = "Update"
)

// TransformData transforms data using a mapping or a function transformation
// and returns the transformed data. inSchema is the schema of data, and
// outSchema is the schema of the transformed data. Only one of mapping and
// transformation must be non-nil. purpose indicates the intent of the
// transformation and can be "Import", "Create", or "Update".
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
	switch purpose {
	case Import, Create, Update:
	default:
		return nil, errors.BadRequest(`purpose must be "Import", "Create" or "Update"`)
	}
	if transformation.Mapping != nil && transformation.Function != nil {
		return nil, errors.BadRequest("mapping and function transformations cannot both be present")
	}

	pipeline := &state.Pipeline{
		InSchema:  inSchema,
		OutSchema: outSchema,
		Transformation: state.Transformation{
			Mapping: transformation.Mapping,
		},
	}

	// provider is a temporary function provider.
	var provider transformers.FunctionProvider

	// Validate the mapping and the transformation.
	switch {
	case transformation.Mapping != nil:
		mapping, err := mappings.New(transformation.Mapping, inSchema, outSchema, false, nil)
		if err != nil {
			return nil, errors.BadRequest("mapping is not valid: %s", err)
		}
		pipeline.Transformation.InPaths = mapping.InPaths()
		pipeline.Transformation.OutPaths = mapping.OutPaths()
	case transformation.Function != nil:
		if transformation.Function.Source == "" {
			return nil, errors.BadRequest("function source is empty")
		}
		switch transformation.Function.Language {
		case "JavaScript":
			if core.functionProvider == nil || !core.functionProvider.SupportLanguage(state.JavaScript) {
				return nil, errors.Unprocessable(UnsupportedLanguage, "JavaScript language is not supported")
			}
		case "Python":
			if core.functionProvider == nil || !core.functionProvider.SupportLanguage(state.Python) {
				return nil, errors.Unprocessable(UnsupportedLanguage, "Python language is not supported")
			}
		case "":
			return nil, errors.BadRequest("function language is empty")
		default:
			return nil, errors.BadRequest("function language %q is not valid", transformation.Function.Language)
		}
		pipeline.Transformation.Function = &state.TransformationFunction{
			Source:  transformation.Function.Source,
			Version: "1", // no matter the version, it will be overwritten by the temporary function.
		}
		name := transformationFunctionName(0)
		switch transformation.Function.Language {
		case "JavaScript":
			pipeline.Transformation.Function.Language = state.JavaScript
		case "Python":
			pipeline.Transformation.Function.Language = state.Python
		}
		pipeline.Transformation.Function.PreserveJSON = transformation.Function.PreserveJSON
		// In InPaths and OutPaths, list only top-level property names; there is
		// no need to list sub-property paths (as the behavior is the same).
		pipeline.Transformation.InPaths = pipeline.InSchema.Properties().SortedNames()
		pipeline.Transformation.OutPaths = pipeline.OutSchema.Properties().SortedNames()
		provider = newTempTransformerProvider(name, pipeline.Transformation.Function.Language, pipeline.Transformation.Function.Source, core.functionProvider)
	default:
		return nil, errors.BadRequest("mapping (or function) is required")
	}

	attributes, err := types.Decode[map[string]any](bytes.NewReader(data), inSchema)
	if err != nil {
		return nil, errors.BadRequest("data does not validate against the input schema: %w", err)
	}

	// Transform the attributes.
	transformer, err := transformers.New(pipeline, provider, nil)
	if err != nil {
		return nil, err
	}
	records := []transformers.Record{
		{Purpose: transformers.Import, Attributes: attributes},
	}
	if purpose == "Create" {
		records[0].Purpose = transformers.Create
	} else if purpose == "Update" {
		records[0].Purpose = transformers.Update
	}
	err = transformer.Transform(ctx, records)
	if err != nil {
		if _, ok := err.(transformers.FunctionExecError); ok {
			err = errors.Unprocessable(TransformationFailed, "%s", err)
		}
		return nil, err
	}
	if err = records[0].Err; err != nil {
		return nil, errors.Unprocessable(TransformationFailed, "%s", err)
	}

	return types.Marshal(records[0].Attributes, outSchema)
}

// TransformationLanguages returns the supported transformation languages.
// Possible returned languages are "JavaScript" and "Python".
func (core *Core) TransformationLanguages() []string {
	if core.functionProvider == nil {
		return []string{}
	}
	languages := make([]string, 0, 2)
	if core.functionProvider.SupportLanguage(state.JavaScript) {
		languages = append(languages, "JavaScript")
	}
	if core.functionProvider.SupportLanguage(state.Python) {
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
	_, _, err = mappings.Compile(expression, schema, typ)
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}

// WarehousePlatform represents a warehouse platform.
type WarehousePlatform struct {
	Name string `json:"name"`
}

// WarehousePlatforms returns the warehouse platforms.
func (core *Core) WarehousePlatforms() []WarehousePlatform {
	core.mustBeOpen()
	platforms := core.state.WarehousePlatforms()
	warehousePlatforms := make([]WarehousePlatform, len(platforms))
	for i, p := range platforms {
		warehousePlatforms[i] = WarehousePlatform{
			Name: p.Name,
		}
	}
	return warehousePlatforms
}

// mustBeOpen panics if core has been closed.
func (core *Core) mustBeOpen() {
	if core.closed.Load() {
		panic("core is closed")
	}
}

// onExecutePipeline is called when a pipeline is executed.
func (core *Core) onExecutePipeline(n state.RunPipeline) {
	core.tryStartPipelineRun(n.Pipeline)
}

// pipelineError represents a pipeline error.
type pipelineError struct {
	step coremetrics.Step
	err  error
}

func newPipelineError(step coremetrics.Step, err error) *pipelineError {
	return &pipelineError{step, err}
}

func (err pipelineError) Error() string {
	return err.err.Error()
}

// tryStartPipelineRun attempts to start a pipeline run.
// It returns immediately and spawns a new goroutine to handle the run.
func (core *Core) tryStartPipelineRun(pipelineID int) {

	core.close.Go(func() {

		ctx := core.close.ctx

		var pipeline *state.Pipeline
		var run *state.PipelineRun

		var ok bool
		pipeline, ok = core.state.Pipeline(pipelineID)
		if !ok {
			return
		}
		run, ok = pipeline.Run()
		if !ok {
			return
		}
		if _, ok := run.Node(); ok {
			return
		}

		// Attempt to acquire the run. If already acquired by another node, return early.
		bo := backoff.New(200)
		for bo.Next(ctx) {
			var node uuid.UUID
			err := core.db.QueryRow(core.close.ctx,
				"UPDATE pipelines_runs\nSET node = $1\nWHERE id = $2 AND node IS NULL RETURNING node",
				core.state.ID(), run.ID).Scan(&node)
			if err != nil {
				if err == sql.ErrNoRows {
					// The run no longer exists.
					return
				}
				if err := ctx.Err(); err != nil {
					// The context has been canceled.
					break
				}
				slog.Error(fmt.Sprintf("core: cannot start pipeline run, retrying after %s", bo.WaitTime()), "error", err)
				continue
			}
			if node != core.state.ID() {
				// Another node acquired the run.
				return
			}
			break
		}
		if err := ctx.Err(); err != nil {
			// The context has been canceled.
			return
		}

		// Ping the run.
		pingCtx, stopPing := context.WithCancel(ctx)
		go func(id int) {
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ticker.C:
					pingTime := time.Now().UTC()
					_, err := core.db.Exec(pingCtx, "UPDATE pipelines_runs SET ping_time = $1 WHERE id = $2", pingTime, id)
					if err != nil && pingCtx.Err() == nil {
						slog.Error("core: cannot update pipeline run ping time", "err", err)
					}
				case <-pingCtx.Done():
					return
				}
			}
		}(run.ID)
		defer stopPing()

		// Prepare the run metrics.
		timeSlot := coremetrics.TimeSlotFromTime(run.StartTime)
		bo = backoff.New(200)
		for bo.Next(ctx) {
			_, err := core.db.Exec(ctx,
				// If statistics from previous runs of the same pipeline are available,
				// they are subtracted from the current run's statistics. This ensures
				// that when all slot statistics are merged into those of this run,
				// the resulting data will be accurate and consistent.
				"WITH s AS (\n"+
					"	SELECT -passed_0 as passed_0, -passed_1 as passed_1, -passed_2 as passed_2, -passed_3 as passed_3,"+
					" -passed_4 as passed_4, -passed_5 as passed_5, -failed_0 as failed_0, -failed_1 as failed_1,"+
					" -failed_2 as failed_2, -failed_3 as failed_3, -failed_4 as failed_4, -failed_5 as failed_5\n"+
					"	FROM pipelines_metrics\n"+
					"	WHERE pipeline = $2 AND timeslot = $3\n"+
					")\n"+
					"UPDATE pipelines_runs\n"+
					"SET passed_0 = s.passed_0, passed_1 = s.passed_1, passed_2 = s.passed_2, passed_3 = s.passed_3,"+
					" passed_4 = s.passed_4, passed_5 = s.passed_5, failed_0 = s.failed_0, failed_1 = s.failed_1,"+
					" failed_2 = s.failed_2, failed_3 = s.failed_3, failed_4 = s.failed_4, failed_5 = s.failed_5\n"+
					"FROM s\n"+
					"WHERE id = $1", run.ID, pipeline.ID, timeSlot)
			if err != nil {
				if err == sql.ErrNoRows {
					// The run no longer exists.
					return
				}
				if err := ctx.Err(); err != nil {
					// The context has been canceled.
					break
				}
				slog.Error(fmt.Sprintf("core: cannot start pipeline run, retrying after %s", bo.WaitTime()), "error", err)
				continue
			}
			break
		}
		if err := ctx.Err(); err != nil {
			// The context has been canceled.
			// TODO(marco): What happens if the node successfully assigns itself the run, but the context gets canceled?
			return
		}

		// Starts the run.
		c := pipeline.Connection()
		ws := c.Workspace()
		store := core.datastore.Store(ws.ID)
		connection := &Connection{core: core, store: store, connection: c}
		p := &Pipeline{core: core, pipeline: pipeline, connection: connection}

		var err error
		if c.Role == state.Source {
			err = p.importUsers(ctx)
		} else {
			err = p.exportProfiles(ctx)
		}

		// Mark the run as ended.
		p.endRun(err)

		// Stop pinging as it is no longer required.
		stopPing()

		// Start the Identity Resolution, if necessary.
		if c.Role == state.Source && ws.ResolveIdentitiesOnBatchImport {
			err := core.startIdentityResolution(ctx, ws.ID)
			if err != nil {
				if err2, ok := err.(*errors.UnprocessableError); ok && err2.Code == OperationAlreadyExecuting {
					// Do nothing.
				} else {
					slog.Error("core: cannot start Identity Resolution at the end of import", "pipeline", pipeline.ID, "run", run.ID, "err", err)
				}
			}
		}

	})
}

// executeAlterProfileSchema executes the alter of the profile schema, not
// returning until it has completed (with success or with an operation error).
//
// primarySources cannot be nil.
func (core *Core) executeAlterProfileSchema(workspace int, opID string, schema types.Type,
	primarySources map[string]int, operations []warehouses.AlterOperation) {
	ctx := core.close.ctx
	store := core.datastore.Store(workspace)
	ws, ok := core.state.Workspace(workspace)
	if !ok {
		return
	}
	// Keep calling 'AlterProfileSchema' until it (1) returns successfully, (2)
	// returns with a *warehouses.OperationError, or (3) the context is
	// canceled.
	var alterSchemaErr *warehouses.OperationError
	bo := backoff.New(200)
	bo.SetCap(5 * time.Minute)
	for bo.Next(ctx) {
		err := store.AlterProfileSchema(ctx, opID, schema, operations)
		// In case of success, go on and send an EndAlterProfileSchema
		// notification.
		if err == nil {
			break
		}
		// If the context has expired, just return.
		if ctx.Err() != nil {
			return
		}
		// In case of OperationError log it, then go on and send an
		// EndAlterProfileSchema notification.
		if err2, ok := err.(*warehouses.OperationError); ok {
			slog.Error("alter schema ended with an error", "err", err2)
			alterSchemaErr = err2
			break
		}
		// In case of unknown error, try again.
		slog.Error("alter schema on warehouse returned an unknown error, "+
			"so the operation will be retried after the indicated timeout or the next time you restart Meergo",
			"err", err, "timeout", bo.WaitTime())

	}
	nEnd := state.EndAlterProfileSchema{
		Workspace: workspace,
		ID:        opID,
		EndTime:   time.Now().UTC(),
		Schema:    schema,
	}
	if alterSchemaErr != nil {
		nEnd.Err = alterSchemaErr.Error()
	}
	// Build the query to insert the primary paths.
	var insertPrimarySources string
	var paths []any
	if len(primarySources) > 0 {
		i := 0
		var b strings.Builder
		for path, source := range primarySources {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('(')
			b.WriteString(strconv.Itoa(source))
			b.WriteString(",$")
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(")")
			paths = append(paths, path)
			i++
		}
		insertPrimarySources = "INSERT INTO primary_sources (source, path) VALUES " + b.String()
	}
	// Update the identifiers.
	nEnd.Identifiers = make([]string, 0, len(ws.Identifiers))
Identifiers:
	for _, identifier := range ws.Identifiers {
		for _, operation := range operations {
			if operation.Operation == warehouses.OperationAddColumn {
				continue
			}
			if path := strings.ReplaceAll(operation.Column, "_", "."); path != identifier {
				continue
			}
			if operation.Operation == warehouses.OperationRenameColumn {
				nEnd.Identifiers = append(nEnd.Identifiers, strings.ReplaceAll(operation.NewColumn, "_", "."))
			}
			continue Identifiers
		}
		nEnd.Identifiers = append(nEnd.Identifiers, identifier)
	}
	for {
		err := core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
			if nEnd.Err == "" {
				// These columns should be updated only in case of success,
				// otherwise, in case of error, the current ones should be left.
				//
				// Update the profile schema.
				query := "UPDATE workspaces SET profile_schema = alter_profile_schema_schema," +
					" identifiers = $1 WHERE id = $2"
				_, err := tx.Exec(ctx, query, nEnd.Identifiers, nEnd.Workspace)
				if err != nil {
					return nil, err
				}
				// Update the primary sources.
				_, err = tx.Exec(ctx, "DELETE FROM primary_sources s USING connections c\n"+
					"WHERE c.workspace = $1 AND s.source = c.id", workspace)
				if err != nil {
					return nil, err
				}
				if insertPrimarySources != "" {
					_, err = tx.Exec(ctx, insertPrimarySources, paths...)
					if err != nil {
						return nil, err
					}
				}
			}
			// Set the alter schema update as completed.
			query := "UPDATE workspaces SET alter_profile_schema_id = NULL," +
				" alter_profile_schema_schema = 'null', alter_profile_schema_primary_sources = 'null'," +
				" alter_profile_schema_operations = 'null', alter_profile_schema_end_time = $1," +
				" alter_profile_schema_error = $2 WHERE id = $3 AND alter_profile_schema_id = $4"
			res, err := tx.Exec(ctx, query, nEnd.EndTime, nEnd.Err, nEnd.Workspace, nEnd.ID)
			if err != nil {
				return nil, err
			}
			if res.RowsAffected() == 0 {
				// This happens in cases where the query has been executed
				// more than once (because an error occurred), but in fact
				// the database has already been modified, so we don't want
				// to send the notification more than once.
				return nil, nil
			}
			return nEnd, nil
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Try again to do the queries and send the notification.
			continue
		}
		// No errors: break the loop.
		break
	}
}

// executeIdentityResolution executes the Identity Resolution, not returning
// until it has completed (with success or with an operation error).
func (core *Core) executeIdentityResolution(workspace int, opID string) {
	ctx := core.close.ctx
	store := core.datastore.Store(workspace)
	// Keep calling 'ResolveIdentities' until it (1) returns successfully,
	// (2) returns with a *warehouses.OperationError, or (3) the context is
	// canceled.
	bo := backoff.New(200)
	bo.SetCap(5 * time.Minute)
	for bo.Next(ctx) {
		err := store.ResolveIdentities(ctx, opID)
		// In case of success, go on and send an EndIdentityResolution
		// notification.
		if err == nil {
			break
		}
		// If the context has expired, just return.
		if ctx.Err() != nil {
			return
		}
		// In case of OperationError log it, then go on and send an
		// EndIdentityResolution notification.
		if err2, ok := err.(*warehouses.OperationError); ok {
			slog.Error("identity resolution ended with an error", "err", err2)
			break
		}
		// In case of unknown error, try again.
		slog.Error("identity resolution on warehouse returned an unknown error, "+
			"so the operation will be retried after the indicated timeout or the next time you restart Meergo",
			"err", err, "timeout", bo.WaitTime())

	}
	nEnd := state.EndIdentityResolution{
		Workspace: workspace,
		ID:        opID,
		EndTime:   time.Now().UTC(),
	}
	bo = backoff.New(200)
	bo.SetCap(time.Second)
	for bo.Next(ctx) {
		err := core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
			query := "UPDATE workspaces SET ir_id = NULL, ir_end_time = $1 WHERE id = $2 AND ir_id = $3"
			res, err := tx.Exec(ctx, query, nEnd.EndTime, nEnd.Workspace, nEnd.ID)
			if err != nil {
				return nil, err
			}
			if res.RowsAffected() == 0 {
				// This happens in cases where the query has been executed
				// more than once (because an error occurred), but in fact
				// the database has already been modified, so we don't want
				// to send the notification more than once.
				return nil, nil
			}
			return nEnd, nil
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Try again to do the query and send the notification.
			continue
		}
		// No errors: break the loop.
		break
	}
}

// onCreateWorkspace is called when a workspace is created.
func (core *Core) onCreateWorkspace(n state.CreateWorkspace) {
	ws, _ := core.state.Workspace(n.ID)
	var wh warehouses.Warehouse
	if ws.Warehouse.MCPSettings != nil {
		wh, _ = getMCPWarehouseInstance(ws.Warehouse.Platform, ws.Warehouse.MCPSettings)
	}
	core.mcpMu.Lock()
	core.mcp[ws.ID] = wh
	core.mcpMu.Unlock()
}

// onDeleteWorkspace is called when a workspace is deleted.
func (core *Core) onDeleteWorkspace(n state.DeleteWorkspace) {
	core.mcpMu.Lock()
	wh, ok := core.mcp[n.ID]
	delete(core.mcp, n.ID)
	core.mcpMu.Unlock()
	if ok && wh != nil {
		go func(workspace int) {
			err := wh.Close()
			if err != nil {
				slog.Error("core: error closing a MCP warehouse", "workspace", workspace, "err", err)
			}
		}(n.ID)
	}
}

// onElectLeader is called when a leader is elected.
func (core *Core) onElectLeader(n state.ElectLeader) {
	if !core.state.IsLeader() {
		return
	}
	workspaces := core.state.Workspaces()
	for _, ws := range workspaces {
		if ws.IR.ID != nil {
			go core.executeIdentityResolution(ws.ID, *ws.IR.ID)
			// At most only one operation between Identity Resolution and update
			// profile schema can be started, so continue to the next workspace.
			continue
		}
		if ws.AlterProfileSchema.ID != nil {
			go core.executeAlterProfileSchema(ws.ID, *ws.AlterProfileSchema.ID,
				ws.AlterProfileSchema.Schema, ws.AlterProfileSchema.PrimarySources,
				ws.AlterProfileSchema.Operations)
		}
	}
}

// onStartAlterProfileSchema is called when the alter of the profile schema is
// started.
func (core *Core) onStartAlterProfileSchema(n state.StartAlterProfileSchema) {
	if !core.state.IsLeader() {
		return
	}
	go core.executeAlterProfileSchema(n.Workspace, n.ID, n.Schema, n.PrimarySources, n.Operations)
}

// onStartIdentityResolution is called when the identity resolution is started.
func (core *Core) onStartIdentityResolution(n state.StartIdentityResolution) {
	if !core.state.IsLeader() {
		return
	}
	go core.executeIdentityResolution(n.Workspace, n.ID)
}

// onUpdateWarehouse is called when a warehouse is updated.
func (core *Core) onUpdateWarehouse(n state.UpdateWarehouse) {

	// Update the MCP warehouse if the settings have changed.
	core.mcpMu.Lock()
	prevWarehouse := core.mcp[n.Workspace]
	core.mcpMu.Unlock()
	ws, _ := core.state.Workspace(n.Workspace)

	// The MCP settings were and have remained unset (nil).
	if prevWarehouse == nil && n.MCPSettings == nil {
		// Nothing to do.
		return
	}

	// The MCP settings changed from set to unset (nil).
	if prevWarehouse != nil && n.MCPSettings == nil {
		core.mcpMu.Lock()
		core.mcp[n.Workspace] = nil
		core.mcpMu.Unlock()
		// Close the previous warehouse.
		go func(workspace int) {
			err := prevWarehouse.Close()
			if err != nil {
				slog.Error("core: error closing a MCP warehouse", "workspace", workspace, "err", err)
			}
		}(ws.ID)
		return
	}

	// The MCP settings were unset (nil) and have now been set.
	if prevWarehouse == nil && n.MCPSettings != nil {
		nextWarehouse, _ := getMCPWarehouseInstance(ws.Warehouse.Platform, n.MCPSettings)
		core.mcpMu.Lock()
		core.mcp[n.Workspace] = nextWarehouse
		core.mcpMu.Unlock()
		return
	}

	// The MCP settings were set and have been set again with the same value.
	nextWarehouse, _ := getMCPWarehouseInstance(ws.Warehouse.Platform, n.MCPSettings)
	if bytes.Equal(prevWarehouse.Settings(), nextWarehouse.Settings()) {
		return
	}

	// The MCP settings were set and have been set with a different value.
	core.mcpMu.Lock()
	core.mcp[n.Workspace] = nextWarehouse
	core.mcpMu.Unlock()
	// Close the previous warehouse.
	go func(workspace int) {
		err := prevWarehouse.Close()
		if err != nil {
			slog.Error("core: error closing a MCP warehouse", "workspace", workspace, "err", err)
		}
	}(ws.ID)

}

// startAlterProfileSchema starts the alter of the profile schema.
//
// primarySources cannot be nil.
//
// It returns an errors.UnprocessableError error with code
//
//   - OperationAlreadyExecuting, if another operation (identity resolution or
//     profile schema update) is already running.
//   - ConnectionNotExist, if a connection referred in the primary sources does
//     not exist.
func (core *Core) startAlterProfileSchema(ctx context.Context, ws int, schema types.Type, primarySources map[string]int, operations []warehouses.AlterOperation) error {
	core.mustBeOpen()
	opID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	n := state.StartAlterProfileSchema{
		Workspace:      ws,
		ID:             opID.String(),
		StartTime:      time.Now().UTC(),
		Schema:         schema,
		PrimarySources: primarySources,
		Operations:     operations,
	}
	// Prepare the query to check whether the connections referred within the
	// primary sources exist or not.
	connIDs := []int{}
	var connQuery strings.Builder
	if len(primarySources) > 0 {
		for _, connID := range primarySources {
			if !slices.Contains(connIDs, connID) {
				connIDs = append(connIDs, connID)
			}
		}
		slices.Sort(connIDs)
		connQuery.WriteString("SELECT count(*) FROM connections WHERE id IN (")
		for i, c := range connIDs {
			if i > 0 {
				connQuery.WriteByte(',')
			}
			connQuery.WriteString(strconv.Itoa(c))
		}
		connQuery.WriteByte(')')
	}
	err = core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
		// Check if primary sources connections exist.
		if len(primarySources) > 0 {
			var count int
			err := tx.QueryRow(ctx, connQuery.String()).Scan(&count)
			if err != nil {
				return nil, err
			}
			if count < len(connIDs) {
				return nil, errors.Unprocessable(ConnectionNotExist, "a primary source does not exist")
			}
		}
		// Check that there are no other operations in progress on the
		// warehouse.
		var ongoingOp bool
		query := `SELECT alter_profile_schema_id IS NOT NULL OR ir_id IS NOT NULL FROM workspaces WHERE id = $1`
		err = tx.QueryRow(ctx, query, n.Workspace).Scan(&ongoingOp)
		if err != nil {
			return nil, err
		}
		if ongoingOp {
			return nil, errors.Unprocessable(OperationAlreadyExecuting, "another operation is already executing")
		}
		// Sets the alter profile schema operation to running.
		query = "UPDATE workspaces SET alter_profile_schema_id = $1," +
			" alter_profile_schema_schema = $2, alter_profile_schema_primary_sources = $3," +
			" alter_profile_schema_operations = $4, alter_profile_schema_start_time = $5," +
			" alter_profile_schema_end_time = NULL, alter_profile_schema_error = NULL WHERE id = $6"
		_, err = tx.Exec(ctx, query, n.ID, n.Schema, n.PrimarySources, n.Operations,
			n.StartTime, n.Workspace)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// startIdentityResolution starts an Identity Resolution.
//
// If another operation (identity resolution or profile schema update) is
// already running, this method returns an errors.UnprocessableError error with
// code OperationAlreadyExecuting.
func (core *Core) startIdentityResolution(ctx context.Context, ws int) error {
	core.mustBeOpen()
	opID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	n := state.StartIdentityResolution{
		Workspace: ws,
		ID:        opID.String(),
		StartTime: time.Now().UTC(),
	}
	err = core.state.Transaction(ctx, func(tx *dbpkg.Tx) (any, error) {
		var ongoingOp bool
		query := `SELECT alter_profile_schema_id IS NOT NULL OR ir_id IS NOT NULL FROM workspaces WHERE id = $1`
		err := tx.QueryRow(ctx, query, n.Workspace).Scan(&ongoingOp)
		if err != nil {
			return nil, err
		}
		if ongoingOp {
			return nil, errors.Unprocessable(OperationAlreadyExecuting, "another operation is already executing")
		}
		query = "UPDATE workspaces SET ir_id = $1, ir_start_time = $2, ir_end_time = NULL WHERE id = $3"
		_, err = tx.Exec(ctx, query, n.ID, n.StartTime, n.Workspace)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// EventColumnByPath returns the warehouses.Column corresponding to the property
// of the events schema with the specified path.
// propertyPath must always refer to an existing property in the event schema.
func EventColumnByPath(propertyPath string) warehouses.Column {
	return datastore.EventColumnByPath(propertyPath)
}

// EventSchema returns the event schema.
func EventSchema() types.Type {
	return schemas.Event
}

// categoryBitmaskToCategoryNames converts a bitmask representing a connector's
// categories into a slice of strings containing the various category names.
func categoryBitmaskToCategoryNames(categoryBitmask connectors.Categories) []string {
	categoryNames := []string{}
	for i := range 64 {
		if categoryBitmask&(1<<i) != 0 {
			categoryName := connectors.Categories(1 << i).String()
			categoryNames = append(categoryNames, categoryName)
		}
	}
	return categoryNames
}

// getMCPWarehouseInstance returns a meergo.Warehouse instance that can be used
// to implement features for the MCP server.
// platform is the warehouse platform and settings are the settings for
// connecting to it.
func getMCPWarehouseInstance(platform string, settings json.Value) (warehouses.Warehouse, error) {
	wh, err := warehouses.Registered(platform).New(&warehouses.Config{Settings: settings})
	if err != nil {
		return nil, err
	}
	return wh, nil
}

func isValidMemberToken(token string) bool {
	if len(token) != 44 {
		return false
	}
	_, err := base64.URLEncoding.DecodeString(token)
	return err == nil
}

func stateToCoreTargets(targets state.ConnectorTargets) []Target {
	ts := []Target{}
	if targets.Contains(state.TargetUser) {
		ts = append(ts, TargetUser)
	}
	if targets.Contains(state.TargetGroup) {
		ts = append(ts, TargetGroup)
	}
	if targets.Contains(state.TargetEvent) {
		ts = append(ts, TargetEvent)
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
