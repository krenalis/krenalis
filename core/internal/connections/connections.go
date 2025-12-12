// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package connections provides the interface to interact with API, database,
// file storage, and message broker connections, and to file pipelines.
package connections

import (
	"context"
	"fmt"
	"iter"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections/httpclient"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/itchyny/timefmt-go"
	"github.com/relvacode/iso8601"
)

// maxSettingsLen is the maximum length of settings in runes.
const maxSettingsLen = 10_000

// AckFunc is the function invoked when a write of one or more records
// terminates. ids represents the acknowledgment identifiers and err is the
// error that occurred while writing the records, if any.
type AckFunc func(ids []string, err error)

// Authorization represents a granted OAuth authorization.
type Authorization struct {
	AccountCode  string    // code of the account.
	AccessToken  string    // access token.
	RefreshToken string    // refresh token.
	ExpiresIn    time.Time // expiration time of the access token.
}

var (
	ErrNoColumnsFound = errors.New("file has no columns")
	ErrNoWebhooks     = errors.New("API has no webhooks")
)

// LastChangeTimeColumn represents the last change time column passed to the
// (*File).ReadFunc method.
type LastChangeTimeColumn struct {
	Name   string
	Format string
}

// PlaceholderError is an error representing a placeholder error.
type PlaceholderError string

// Error implements the error interface for PlaceholderError.
func (e *PlaceholderError) Error() string {
	return string(*e)
}

// placeholderErrorf formats according to a format specifier and returns a
// *PlaceholderError value.
func placeholderErrorf(format string, a ...any) error {
	err := PlaceholderError(fmt.Sprintf(format, a...))
	return &err
}

// UnavailableError represents an error that occurs when a connector cannot
// fulfill a request due to an internal error.
type UnavailableError struct {
	Err error
}

func (err *UnavailableError) Error() string {
	return err.Err.Error()
}

// PlaceholderReplacer is the type of functions accepted by ReplacePlaceholders.
// name is the placeholder's name and the returned values are the replacement
// value (if any, otherwise the empty string) and a boolean indicating whether
// the placeholder is allowed.
type PlaceholderReplacer func(name string) (string, bool)

// Records is the iterator interface used to iterate over the records read from
// apps, databases, and files.
type Records interface {

	// All returns an iterator to iterate over the records. After All completes, it
	// is also necessary to check the result of Err for any potential errors.
	All(ctx context.Context) iter.Seq[Record]

	// Close closes the iterator. It is automatically called by the For method
	// before returning. Close is idempotent and does not impact the result of Err.
	Close() error

	// Err returns any error encountered during iteration, excluding errors returned
	// by the yield function, which may have occurred after an explicit or implicit
	// Close.
	Err() error

	// Last reports whether the last record has been read.
	Last() bool
}

type EventType = connectors.EventType

// Record represents a record. If an error occurs during the reading or
// validation of the record, the Err field contains the specific error.
type Record struct {
	ID             string         // Identifier.
	Attributes     map[string]any // Attributes.
	LastChangeTime time.Time      // Last modification time, in UTC.

	// Associations contains the identifiers of the user's groups or the group's users.
	// It is not significant if it is nil.
	Associations []string

	// Err reports an error that occurred while reading the record. If Err is not
	// nil, only the ID field is significant. For validation errors, the error type
	// implements core.ValidationError interface.
	Err error
}

// Writer is the interface implemented by API, database, and file connectors to
// write records.
type Writer interface {

	// Close terminates the writer, ensuring that all records are processed before
	// returning, unless the provided context is canceled.
	// If processing all records fails, an error is returned.
	//
	// If the writer is already closed, it does nothing and returns immediately.
	Close(ctx context.Context) error

	// Write writes a record. Typically, Write returns immediately, deferring the
	// actual write operation to a later time. record must contain at least one
	// attribute.
	//
	// If it returns false, no further Write operations can be performed, and a call
	// to Close will return the occurred error.
	//
	// It panics if called on a closed writer.
	Write(ctx context.Context, id string, attributes map[string]any) bool
}

// Connections provides access to API, database, file, file storage, SDK, and
// message broker connections.
type Connections struct {
	state *state.State
	http  *httpclient.HTTP
}

