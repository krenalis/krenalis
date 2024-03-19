//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package connectors provides the interface to interact with app, database,
// file, mobile, server, stream and website connectors.
package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/connectors/httpclient"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

// SchemaError represents an error with a schema.
type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}

// validationError represents a record validation error. It implements the
// ValidationError interface of apis.
type validationError struct {
	path string
	msg  string
}

func (err *validationError) Error() string {
	return err.msg
}

func (err *validationError) PropertyPath() string {
	return err.path
}

// newValidationErrorf returns a *validationError error based on a format
// specifier. The error message can report the invalid value and should complete
// the sentence "property foo ".
func newValidationErrorf(path string, format string, a ...any) error {
	return &validationError{
		path: path,
		msg:  fmt.Sprintf("property %q ", path) + " " + fmt.Sprintf(format, a...),
	}
}

type Event = _connector.Event
type EventType = _connector.EventType

// Record represents a record. If an error occurs during the reading or
// validation of the record, the Err field contains the specific error,
// which type implements the ValidationError interface of apis.
type Record = _connector.Record

// Records is the iterator interface used to iterate over the records read from
// apps, databases, and files.
type Records interface {

	// Close closes the iterator. It is automatically called by the For method
	// before returning. Close is idempotent and does not impact the result of Err.
	Close() error

	// Err returns any error encountered during iteration, excluding errors returned
	// by the yield function, which may have occurred after an explicit or implicit
	// Close.
	Err() error

	// For calls the yield function for each record (r) in the sequence. If yield
	// returns an error, For stops and returns the error. After For completes, it
	// is also necessary to check the result of Err for any potential errors.
	For(yield func(Record) error) error
}

// AckFunc is the function called when a write of one or more records
// terminates. The parameter err represents the error that occurred while
// writing the records, if any, and ids represents the identifiers of the
// records.
//
// TODO: ids should be []string if ids represents the data warehouse identifiers.
type AckFunc func(err error, ids []int)

// Writer is the interface implemented by app, database, and file connectors to
// write records.
type Writer interface {

	// Close closes the writer. For non-committable writers, it ensures the
	// completion of all pending or ongoing write operations. In the event of a
	// canceled context, it interrupts ongoing writes, discards pending ones, and
	// returns. For committable writers, it discards all writes, including those
	// already executed.
	//
	// If the writer is already closed, it does nothing and returns immediately.
	Close(ctx context.Context) error

	// Write writes a record. Typically, Write returns immediately, deferring the
	// actual write operation to a later time. If it returns false, no further Write
	// operations can be performed, and a call to Close will return an error.
	//
	// If the record is successfully written, the ack function is invoked with a nil
	// error and the record's ID as arguments. If writing the record fails, the ack
	// function is invoked with a non-nil error and the record's ID as arguments.
	// The ack function is invoked even if Write returns false.
	//
	// It panics if called on a closed writer.
	Write(ctx context.Context, gid int, record Record) bool
}

// CommittableWriter is the interface implemented by writers that support
// committable writes.
type CommittableWriter interface {

	// Commit commits executed, ongoing, and pending write operations, ensuring
	// their completion. If the commit fails, no records are written.
	// Commit always closes the writer.
	//
	// It panics if called on a closed writer.
	Commit(ctx context.Context) error
}

// TimestampColumn represents the timestamp column passed to the
// (*File).ReadFunc method.
type TimestampColumn struct {
	Name   string
	Format string
}

// An InvalidSettingsError is returned by UI-related functions when the settings
// passed as an argument are not valid.
type InvalidSettingsError struct {
	Msg string
}

func (err InvalidSettingsError) Error() string {
	return err.Msg
}

var (
	ErrEventNotExist       = errors.New("user interface event does not exist")
	ErrEventTypeNotExist   = errors.New("event type does not exist")
	ErrNoColumns           = errors.New("file has no columns")
	ErrNoUserInterface     = errors.New("connector has no user interface")
	ErrNoWebhooks          = errors.New("app has no webhooks")
	ErrSheetNotExist       = errors.New("sheet does not exist")
	ErrWebhookUnauthorized = errors.New("webhook request was not unauthorized")
)

