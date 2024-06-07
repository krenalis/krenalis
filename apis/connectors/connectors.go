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
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/connectors/httpclient"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/itchyny/timefmt-go"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
)

// Seq represents a sequence of V values.
type Seq[V any] func(yield func(V) bool)

// SchemaError represents an error with a schema.
type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}

type Event = chichi.Event
type EventType = chichi.EventType

// Record represents a record. If an error occurs during the reading or
// validation of the record, the Err field contains the specific error,
// which type implements the ValidationError interface of apis.
type Record struct {
	ID             string         // Identifier.
	Properties     map[string]any // Properties.
	LastChangeTime time.Time      // Last modification time, in UTC.

	// DisplayedProperty, if any, otherwise the empty string. Cannot be longer than 40
	// runes.
	DisplayedProperty string

	// Associations contains the identifiers of the user's groups or the group's users.
	// It is not significant if it is nil.
	Associations []string

	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// Records is the iterator interface used to iterate over the records read from
// apps, databases, and files.
type Records interface {

	// All returns an iterator to iterate over the records. After All completes, it
	// is also necessary to check the result of Err for any potential errors.
	All(ctx context.Context) Seq[Record]

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

// AckFunc is the function called when a write of one or more records
// terminates. ids represents the ack identifiers, and the parameter err
// represents the error that occurred while writing the records, if any.
type AckFunc func(ids []string, err error)

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
	// If the record is successfully written, the ack function is invoked with
	// the ack ID and a nil error as arguments. If writing the record fails, the
	// ack function is invoked with the ack ID and a non-nil error as arguments.
	// The ack function is invoked even if Write returns false.
	//
	// record must contain at least one property.
	//
	// It panics if called on a closed writer.
	Write(ctx context.Context, id string, properties map[string]any, ackID string) bool
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

// LastChangeTimeProperty represents the lat change time property passed to the
// (*File).ReadFunc method.
type LastChangeTimeProperty struct {
	Name   string
	Format string
}

// An InvalidUIValuesError is returned by UI-related functions when the
// user-entered values passed as an argument are not valid.
type InvalidUIValuesError = chichi.InvalidUIValuesError

var (
	ErrEventTypeNotExist   = chichi.ErrEventTypeNotExist
	ErrNoColumns           = errors.New("file has no columns")
	ErrNoWebhooks          = errors.New("app has no webhooks")
	ErrSheetNotExist       = errors.New("sheet does not exist")
	ErrUIEventNotExist     = chichi.ErrUIEventNotExist
	ErrWebhookUnauthorized = errors.New("webhook request was not unauthorized")
)

// Connectors allows to interact with the apps, databases, files, file storages,
// mobile, streams, and websites connectors.
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
	AccountCode  string    // code of the account.
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
	app, err := chichi.RegisteredApp(connector.Name).New(&chichi.AppConfig{
		HTTPClient: connectors.http.Client(connector.OAuth.ClientSecret, accessToken),
		Region:     chichi.PrivacyRegion(region),
	})
	if err != nil {
		return nil, err
	}
	aa, ok := app.(chichi.AppOAuth)
	if !ok {
		return nil, errors.New("connector does not implement the AppOAuth interface")
	}
	account, err := aa.OAuthAccount(ctx)
	if err != nil {
		return nil, err
	}
	authorization := &Authorization{
		AccountCode:  account,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}
	return authorization, nil
}

// AuthorizationEndpoint returns the OAuth authorization endpoint URI for the
// provided app connector. This URI is used to redirect users to the consent
// page of the OAuth provider, where they can grant explicit permissions for the
// specified role's scopes. After granting permissions, the provider redirects
// the user to the URI specified by redirectionURI.
//
// After acquiring the authorization code, call GrantAuthorization to obtain the
// resulting account code, access token, refresh token, and expiration time.
//
// Panics if the connector does not support OAuth.
func (connectors *Connectors) AuthorizationEndpoint(connector *state.Connector, role state.Role, redirectionURI string) (string, error) {
	oauth := connector.OAuth
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

// ReceivePerAccountWebhook receives a per account webhook request and returns
// its payloads. The context is the request's context.
//
// It returns the ErrNoWebhooks error if the connector of the account
// is not an app, or it does not support per account webhooks. It returns the
// ErrWebhookUnauthorized error if the request was not authorized.
func (connectors *Connectors) ReceivePerAccountWebhook(account *state.Account, req *http.Request) ([]WebhookPayload, error) {
	connector := account.Connector()
	if connector.WebhooksPer != state.WebhooksPerAccount {
		return nil, ErrNoWebhooks
	}
	config := &chichi.AppConfig{
		OAuthAccount: account.Code,
	}
	if connector.OAuth != nil {
		config.HTTPClient = connectors.http.Client(connector.OAuth.ClientSecret, account.AccessToken)
	}
	config.Region = chichi.PrivacyRegion(account.Workspace().PrivacyRegion)
	inner, err := chichi.RegisteredApp(connector.Name).New(config)
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(chichi.Webhooks); ok {
		payload, err := inner.ReceiveWebhook(req, chichi.Both)
		if err != nil {
			if err == chichi.ErrWebhookUnauthorized {
				err = ErrWebhookUnauthorized
			}
			return nil, err
		}
		return payload, nil
	}
	return nil, ErrNoWebhooks
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
	var accountID int
	var accountCode string
	if a, ok := connection.Account(); ok {
		accountID = a.ID
		accountCode = a.Code
	}
	inner, err := chichi.RegisteredApp(connector.Name).New(&chichi.AppConfig{
		Settings:     connection.Settings,
		SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
		OAuthAccount: accountCode,
		HTTPClient:   connectors.http.ConnectionClient(connection.ID),
		Region:       chichi.PrivacyRegion(connection.Workspace().PrivacyRegion),
		WebhookURL:   webhookURL(connection, accountID),
	})
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(chichi.Webhooks); ok {
		payload, err := inner.ReceiveWebhook(req, chichi.Role(connection.Role))
		if err != nil {
			if err == chichi.ErrWebhookUnauthorized {
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
	inner, err := chichi.RegisteredApp(connector.Name).New(&chichi.AppConfig{})
	if err != nil {
		return nil, err
	}
	if inner, ok := inner.(chichi.Webhooks); ok {
		payload, err := inner.ReceiveWebhook(req, chichi.Both)
		if err != nil {
			if err == chichi.ErrWebhookUnauthorized {
				err = ErrWebhookUnauthorized
			}
			return nil, err
		}
		return payload, nil
	}
	return nil, ErrNoWebhooks
}

// PlaceholderError is an error representing a placeholder error.
type PlaceholderError string

// Error implements the interface "error" for PlaceholderError.
func (e PlaceholderError) Error() string {
	return string(e)
}

// PlaceholderReplacer is the type of functions accepted by ReplacePlaceholders,
// where name is the name of the placeholder, and the returned values are the
// value to replace (if any, otherwise the empty string) and a boolean
// indicating if a placeholder with that name is allowed or not.
type PlaceholderReplacer func(name string) (string, bool)

// ReplacePlaceholders replaces the placeholders in s with the values read
// calling the f function (that must be non-nil) with the name of each
// placeholder as argument.
// In case of error, returns "" and a PlaceholderError error.
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
			return "", PlaceholderError("a placeholder is not closed")
		}
		name, s = strings.TrimSpace(s[:i]), s[i+1:]
		if strings.Contains(name, "${") {
			return "", PlaceholderError("a placeholder is not closed")
		}
		value, ok = f(name)
		if !ok {
			return "", PlaceholderError(fmt.Sprintf("placeholder %q does not exist", name))
		}
		b.WriteString(value)
	}
	if b.Len() == 0 {
		return s, nil
	}
	b.WriteString(s)
	return b.String(), nil
}

// checkSchemaAlignment checks whether the schema t1 is aligned with t2 and
// returns a *SchemaError error if it is not aligned.
// It panics if a schema is not valid.
func checkSchemaAlignment(t1, t2 types.Type) error {
	return checkTypeAlignment("", t1, t2)
}

// checkTypeAlignment is called by checkSchemaAlignment to check if t1 is
// aligned with t2.
func checkTypeAlignment(name string, t1, t2 types.Type) error {
	k1 := t1.Kind()
	k2 := t2.Kind()
	switch {
	// Types Int, Uint, Float, Decimal, and Year are aligned.
	case k1 >= types.IntKind && k1 <= types.DecimalKind || k1 == types.YearKind:
		if k2 >= types.IntKind && k2 <= types.DecimalKind || k2 == types.YearKind {
			return nil
		}
	// Types Text and UUID are aligned.
	// Types Text and Inet are aligned.
	case k1 == types.TextKind || k1 == types.InetKind || k1 == types.UUIDKind:
		if k2 == k1 || k2 == types.TextKind {
			return nil
		}
	// An Array type is aligned with another Array type if its item type is aligned with the other item type.
	case k1 == types.ArrayKind:
		if k2 == types.ArrayKind {
			return checkTypeAlignment(name, t1.Elem(), t2.Elem())
		}
	// An Object type is aligned with another Object type if its property names are also present in the other Object
	// and the types of the properties are aligned with the types of the respective properties.
	case k1 == types.ObjectKind:
		if k2 == types.ObjectKind {
			for _, p1 := range t1.Properties() {
				path := p1.Name
				if name != "" {
					path = name + "." + path
				}
				p2, ok := t2.Property(p1.Name)
				if !ok {
					return &SchemaError{Msg: fmt.Sprintf(`%q property no longer exists`, path)}
				}
				err := checkTypeAlignment(path, p1.Type, p2.Type)
				if err != nil {
					return err
				}
			}
			return nil
		}
	// A Map type is aligned with another Map type if its value type is aligned with the other value type.
	case k1 == types.MapKind:
		if k2 == types.MapKind {
			return checkTypeAlignment(name, t1.Elem(), t2.Elem())
		}
	// Apart from the previous cases, if two types have the same kind, they are aligned.
	case k1 == k2:
		return nil
	}
	return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
}

// maxSettingsLen is the maximum length of settings in runes.
// Keep in sync with the events.maxSettingsLen constant.
const maxSettingsLen = 10_000

// displayedPropertyFromSchema returns the displayed property from the given
// schema, if found and its type is compatible, otherwise returns the zero
// Property and an error.
func displayedPropertyFromSchema(schema types.Type, displayedPropertyName string) (types.Property, error) {
	p, ok := schema.Property(displayedPropertyName)
	if !ok {
		return types.Property{}, fmt.Errorf("displayed property %q not found in schema", displayedPropertyName)
	}
	if !supportedTypeForDisplayedProperty(p.Type) {
		return types.Property{}, fmt.Errorf("displayed property %q has an unsupported type %s", displayedPropertyName, p.Type)
	}
	return p, nil
}

// displayedPropertyToString returns a string representation of the displayed property
// value. If value cannot be represented as a valid displayed property value, an
// error is returned.
func displayedPropertyToString(value any) (string, error) {
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
		return "", fmt.Errorf("unexpected displayed property value with type %T", src)
	}
	if utf8.RuneCountInString(s) > 40 {
		return "", fmt.Errorf("the displayed property value is longer than 40 runes")
	}
	return s, nil
}

// isExcelSimpleFloat reports whether s is a string representing a float value
// encoding an Excel date / datetime value.
func isExcelSimpleFloat(s string) bool {
	// NOTE: keep in sync with the function within 'apis/transformers/mappings'.
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

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

// parseTimestamp parses a timestamp with the given format.
//
// Accepted values for format are:
//
//   - "DateTime", to parse timestamps in the format "2006-01-02 15:04:05"
//   - "DateOnly", to parse date-only timestamps in the format "2006-01-02"
//   - "ISO8601", to parse the timestamp as a ISO 8601 timestamp.
//   - "Excel", to parse the timestamp as a string representing a float value
//     stored in a Excel cell representing a date / datetime.
//   - a strptime format, enclosed by single quote characters, compatible with
//     the standard C89 functions strptime/strftime.
//
// NOTE: keep in sync with the function 'apis.validateTimestampFormat'.
func parseTimestamp(format, timestamp string) (time.Time, error) {
	switch format {
	case "DateTime":
		dt, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp has not the format '2006-01-02 15:04:05'")
		}
		return dt.UTC(), nil
	case "DateOnly":
		date, err := time.Parse("2006-01-02", timestamp)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp has not the format '2006-01-02'")
		}
		return date.UTC(), nil
	case "ISO8601":
		dt, err := iso8601.ParseString(timestamp)
		if err != nil {
			return time.Time{}, errors.New("timestamp format is not compatible with ISO 8601")
		}
		return dt.UTC(), err
	case "Excel":
		if !isExcelSimpleFloat(timestamp) {
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		// Parse as Excel serial date-time.
		// https://support.microsoft.com/en-us/office/datetime-function-812ad674-f7dd-4f31-9245-e79cfa358a4e
		// https://support.microsoft.com/en-us/office/datevalue-function-df8b07d4-7761-4a93-bc33-b7471bbff252
		days, err := strconv.ParseFloat(timestamp, 64)
		if err != nil {
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		if days == 60 {
			// 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
			return time.Time{}, errors.New("invalid timestamp for Excel")
		}
		if days > 60 {
			days--
		}
		d := time.Duration(days * 24 * 3600 * 1e9)
		t := excelEpoch.Add(d)
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	default: // a format compatible with strptime, for example: '%Y-%m-%d'.
		f, ok := strings.CutPrefix(format, "'")
		if !ok {
			return time.Time{}, fmt.Errorf("invalid format %q", format)
		}
		f, ok = strings.CutSuffix(f, "'")
		if !ok {
			return time.Time{}, fmt.Errorf("invalid format %q", format)
		}
		t, err := timefmt.Parse(timestamp, f)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}
}

// parseTimestampColumn parses a timestamp column value. If the timestamp cannot
// be parsed or it is not valid, returns an error.
//
// To see a list of accepted format values, see the documentation of
// 'parseTimestamp'.
func parseTimestampColumn(name string, typ types.Type, format string, value any, layouts *state.TimeLayouts) (time.Time, error) {
	timestamp, err := normalize(name, typ, value, false, layouts)
	if err != nil {
		return time.Time{}, err
	}
	switch timestamp := timestamp.(type) {
	case nil:
		return time.Time{}, errors.New("timestamp value is null")
	case time.Time:
		err = validateTimestamp(timestamp)
		if err != nil {
			return time.Time{}, err
		}
		return timestamp, nil
	case string:
		ts, err := parseTimestamp(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		err = validateTimestamp(ts)
		if err != nil {
			return time.Time{}, err
		}
		return ts, nil
	case json.RawMessage:
		var s string
		err := json.Unmarshal(timestamp, &s)
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
		}
		ts, err := parseTimestamp(format, value.(string))
		if err != nil {
			return time.Time{}, fmt.Errorf("timestamp %q does not conform to the %q format", value, format)
		}
		err = validateTimestamp(ts)
		if err != nil {
			return time.Time{}, err
		}
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("timestamp value is not a JSON string")
}

// parseIdentityProperty parses the value for the identity property.
func parseIdentityProperty(name string, typ types.Type, value any, layouts *state.TimeLayouts) (string, error) {
	id, err := normalize(name, typ, value, false, layouts)
	if err != nil {
		return "", err
	}
	switch id := id.(type) {
	case nil:
		return "", fmt.Errorf("identify value is null")
	case int:

		return strconv.FormatInt(int64(id), 10), nil
	case uint:
		return strconv.FormatUint(uint64(id), 10), nil
	case string:
		if id == "" {
			return "", fmt.Errorf("identify value is empty")
		}
		return id, nil
	case float64:
		if int(math.Round(id)) == int(id) {
			return strconv.FormatInt(int64(id), 10), nil
		}
	case json.Number:
		var n int64
		err := json.Unmarshal([]byte(id), &n)
		if err == nil {
			return strconv.FormatInt(n, 10), nil
		}
	case json.RawMessage:
		if id[0] == '"' {
			var s string
			_ = json.Unmarshal(id, &s)
			if s == "" {
				return "", fmt.Errorf("identify value is empty")
			}
			return s, nil
		} else {
			var n int64
			err := json.Unmarshal(id, &n)
			if err == nil {
				return strconv.FormatInt(n, 10), nil
			}
		}
	}
	return "", fmt.Errorf("identify value is not a JSON string or JSON integer number")
}

// setActionSettingsFunc returns a connector.SetSettingsFunc function that sets
// the settings for the action.
func setActionSettingsFunc(st *state.State, a *state.Action) chichi.SetSettingsFunc {
	return func(ctx context.Context, settings []byte) error {
		return setActionSettings(ctx, st, a.ID, settings)
	}
}

// setSettingsFunc returns a connector.SetSettingsFunc function that sets the
// settings for the connection.
func setConnectionSettingsFunc(st *state.State, c *state.Connection) chichi.SetSettingsFunc {
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

// supportedTypeForDisplayedProperty reports whether the type t is supported as
// a displayed property type.
func supportedTypeForDisplayedProperty(t types.Type) bool {
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
// account.
// If the connector does not support webhooks, it returns an empty string.
func webhookURL(connection *state.Connection, account int) string {
	connector := connection.Connector()
	u := "https://localhost:9090/webhook/"
	switch connector.WebhooksPer {
	case state.WebhooksPerNone:
		return ""
	case state.WebhooksPerAccount:
		return u + "a/" + strconv.Itoa(account) + "/"
	case state.WebhooksPerConnection:
		return u + "s/" + strconv.Itoa(connection.ID) + "/"
	case state.WebhooksPerConnector:
		return u + "c/" + url.PathEscape(connector.Name) + "/"
	}
	panic("unexpected webhooksPer value")
}