// New returns a new *Connections value.
func New(state *state.State) *Connections {
	h := httpclient.New(state, http.DefaultTransport)
	return &Connections{state: state, http: h}
}

// AuthorizationEndpoint returns the OAuth authorization endpoint URI for the
// provided API connector. This URI is used to redirect users to the OAuth
// provider's consent page, where they can grant permissions for the scopes of
// the specified role. After granting permissions, the provider redirects the
// user to the URI specified by redirectionURI.
//
// After obtaining the authorization code, call GrantAuthorization to retrieve
// the account code, access token, refresh token, and expiration time.
//
// If the connector is not configured for OAuth (that is, ClientID or
// ClientSecret is empty), it returns an *UnavailableError.
//
// It panics if the connector does not support OAuth.
func (c *Connections) AuthorizationEndpoint(connector *state.Connector, role state.Role, redirectionURI string) (string, error) {
	oauth := connector.OAuth
	if oauth.ClientID == "" || oauth.ClientSecret == "" {
		return "", &UnavailableError{Err: fmt.Errorf("%s OAuth authentication is not configured. Please check the environment variables passed to Meergo", connector.Code)}
	}
	var b strings.Builder
	b.WriteString(oauth.AuthURL)
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {oauth.ClientID},
		"redirect_uri":  {redirectionURI},
		"state":         {"state"},
	}
	scopes := oauth.SourceScopes
	if role == state.Destination {
		scopes = oauth.DestinationScopes
	}
	if len(scopes) > 0 {
		v.Set("scope", strings.Join(scopes, " "))
	}
	if strings.Contains(oauth.AuthURL, "?") {
		b.WriteByte('&')
	} else {
		b.WriteByte('?')
	}
	b.WriteString(v.Encode())
	return b.String(), nil
}

// GrantAuthorization grants an OAuth authorization for an API connector, using
// the provided authorization code and redirection URI.
//
// This method can only be called on a connector that implements OAuth.
func (c *Connections) GrantAuthorization(ctx context.Context, connector *state.Connector, code, redirectionURI string) (*Authorization, error) {
	accessToken, refreshToken, expiresIn, err := c.http.GrantAuthorization(ctx, connector, code, redirectionURI)
	if err != nil {
		return nil, err
	}
	api, err := connectors.RegisteredAPI(connector.Code).New(&connectors.APIEnv{
		HTTPClient: c.http.ConnectorClient(connector, connector.OAuth.ClientSecret, accessToken),
	})
	if err != nil {
		return nil, connectorError(err)
	}
	account, err := api.(apiOAuthConnector).OAuthAccount(ctx)
	if err != nil {
		return nil, connectorError(err)
	}
	authorization := &Authorization{
		AccountCode:  account,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}
	return authorization, nil
}

// TODO(marco): implement webhooks
//// ReceivePerAccountWebhook receives a per account webhook request and returns
//// its payloads. The context is the request's context.
////
//// If the connector of the account is not an API or does not support per account
//// webhooks, it returns the ErrNoWebhooks error. If the request is not
//// authorized, it returns the connectors.ErrWebhookUnauthorized error.
//func (connectors *Connections) ReceivePerAccountWebhook(account *state.Account, req *http.Request) ([]connectors.WebhookPayload, error) {
//	connector := account.Connector()
//	if connector.WebhooksPer != state.WebhooksPerAccount {
//		return nil, ErrNoWebhooks
//	}
//	config := &connectors.APIEnv{
//		OAuthAccount: account.Code,
//	}
//	if connector.OAuth != nil {
//		config.HTTPClient = connectors.http.Client(connector.OAuth.ClientSecret, account.AccessToken, connector.RetryPolicy)
//	}
//	inner, err := connectors.RegisteredAPI(connector.Name).New(config)
//	if err != nil {
//		return nil, err
//	}
//	payload, err := inner.(webhookConnector).ReceiveWebhook(req, connectors.Both)
//	if err != nil {
//		return nil, err
//	}
//	return payload, nil
//}