// Connectors allows to interact with the apps, databases, files, mobile,
// servers, storage, streams, and websites connectors.
type Connectors struct {
	state *state.State
	http  *httpclient.HTTP
}

// New returns a new *Connectors value.
func New(db *postgres.DB, state *state.State) *Connectors {
	h := httpclient.New(db, state, http.DefaultTransport)
	h.SetTrace(os.Stdout)
	return &Connectors{state: state, http: h}
}

// Authorization represents a granted OAuth authorization.
type Authorization struct {
	ResourceCode string    // code of the resource.
	AccessToken  string    // access token.
	RefreshToken string    // refresh token.
	ExpiresIn    time.Time // expire time of the access token.
}

// GrantAuthorization grants an OAuth authorization from an app connector
// provided an authorization code and a redirection URI.
func (connectors *Connectors) GrantAuthorization(ctx context.Context, connector *state.Connector, code, redirectionURI string, region state.PrivacyRegion) (*Authorization, error) {
	accessToken, refreshToken, expiresIn, err := connectors.http.GrantAuthorization(ctx, connector.OAuth, code, redirectionURI)
	if err != nil {
		return nil, err
	}
	cc, err := _connector.RegisteredApp(connector.Name).New(&_connector.AppConfig{
		HTTPClient: connectors.http.Client(connector.OAuth.ClientSecret, accessToken),
		Region:     _connector.PrivacyRegion(region),
	})
	if err != nil {
		return nil, err
	}
	resource, err := cc.Resource(ctx)
	if err != nil {
		return nil, err
	}
	authorization := &Authorization{
		ResourceCode: resource,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}
	return authorization, nil
}

// AuthorizationEndpoint returns an OAuth authorization endpoint URI for the
// provided app connector, used to redirects to the consent page of its OAuth
// provider. This page requests explicit permissions for the required scopes.
// After that, the provider redirects to the URI specified by redirectionURI.
//
// After acquiring the authorization code, call GrantAuthorization to get the
// resulting resource code, access token, refresh token and expiration time.
//
// Panics if the connector does not support OAuth.
func (connectors *Connectors) AuthorizationEndpoint(connector *state.Connector, redirectionURI string) (string, error) {
	oauth := connector.OAuth
	var b strings.Builder
	b.WriteString(oauth.AuthURL)
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {oauth.ClientID},
		"redirect_uri":  {redirectionURI},
		"state":         {"state"},
	}
	if len(oauth.Scopes) > 0 {
		v.Set("scope", strings.Join(oauth.Scopes, " "))
	}
	if strings.Contains(oauth.AuthURL, "?") {
		b.WriteByte('&')
	} else {
		b.WriteByte('?')
	}
	b.WriteString(v.Encode())
	return b.String(), nil
}

// ReceivePerConnectionWebhook receives a per connection webhook request and
// returns its payloads. The context is the request's context.
//
// It returns the ErrNoWebhooks error if the connection is not an app,
// or it does not support per connection webhooks. It returns the
// ErrWebhookUnauthorized error if the request was not authorized.
func (connectors *Connectors) ReceivePerConnectionWebhook(connection *state.Connection, req *http.Request) ([]WebhookPayload, error) {
	connector := connection.Connector()
	if connector.WebhooksPer != state.WebhooksPerConnection {
		return nil, ErrNoWebhooks
	}
	var resourceID int
	var resourceCode string
	if r, ok := connection.Resource(); ok {
		resourceID = r.ID
		resourceCode = r.Code
	}
	inner, err := _connector.RegisteredApp(connector.Name).New(&_connector.AppConfig{
		Role:        _connector.Role(connection.Role),
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		Resource:    resourceCode,
		HTTPClient:  connectors.http.ConnectionClient(connection.ID),
		Region:      _connector.PrivacyRegion(connection.Workspace().PrivacyRegion),
		WebhookURL:  webhookURL(connection, resourceID),
	})
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(webhookReceiver); ok {
		payload, err := inner.ReceiveWebhook(req)
		if err != nil {
			if err == _connector.ErrWebhookUnauthorized {
				err = ErrWebhookUnauthorized
			}
			return nil, err
		}
		return payload, nil
	}
	return nil, ErrNoWebhooks
}

