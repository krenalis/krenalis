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
)

type Cursor = _connector.Cursor
type Event = _connector.Event
type EventType = _connector.EventType
type Record = _connector.Record

// WriteFunc represents the function passed to the (*File).ReadFunc method to
// read file records.
type WriteFunc func(Record) error

// TimestampColumn represents the timestamp column passed to the
// (*File).ReadFunc method.
type TimestampColumn struct {
	Name   string
	Format string
}

// An InvalidPathError is returned when a path name is not valid.
type InvalidPathError = _connector.InvalidPathError

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
	ErrNoStorage           = errors.New("file has no storage")
	ErrNoUserInterface     = errors.New("connector has no user interface")
	ErrNoWebhooks          = errors.New("app has no webhooks")
	ErrWebhookUnauthorized = errors.New("webhook request was not unauthorized")
)

// Connectors allows to interact with the apps, databases, files, mobile,
// servers, storage, streams, and websites connectors.
type Connectors struct {
	db    *postgres.DB
	state *state.State
	http  *httpclient.HTTP
}

// New returns a new *Connectors value.
func New(db *postgres.DB, state *state.State) *Connectors {
	h := httpclient.New(db, state, http.DefaultTransport)
	h.SetTrace(os.Stdout)
	return &Connectors{db: db, state: state, http: h}
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
		SetSettings: setSettingsFunc(connectors.db, connection),
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

// checkConformity checks whether the schema t1 conforms to the new schema t2
// and returns a SchemaError error if it does not conform.
func checkConformity(name string, t1, t2 types.Type) error {
	if t1.EqualTo(t2) {
		return nil
	}
	pt1 := t1.PhysicalType()
	pt2 := t2.PhysicalType()
	if pt1 != pt2 {
		if pt1 == types.PtInt && pt2 == types.PtUint || pt1 == types.PtUint && pt2 == types.PtInt {
			return nil
		}
		return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
	}
	switch pt1 {
	case types.PtArray:
		return checkConformity(name, t1.Elem(), t2.Elem())
	case types.PtObject:
		for _, p1 := range t1.Properties() {
			path := p1.Name
			if name != "" {
				path = name + "." + path
			}
			p2, ok := t2.Property(p1.Name)
			if !ok {
				return &SchemaError{Msg: fmt.Sprintf(`"%s" property no longer exists`, path)}
			}
			err := checkConformity(path, p1.Type, p2.Type)
			if err != nil {
				return err
			}
		}
	case types.PtMap:
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

// setSettingsFunc returns a connector.SetSettingsFunc function that sets the
// settings for the connection.
func setSettingsFunc(db *postgres.DB, c *state.Connection) _connector.SetSettingsFunc {
	return func(ctx context.Context, settings []byte) error {
		return setSettings(ctx, db, c.ID, settings)
	}
}

// setSettings sets the settings of the provided connection.
// It is a copy of the apis.setSettings function, so keep in sync.
func setSettings(ctx context.Context, db *postgres.DB, connection int, settings []byte) error {
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
	err := db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
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