// TODO(marco): implement webhooks
//// ReceivePerConnectionWebhook receives a per connection webhook request and
//// returns its payloads. The context is the request's context.
////
//// if the connection is not an API, or it does not support per connection
//// webhooks, it returns the ErrNoWebhooks error. If the request is not
//// authorized, it returns the connectors.ErrWebhookUnauthorized error.
//func (connectors *Connections) ReceivePerConnectionWebhook(connection *state.Connection, req *http.Request) ([]connectors.WebhookPayload, error) {
//	connector := connection.Connector()
//	if connector.WebhooksPer != state.WebhooksPerConnection {
//		return nil, ErrNoWebhooks
//	}
//	var accountID int
//	var accountCode string
//	if a, ok := connection.Account(); ok {
//		accountID = a.ID
//		accountCode = a.Code
//	}
//	inner, err := connectors.RegisteredAPI(connector.Name).New(&connectors.APIEnv{
//		Settings:     connection.Settings,
//		SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
//		OAuthAccount: accountCode,
//		HTTPClient:   connectors.http.ConnectionClient(connection.ID),
//		WebhookURL:   webhookURL(connection, accountID),
//	})
//	if err != nil {
//		return nil, err
//	}
//	payload, err := inner.(webhookConnector).ReceiveWebhook(req, connectors.Role(connection.Role))
//	if err != nil {
//		return nil, err
//	}
//	return payload, nil
//}

// TODO(marco): implement webhooks
//// ReceivePerConnectorWebhook receives a per connector webhook request and
//// returns its payloads. The context is the request's context.
////
//// If the connector is not an API, or it does not support per connector
//// webhooks, it returns the ErrNoWebhooks error. If the request was not
//// authorized, it returns the connectors.ErrWebhookUnauthorized error.
//func (connectors *Connections) ReceivePerConnectorWebhook(connector *state.Connector, req *http.Request) ([]connectors.WebhookPayload, error) {
//	if connector.WebhooksPer != state.WebhooksPerConnector {
//		return nil, ErrNoWebhooks
//	}
//	inner, err := connectors.RegisteredAPI(connector.Name).New(&connectors.APIEnv{})
//	if err != nil {
//		return nil, err
//	}
//	payload, err := inner.(webhookConnector).ReceiveWebhook(req, connectors.Both)
//	if err != nil {
//		return nil, err
//	}
//	return payload, nil
//}

// ReplacePlaceholders replaces the placeholders in s by calling the non-nil
// function f with the name of each placeholder. It returns the string with the
// placeholders replaced. In case of error it returns an empty string and a
// *PlaceholderError.
func ReplacePlaceholders(s string, f PlaceholderReplacer) (string, error) {
	var b strings.Builder
	var name string
	var value string
	var ok bool
	for {
		i := strings.Index(s, "${")
		if i < 0 {
			break
		}
		b.WriteString(s[:i])
		s = s[i+2:]
		i = strings.IndexByte(s, '}')
		if i < 0 {
			return "", placeholderErrorf("a placeholder is not closed")
		}
		name, s = strings.TrimSpace(s[:i]), s[i+1:]
		if strings.Contains(name, "${") {
			return "", placeholderErrorf("a placeholder is not closed")
		}
		value, ok = f(name)
		if !ok {
			return "", placeholderErrorf("placeholder %q does not exist", name)
		}
		b.WriteString(value)
	}
	if b.Len() == 0 {
		return s, nil
	}
	b.WriteString(s)
	return b.String(), nil
}

// connectorError checks the type of err and returns it unwrapped if it is one
// of the expected errors from a connector. Otherwise, it wraps err in an
// *UnavailableError before returning.
func connectorError(err error) error {
	switch err {
	case nil:
	case connectors.ErrEventTypeNotExist:
	case connectors.ErrSheetNotExist:
	case connectors.ErrUIEventNotExist:
	//case connectors.ErrWebhookUnauthorized: // TODO(marco): implement webhooks
	default:
		switch err.(type) {
		case *connectors.InvalidPathError:
		case *connectors.InvalidSettingsError:
		case *UnavailableError:
		default:
			err = &UnavailableError{Err: err}
		}
	}
	return err
}