// ReceivePerConnectorWebhook receives a per connector webhook request and
// returns its payloads. The context is the request's context.
//
// It returns the ErrNoWebhooks error if the connector is not an app,
// or it does not support per connector webhooks. It returns the
// ErrWebhookUnauthorized error if the request was not authorized.
func (connectors *Connectors) ReceivePerConnectorWebhook(connector *state.Connector, req *http.Request) ([]WebhookPayload, error) {
	if connector.WebhooksPer != state.WebhooksPerConnector {
		return nil, ErrNoWebhooks
	}
	inner, err := _connector.RegisteredApp(connector.Name).New(&_connector.AppConfig{
		Role: _connector.Source,
	})
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(webhookReceiver); ok {
		payload, err := inner.ReceiveWebhook(req)
		if err != nil {
			if err == _connector.ErrWebhookUnauthorized {
				err = ErrWebhookUnauthorized
			}
			return nil, err
		}
		return payload, nil
	}
	return nil, ErrNoWebhooks
}

// ReceivePerResourceWebhook receives a per resource webhook request and returns
// its payloads. The context is the request's context.
//
// It returns the ErrNoWebhooks error if the connector of the resource
// is not an app, or it does not support per resource webhooks. It returns the
// ErrWebhookUnauthorized error if the request was not authorized.
func (connectors *Connectors) ReceivePerResourceWebhook(resource *state.Resource, req *http.Request) ([]WebhookPayload, error) {
	connector := resource.Connector()
	if connector.WebhooksPer != state.WebhooksPerResource {
		return nil, ErrNoWebhooks
	}
	config := &_connector.AppConfig{
		Role:     _connector.Source,
		Resource: resource.Code,
	}
	if connector.OAuth != nil {
		config.HTTPClient = connectors.http.Client(connector.OAuth.ClientSecret, resource.AccessToken)
	}
	config.Region = _connector.PrivacyRegion(resource.Workspace().PrivacyRegion)
	inner, err := _connector.RegisteredApp(connector.Name).New(config)
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(webhookReceiver); ok {
		payload, err := inner.ReceiveWebhook(req)
		if err != nil {
			if err == _connector.ErrWebhookUnauthorized {
				err = ErrWebhookUnauthorized
			}
			return nil, err
		}
		return payload, nil
	}
	return nil, ErrNoWebhooks
}

// yieldError is an error returned by the yield function of Records when
// iterating over records.
type yieldError struct {
	err error
}

func (err yieldError) Error() string {
	return err.err.Error()
}

// checkConformity checks whether the schema t1 conforms to the new schema t2
// and returns a *SchemaError error if it does not conform.
// It panics if a schema is not valid.
func checkConformity(name string, t1, t2 types.Type) error {
	if t1.EqualTo(t2) {
		return nil
	}
	pt1 := t1.Kind()
	pt2 := t2.Kind()
	if pt1 != pt2 {
		if pt1 == types.IntKind && pt2 == types.UintKind || pt1 == types.UintKind && pt2 == types.IntKind {
			return nil
		}
		return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
	}
	switch pt1 {
	case types.ArrayKind:
		return checkConformity(name, t1.Elem(), t2.Elem())
	case types.ObjectKind:
		for _, p1 := range t1.Properties() {
			path := p1.Name
			if name != "" {
				path = name + "." + path
			}
			p2, ok := t2.Property(p1.Name)
			if !ok {
				return &SchemaError{Msg: fmt.Sprintf(`%q property no longer exists`, path)}
			}
			err := checkConformity(path, p1.Type, p2.Type)
			if err != nil {
				return err
			}
		}
	case types.MapKind:
		return checkConformity(name, t1.Elem(), t2.Elem())
	}
	return nil
}

// maxSettingsLen is the maximum length of settings in runes.
// Keep in sync with the events.maxSettingsLen constant.
const maxSettingsLen = 10_000

type webhookReceiver interface {
	ReceiveWebhook(*http.Request) ([]WebhookPayload, error)
}

