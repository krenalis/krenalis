// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package activecampaign provides a connector for ActiveCampaign.
// (https://developers.activecampaign.com/)
//
// ActiveCampaign is a trademark of ActiveCampaign, LLC.
// This connector is not affiliated with or endorsed by ActiveCampaign, LLC.
package activecampaign

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

//go:embed documentation/destination/overview.md
var destinationOverview string

const (
	// ActiveCampaign collection endpoints typically allow at most 100 records.
	// https://developers.activecampaign.com/reference/pagination
	contactsPageLimit = 100

	// Bound successful JSON responses before decoding them. The selected
	// Contact endpoint returns at most contactsPageLimit records per request.
	maxResponseBodySize = 8 << 20
)

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "activecampaign",
		Label:      "ActiveCampaign",
		Categories: connectors.CategorySaaS,
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export profiles as contacts to ActiveCampaign",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.ApplicationTerms{
			User:   "Contact",
			Users:  "Contacts",
			UserID: "Contact ID",
		},
		EndpointGroups: []connectors.EndpointGroup{{
			// ActiveCampaign documents a limit of five requests per second per
			// account. Krenalis limiters are per connection, so separate
			// connections using the same account still share the provider quota.
			// https://developers.activecampaign.com/reference/rate-limits
			RateLimit: connectors.RateLimit{RequestsPerSecond: 5, Burst: 5},
		}},
	}, New)
}

// New returns a new connector instance for ActiveCampaign.
func New(env *connectors.ApplicationEnv) (*ActiveCampaign, error) {
	return &ActiveCampaign{env: env}, nil
}

// ActiveCampaign is an ActiveCampaign application connector.
type ActiveCampaign struct {
	env *connectors.ApplicationEnv
}

var _ connectors.RecordUpserter = (*ActiveCampaign)(nil)

type innerSettings struct {
	APIURL   string `json:"apiURL"`
	APIToken string `json:"apiToken"`
}