// formatLastChangeTimeColumn formats a time.Time value using the provided
// format. The Excel format is not allowed here.
//
// format must be a valid change time format; for accepted formats, refer to the
// 'core.validateLastChangeTimeFormat' function.
func formatLastChangeTimeColumn(format string, t time.Time) string {
	switch format {
	case "ISO8601":
		return t.Format(time.RFC3339)
	case "Excel":
		panic("unexpected Excel format")
	default: // any format compatible with strptime, for example '%Y-%m-%d'.
		return timefmt.Format(t, format)
	}
}

// isExcelSimpleFloat reports whether s is a string representing an Excel date or
// datetime encoded as a floating-point number.
func isExcelSimpleFloat(s string) bool {
	if len(s) < 3 {
		return false
	}
	var dot bool
	for i, c := range []byte(s) {
		if c == '.' {
			if dot || i == 0 || i == len(s)-1 {
				return false
			}
			dot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Ordinal returns the ordinal form of n.
func ordinal(n int) string {
	if n >= 11 && n <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	}
	return fmt.Sprintf("%dth", n)
}

// parseIdentityColumn parses the value for the identity column.
func parseIdentityColumn(name string, typ types.Type, value any, layouts *state.TimeLayouts) (string, error) {
	id, err := normalize(name, typ, value, false, layouts)
	if err != nil {
		return "", err
	}
	switch id := id.(type) {
	case nil:
		return "", fmt.Errorf("identity value is null")
	case string:
		if id == "" {
			return "", fmt.Errorf("identity value is empty")
		}
		return id, nil
	case int:
		return strconv.FormatInt(int64(id), 10), nil
	case uint:
		return strconv.FormatUint(uint64(id), 10), nil
	case float64:
		if int(math.Round(id)) == int(id) {
			return strconv.FormatInt(int64(id), 10), nil
		}
	case json.Value:
		switch id.Kind() {
		case json.String:
			s := id.String()
			if s == "" {
				return "", fmt.Errorf("identity value is empty")
			}
			return s, nil
		case json.Number:
			if _, err := id.Int(); err == nil {
				return string(id), nil
			}
		}
	}
	return "", fmt.Errorf("identity value is not a JSON string or JSON integer number")
}

// parseLastChangeTimeColumn parses a last change time column value. If the
// value cannot be parsed or is not valid, it returns an error. If the value is
// valid but nil, and nullable is true, it returns the zero time and a nil
// error.
//
// format must be a valid change time format; for accepted formats, refer to the
// 'core.validateLastChangeTimeFormat' function.
func parseLastChangeTimeColumn(name string, typ types.Type, format string, value any, nullable bool, layouts *state.TimeLayouts) (time.Time, error) {
	v, err := normalize(name, typ, value, nullable, layouts)
	if err != nil {
		return time.Time{}, err
	}
	switch v := v.(type) {
	case nil:
		return time.Time{}, nil
	case time.Time:
		err = validateLastChangeTime(v)
		if err != nil {
			return time.Time{}, err
		}
		return v, nil
	case string:
		t, err := parseLastChangeTimeColumnWithFormat(format, v)
		if err != nil {
			return time.Time{}, err
		}
		err = validateLastChangeTime(t)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	case json.Value:
		if !v.IsString() {
			return time.Time{}, fmt.Errorf("last change time is not a JSON string")
		}
		t, err := parseLastChangeTimeColumnWithFormat(format, v.String())
		if err != nil {
			return time.Time{}, err
		}
		err = validateLastChangeTime(t)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("last change time is not a JSON string")
}

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

// parseLastChangeTimeColumnWithFormat parses a last change time value with
// the given format.
//
// format must be a valid change time format; for accepted formats, refer to the
// 'core.validateLastChangeTimeFormat' function.
func parseLastChangeTimeColumnWithFormat(format, v string) (time.Time, error) {
	switch format {
	case "ISO8601":
		dt, err := iso8601.ParseString(v)
		if err != nil {
			return time.Time{}, fmt.Errorf("last change time does not conform to the ISO8601 format")
		}
		return dt.UTC(), err
	case "Excel":
		if !isExcelSimpleFloat(v) {
			return time.Time{}, errors.New("last change time does not conform to the Excel format")
		}
		// Parse as Excel serial date-time.
		// https://support.microsoft.com/en-us/office/datetime-function-812ad674-f7dd-4f31-9245-e79cfa358a4e
		// https://support.microsoft.com/en-us/office/datevalue-function-df8b07d4-7761-4a93-bc33-b7471bbff252
		days, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return time.Time{}, errors.New("last change time does not conform to the Excel format")
		}
		if days == 60 {
			// 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
			return time.Time{}, errors.New("last change time does not conform to the Excel format")
		}
		if days > 60 {
			days--
		}
		d := time.Duration(days * 24 * 3600 * 1e9)
		t := excelEpoch.Add(d)
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	default: // any format compatible with strptime, for example '%Y-%m-%d'.
		t, err := timefmt.Parse(v, format)
		if err != nil {
			return time.Time{}, fmt.Errorf("last change time does not conform to the %q format", format)
		}
		return t.UTC(), nil
	}
}

// rewriteColumnErrors updates error messages for types.InvalidPropertyNameError
// and types.RepeatedPropertyNameError to enhance clarity. All other errors are
// returned unchanged.
func rewriteColumnErrors(err error) error {
	switch e := err.(type) {
	case types.InvalidPropertyNameError:
		err = fmt.Errorf("the %s column has an invalid name: %q. Column names must start with a letter or underscore [A-Za-z_]"+
			" and subsequently contain only letters, numbers, or underscores [A-Za-z0-9_]", ordinal(e.Index+1), e.Name)
	case types.RepeatedPropertyNameError:
		err = fmt.Errorf("the names of the %s and %s columns are the same: %q", ordinal(e.Index1+1), ordinal(e.Index2+1), e.Name)
	}
	return err
}

// setConnectionSettings sets the settings of the provided connection.
func setConnectionSettings(ctx context.Context, st *state.State, connection int, settings json.Value) error {
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
	err := st.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", n.Settings, n.Connection)
		if err != nil {
			return nil, err
		}
		return n, err
	})
	return err
}