// businessIDFromSchema returns the Business ID from the given schema, if found
// and its type is compatible, otherwise returns the zero Property and an error.
func businessIDFromSchema(schema types.Type, businessIDName string) (types.Property, error) {
	p, ok := schema.Property(businessIDName)
	if !ok {
		return types.Property{}, fmt.Errorf("the Business ID property %q not found in schema", businessIDName)
	}
	if !supportedTypeForBusinessID(p.Type) {
		return types.Property{}, fmt.Errorf("the Business ID property %q has an unsupported type %s", businessIDName, p.Type)
	}
	return p, nil
}

// businessIDToString returns a string representation of the Business ID value.
// If value cannot be represented as a valid Business ID value, an error is
// returned.
func businessIDToString(value any) (string, error) {
	var s string
	switch src := value.(type) {
	case int: // Int(n).
		s = strconv.Itoa(src)
	case uint: // Uint(n).
		s = strconv.Itoa(int(src))
	case string: // Text, JSON String.
		s = src
	case decimal.Decimal: // Decimal.
		s = src.String()
	case json.Number: // JSON Number
		s = src.String()
	case float64:
		s = fmt.Sprint(src)
	default:
		return "", fmt.Errorf("unexpected Business ID value with type %T", src)
	}
	if utf8.RuneCountInString(s) > 40 {
		return "", fmt.Errorf("the Business ID value is longer than 40 runes")
	}
	return s, nil
}

// setActionSettingsFunc returns a connector.SetSettingsFunc function that sets
// the settings for the action.
func setActionSettingsFunc(st *state.State, a *state.Action) _connector.SetSettingsFunc {
	return func(ctx context.Context, settings []byte) error {
		return setActionSettings(ctx, st, a.ID, settings)
	}
}

// setSettingsFunc returns a connector.SetSettingsFunc function that sets the
// settings for the connection.
func setConnectionSettingsFunc(st *state.State, c *state.Connection) _connector.SetSettingsFunc {
	return func(ctx context.Context, settings []byte) error {
		return setConnectionSettings(ctx, st, c.ID, settings)
	}
}

// setActionSettings sets the settings of the provided action.
func setActionSettings(ctx context.Context, st *state.State, action int, settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetActionSettings{
		Action:   action,
		Settings: settings,
	}
	err := st.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE actions SET settings = $1 WHERE id = $2", n.Settings, n.Action)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// setConnectionSettings sets the settings of the provided connection.
func setConnectionSettings(ctx context.Context, st *state.State, connection int, settings []byte) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetConnectionSettings{
		Connection: connection,
		Settings:   settings,
	}
	err := st.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// supportedTypeForBusinessID reports whether the type t is supported as a
// Business ID type.
func supportedTypeForBusinessID(t types.Type) bool {
	switch t.Kind() {
	case types.IntKind,
		types.UintKind,
		types.FloatKind,
		types.JSONKind,
		types.TextKind:
		return true
	case types.DecimalKind:
		return t.Scale() == 0
	default:
		return false
	}
}

// validateTimestamp validates the timestamp t, returning an error if it is not
// valid.
func validateTimestamp(t time.Time) error {
	if t.IsZero() {
		return errors.New("timestamp cannot be the zero time instant (January 1, year 1, 00:00:00 UTC)")
	}
	if y := t.Year(); y < 1 || y > 9999 {
		return fmt.Errorf("timestamp year %d out of range [1,9999]", y)
	}
	now := time.Now().UTC()
	if t.After(now) {
		return errors.New("timestamp cannot be in the future")
	}
	return nil
}

// webhookURL returns the URL of the webhook for the provided connection and
// resource.
// If the connector does not support webhooks, it returns an empty string.
func webhookURL(connection *state.Connection, resource int) string {
	connector := connection.Connector()
	u := "https://localhost:9090/webhook/"
	switch connector.WebhooksPer {
	case state.WebhooksPerNone:
		return ""
	case state.WebhooksPerConnection:
		return u + "s/" + strconv.Itoa(connection.ID) + "/"
	case state.WebhooksPerConnector:
		return u + "c/" + strconv.Itoa(connector.ID) + "/"
	case state.WebhooksPerResource:
		return u + "r/" + strconv.Itoa(resource) + "/"
	}
	panic("unexpected webhooksPer value")
}