type contact struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"cdate"`
	UpdatedAt string `json:"udate"`
}

// RecordSchema returns the Contact schema for the specified role.
func (ac *ActiveCampaign) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	if err := ctx.Err(); err != nil {
		return types.Type{}, err
	}
	if target != connectors.TargetUser {
		return types.Type{}, errors.New("ActiveCampaign supports only the user target")
	}
	if role != connectors.Source && role != connectors.Destination {
		return types.Type{}, errors.New("ActiveCampaign requires a source or destination role")
	}

	schema := types.Object([]types.Property{
		{
			Name:           "email",
			Type:           types.String(),
			CreateRequired: true,
			Description:    "Email address; used as the identity when creating or synchronizing a contact",
		},
		{Name: "firstName", Type: types.String(), Description: "First name"},
		{Name: "lastName", Type: types.String(), Description: "Last name"},
		{Name: "phone", Type: types.String(), Description: "Phone number"},
	})
	return types.AsRole(schema, types.Role(role)), nil
}

// Records returns ActiveCampaign Contacts in ascending Contact ID order.
func (ac *ActiveCampaign) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	if target != connectors.TargetUser {
		return nil, "", errors.New("ActiveCampaign supports only the user target")
	}

	settings, err := ac.loadSettings(ctx)
	if err != nil {
		return nil, "", err
	}

	currentCursor := cursor
	var currentID uint64
	if currentCursor != "" {
		currentID, err = parseContactID(currentCursor)
		if err != nil {
			return nil, "", errors.New("invalid ActiveCampaign cursor")
		}
	}

	for {
		// ActiveCampaign recommends ID-based pagination for large Contact sets.
		// https://developers.activecampaign.com/reference/list-all-contacts
		query := url.Values{
			"limit":      {strconv.Itoa(contactsPageLimit)},
			"orders[id]": {"ASC"},
		}
		if currentCursor != "" {
			query.Set("id_greater", currentCursor)
		}

		var response struct {
			Contacts []contact `json:"contacts"`
			Meta     struct {
				Total json.Value `json:"total"`
			} `json:"meta"`
		}
		err = ac.callWithSettings(ctx, settings, http.MethodGet, "/contacts?"+query.Encode(), nil, http.StatusOK, &response)
		if err != nil {
			return nil, "", err
		}
		if len(response.Contacts) == 0 {
			return nil, "", io.EOF
		}

		nextCursor := response.Contacts[len(response.Contacts)-1].ID
		nextID, err := parseContactID(nextCursor)
		if err != nil {
			return nil, "", errors.New("ActiveCampaign returned a contact with an invalid id")
		}
		if nextID <= currentID {
			return nil, "", errors.New("ActiveCampaign contact pagination did not advance")
		}

		records := make([]connectors.Record, 0, len(response.Contacts))
		for _, c := range response.Contacts {
			if _, err := parseContactID(c.ID); err != nil {
				return nil, "", errors.New("ActiveCampaign returned a contact with an invalid id")
			}
			timestamp, err := contactUpdateTime(c)
			if err != nil {
				return nil, "", err
			}
			if !updatedAt.IsZero() && timestamp.Before(updatedAt) {
				continue
			}
			records = append(records, connectors.Record{
				ID: c.ID,
				Attributes: map[string]any{
					"email":     c.Email,
					"firstName": c.FirstName,
					"lastName":  c.LastName,
					"phone":     c.Phone,
				},
				UpdatedAt: timestamp,
			})
		}

		// ActiveCampaign's updated_after filter is documented as a date rather
		// than a timestamp. Scan by ID and filter locally so that Krenalis'
		// inclusive, microsecond-precision updatedAt contract remains exact.
		// https://developers.activecampaign.com/reference/list-all-contacts
		finalPage := len(response.Contacts) < contactsPageLimit
		if total, ok := responseTotal(response.Meta.Total); ok && total <= len(response.Contacts) {
			finalPage = true
		}
		if len(records) != 0 {
			if finalPage {
				return records, "", io.EOF
			}
			return records, nextCursor, nil
		}
		if finalPage {
			return nil, "", io.EOF
		}

		currentCursor = nextCursor
		currentID = nextID
	}
}

// Upsert creates or updates one ActiveCampaign Contact.
func (ac *ActiveCampaign) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {
	if target != connectors.TargetUser {
		return errors.New("ActiveCampaign supports only the user target")
	}

	record := records.First()
	payload := make(map[string]any, 4)
	for _, name := range []string{"email", "firstName", "lastName", "phone"} {
		if value, ok := record.Attributes[name]; ok {
			payload[name] = value
		}
	}

	method := http.MethodPut
	path := ""
	expectedStatus := http.StatusOK
	if record.IsCreate() {
		email, ok := payload["email"].(string)
		if !ok || email == "" {
			return connectors.RecordsError{0: errors.New("creating an ActiveCampaign contact requires email")}
		}
		// /contact/sync is a synchronous upsert keyed by email. It avoids a
		// duplicate-contact failure when a Krenalis create already exists.
		// https://developers.activecampaign.com/reference/sync-a-contacts-data
		method = http.MethodPost
		path = "/contact/sync"
		expectedStatus = http.StatusCreated
	} else {
		if _, err := parseContactID(record.ID); err != nil {
			return connectors.RecordsError{0: errors.New("updating an ActiveCampaign contact requires a valid contact id")}
		}
		path = "/contacts/" + url.PathEscape(record.ID)
	}

	settings, err := ac.loadSettings(ctx)
	if err != nil {
		return err
	}
	bb := ac.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()
	if err := bb.Encode(map[string]any{"contact": payload}); err != nil {
		return err
	}

	err = ac.callWithSettings(ctx, settings, method, path, bb, expectedStatus, nil)
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
			return connectors.RecordsError{0: apiErr}
		}
	}
	return err
}

// ServeUI serves the connector settings user interface.
func (ac *ActiveCampaign) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {
	switch event {
	case "load":
		var s innerSettings
		if err := ac.env.Settings.Load(ctx, &s); err != nil {
			return nil, err
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ac.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	return &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{
				Name:        "apiURL",
				Type:        "url",
				Label:       "API URL",
				Placeholder: "https://your-account.api-us1.com",
				HelpText:    "Use the API URL shown in ActiveCampaign under Settings > Developer.",
				MinLength:   1,
			},
			&connectors.Input{
				Name:      "apiToken",
				Type:      "password",
				Label:     "API token",
				HelpText:  "Use the API token shown in ActiveCampaign under Settings > Developer.",
				MinLength: 1,
			},
		},
		Settings: settings,
		Buttons:  []connectors.Button{connectors.SaveButton},
	}, nil
}

func (ac *ActiveCampaign) saveSettings(ctx context.Context, value json.Value) error {
	var settings innerSettings
	if err := value.Unmarshal(&settings); err != nil {
		return connectors.NewInvalidSettingsError("settings are not valid")
	}
	if err := normalizeSettings(&settings); err != nil {
		return err
	}

	err := ac.callWithSettings(ctx, settings, http.MethodGet, "/users/me", nil, http.StatusOK, nil)
	var apiErr *apiError
	if errors.As(err, &apiErr) && 300 <= apiErr.StatusCode && apiErr.StatusCode < 500 &&
		apiErr.StatusCode != http.StatusRequestTimeout && apiErr.StatusCode != http.StatusTooManyRequests {
		return connectors.NewInvalidSettingsError("the API URL or API token is not valid")
	}
	if err != nil {
		return err
	}
	return ac.env.Settings.Store(ctx, settings)
}

func (ac *ActiveCampaign) loadSettings(ctx context.Context) (innerSettings, error) {
	var settings innerSettings
	if err := ac.env.Settings.Load(ctx, &settings); err != nil {
		return innerSettings{}, err
	}
	if err := normalizeSettings(&settings); err != nil {
		return innerSettings{}, err
	}
	return settings, nil
}

func normalizeSettings(settings *innerSettings) error {
	apiURL, err := normalizeAPIURL(settings.APIURL)
	if err != nil {
		return connectors.NewInvalidSettingsError("API URL must be an HTTPS ActiveCampaign API URL without query parameters or fragments")
	}
	if settings.APIToken == "" {
		return connectors.NewInvalidSettingsError("API token is required")
	}
	for i := 0; i < len(settings.APIToken); i++ {
		if settings.APIToken[i] < 0x20 || settings.APIToken[i] == 0x7f {
			return connectors.NewInvalidSettingsError("API token contains a character that is not valid in an HTTP header")
		}
	}
	settings.APIURL = apiURL
	return nil
}

func normalizeAPIURL(raw string) (string, error) {
	// ActiveCampaign requires the complete account-specific HTTPS URL shown in
	// the account's Developer settings; the host is not always api-us1.com.
	// https://developers.activecampaign.com/reference/url
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" ||
		u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return "", errors.New("invalid ActiveCampaign API URL")
	}

	path := strings.TrimSuffix(u.Path, "/")
	switch path {
	case "":
		u.Path = "/api/3"
	case "/api/3":
		u.Path = path
	default:
		return "", errors.New("invalid ActiveCampaign API URL path")
	}
	u.RawPath = ""
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	return u.String(), nil
}

func (ac *ActiveCampaign) callWithSettings(ctx context.Context, settings innerSettings, method, path string, body *connectors.BodyBuffer, expectedStatus int, response any) error {
	req, err := body.NewRequest(ctx, method, settings.APIURL+path)
	if err != nil {
		return err
	}
	// https://developers.activecampaign.com/reference/authentication
	req.Header.Set("Api-Token", settings.APIToken)

	res, err := ac.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		return &apiError{StatusCode: res.StatusCode}
	}
	if response == nil {
		return nil
	}
	return decodeResponse(res.Body, response)
}

func decodeResponse(r io.Reader, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r, maxResponseBodySize+1))
	if err != nil {
		return err
	}
	if len(body) > maxResponseBodySize {
		return errors.New("ActiveCampaign response is too large")
	}
	return json.Unmarshal(body, dst)
}

func parseContactID(id string) (uint64, error) {
	n, err := strconv.ParseInt(id, 10, 32)
	if err != nil || n <= 0 {
		return 0, errors.New("invalid ActiveCampaign contact id")
	}
	return uint64(n), nil
}

func contactUpdateTime(c contact) (time.Time, error) {
	value := c.UpdatedAt
	if value == "" || strings.HasPrefix(value, "0000-00-00") {
		value = c.CreatedAt
	}
	if value == "" || strings.HasPrefix(value, "0000-00-00") {
		return time.Time{}, errors.New("ActiveCampaign returned a contact without an update timestamp")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, errors.New("ActiveCampaign returned a contact with an invalid update timestamp")
	}
	return timestamp, nil
}

func responseTotal(value json.Value) (int, bool) {
	if value.IsString() {
		total, err := strconv.Atoi(string(value.Bytes()))
		return total, err == nil && total >= 0
	}
	if value.Kind() == json.Number {
		total, err := value.Int()
		return total, err == nil && total >= 0
	}
	return 0, false
}

type apiError struct {
	StatusCode int
}

func (err *apiError) Error() string {
	if err == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ActiveCampaign returned HTTP %d", err.StatusCode)
}