// setConnectionSettingsFunc returns a connectors.SetSettingsFunc that sets the
// settings for the connection.
func setConnectionSettingsFunc(st *state.State, c *state.Connection) connectors.SetSettingsFunc {
	return func(ctx context.Context, settings json.Value) error {
		return setConnectionSettings(ctx, st, c.ID, settings)
	}
}

// setPipelineSettings sets the settings of the provided pipeline.
func setPipelineSettings(ctx context.Context, st *state.State, pipeline int, settings json.Value) error {
	if !utf8.Valid(settings) {
		return errors.New("settings is not valid UTF-8")
	}
	if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetPipelineFormatSettings{
		Pipeline: pipeline,
		Settings: settings,
	}
	err := st.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "UPDATE pipelines SET format_settings = $1 WHERE id = $2", n.Settings, n.Pipeline)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	return err
}

// setPipelineSettingsFunc returns a connector.SetSettingsFunc function that
// sets the settings for the pipeline.
func setPipelineSettingsFunc(st *state.State, p *state.Pipeline) connectors.SetSettingsFunc {
	return func(ctx context.Context, settings json.Value) error {
		return setPipelineSettings(ctx, st, p.ID, settings)
	}
}

// validateLastChangeTime validates the last change time t, returning an error
// if it is before the year 1900 or too far ahead in the future.
func validateLastChangeTime(t time.Time) error {
	if y := t.Year(); y < 1900 {
		return errors.New("last change time is before the year 1900")
	}
	if t.After(time.Now().UTC().Add(5 * time.Minute)) {
		return errors.New("last change time is too far ahead in the future")
	}
	return nil
}

// webhookURL returns the URL of the webhook for the provided connection and
// account.
// If the connector does not support webhooks, it returns an empty string.
//func webhookURL(connection *state.Connection, account int) string {
//	connector := connection.Connector()
//	u := "https://localhost:2022/webhook/"
//	switch connector.WebhooksPer {
//	case state.WebhooksPerNone:
//		return ""
//	case state.WebhooksPerAccount:
//		return u + "a/" + strconv.Itoa(account) + "/"
//	case state.WebhooksPerConnection:
//		return u + "s/" + strconv.Itoa(connection.ID) + "/"
//	case state.WebhooksPerConnector:
//		return u + "c/" + url.PathEscape(connector.Name) + "/"
//	}
//	panic("unexpected webhooksPer value")
//}
