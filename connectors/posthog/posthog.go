// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package posthog provides a connector for PostHog.
// (https://posthog.com/docs/api)
//
// PostHog is a trademark of PostHog Inc.
// This connector is not affiliated with or endorsed by PostHog Inc.
package posthog

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/tools/validation"
)

// PostHog supports NoEncoding and Gzip for request bodies.
const contentEncoding = connectors.Gzip

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "posthog",
		Label:      "PostHog",
		Categories: connectors.CategorySaaS,
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetEvent,
			HasSettings: true,
			SendingMode: connectors.Server,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Send events to PostHog",
				Overview: overview,
			},
		},
	}, New)
}

type PostHog struct {
	env      *connectors.APIEnv
	settings *innerSettings
}

type cloudSettings struct {
	ProjectRegion string `json:"projectRegion"`
}

type selfHostedSettings struct {
	URL string `json:"url"`
}

type innerSettings struct {
	APIKey     string              `json:"apiKey"`
	Cloud      *cloudSettings      `json:"cloud"`
	SelfHosted *selfHostedSettings `json:"selfHosted"`
}

// New returns a new connector instance for PostHog.
func New(env *connectors.APIEnv) (*PostHog, error) {
	c := PostHog{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for PostHog")
		}
	}
	return &c, nil
}

// EventTypes returns the event types.
func (posthog *PostHog) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	return []*connectors.EventType{
		{
			ID:          "identify",
			Name:        "Identify",
			Description: "Send Identify events to PostHog",
			Filter:      "type is 'identify'",
		},
		{
			ID:          "group",
			Name:        "Group identify",
			Description: "Send Group identify events to PostHog",
			Filter:      "type is 'group'",
		},
		{
			ID:          "alias",
			Name:        "Alias",
			Description: "Send Alias events to PostHog",
			Filter:      "type is 'alias'",
		},
		{
			ID:          "page",
			Name:        "Pageview",
			Description: "Send Pageview events to PostHog",
			Filter:      "type is 'page'",
		},
		{
			ID:          "screen",
			Name:        "Screen",
			Description: "Send Screen events to PostHog",
			Filter:      "type is 'screen'",
		},
		{
			ID:          "track",
			Name:        "Track",
			Description: "Send Track events to PostHog",
			Filter:      "type is 'track'",
		},
	}, nil
}

// EventTypeSchema returns the schema of the specified event type.
func (posthog *PostHog) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	switch eventType {
	case "identify":
		return types.Object([]types.Property{
			{Name: "properties", Prefilled: `traits`, Type: types.Map(types.JSON()), CreateRequired: true, Description: "Properties"},
		}), nil
	case "alias":
		// Segment's "alias" events do not have a "traits" or "properties" object.
		// However, PostHog requires a "properties" object, so we default it to an
		// empty map that can be enriched by mappings.
		return types.Object([]types.Property{
			{Name: "properties", Prefilled: `map()`, Type: types.Map(types.JSON()), CreateRequired: true, Description: "Properties"},
		}), nil
	case "group":
		return types.Object([]types.Property{
			{Name: "group_type", Prefilled: `"company"`, Type: types.String().WithMaxLength(400), CreateRequired: true, Description: "Group type"},
			{Name: "properties", Prefilled: `traits`, Type: types.Map(types.JSON()), CreateRequired: true, Description: "Properties"},
		}), nil
	case "track":
		return types.Object([]types.Property{
			{Name: "event", Prefilled: `event`, Type: types.String(), CreateRequired: true, Description: "Event"},
			{Name: "properties", Prefilled: `properties`, Type: types.Map(types.JSON()), CreateRequired: true, Description: "Properties"},
		}), nil
	case "page", "screen":
		return types.Object([]types.Property{
			{Name: "properties", Prefilled: `properties`, Type: types.Map(types.JSON()), CreateRequired: true, Description: "Properties"},
		}), nil
	}
	return types.Type{}, connectors.ErrEventTypeNotExist
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the API, without actually sending it.
func (posthog *PostHog) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return posthog.sendEvents(ctx, events, true)
}

// SendEvents sends events to the API.
func (posthog *PostHog) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := posthog.sendEvents(ctx, events, false)
	return err
}

// ServeUI serves the connector's user interface.
func (posthog *PostHog) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if posthog.settings != nil {
			s = *posthog.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, posthog.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "apiKey", Label: "Project API key", Placeholder: "phc_FmajIzDq2gv9yZpB2Ooyt9YCurRvGl2QPTcCnxbMl4M", MinLength: 47, MaxLength: 47},
			&connectors.AlternativeFieldSets{
				Label: "Deployment",
				Sets: []connectors.FieldSet{
					{
						Name:  "cloud",
						Label: "Cloud",
						Fields: []connectors.Component{
							&connectors.Select{Name: "projectRegion", Label: "Project region", Options: []connectors.Option{{Text: "US Cloud", Value: "US"}, {Text: "EU Cloud", Value: "EU"}}},
						},
					},
					{
						Name:  "selfHosted",
						Label: "Self-hosted",
						Fields: []connectors.Component{
							&connectors.Input{Name: "url", Label: "URL", Placeholder: "https://www.example.com/", Type: "text", MinLength: 1, MaxLength: 253},
						},
					},
				},
			},
		},
		Settings: settings,
	}

	return ui, nil
}

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (posthog *PostHog) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate APIKey.
	if len(s.APIKey) != 47 || !strings.HasPrefix(s.APIKey, "phc_") {
		return connectors.NewInvalidSettingsError("API key does not appear to be a valid PostHog project API key")
	}
	if s.Cloud != nil && s.SelfHosted != nil {
		return connectors.NewInvalidSettingsError("cloud and selfHosted fields are mutually exclusive")
	}
	if s.Cloud == nil && s.SelfHosted == nil {
		return connectors.NewInvalidSettingsError("either the cloud or the self-hosted field must be set")
	}
	if s.Cloud != nil {
		if r := s.Cloud.ProjectRegion; r != "US" && r != "EU" {
			return connectors.NewInvalidSettingsErrorf("project region is required and must be either US or EU")
		}
	} else {
		if s.SelfHosted.URL == "" {
			return connectors.NewInvalidSettingsError("self-hosted URL cannot be empty")
		}
		var err error
		s.SelfHosted.URL, err = validation.ParseURL(s.SelfHosted.URL, validation.NoPath|validation.NoQuery)
		if err != nil {
			return connectors.NewInvalidSettingsErrorf("self-hosted URL is not valid: %s", err)
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = posthog.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	posthog.settings = &s
	return nil
}

const maxEventRequestSize = 20 * 1024 * 1024 // from https://posthog.com/docs/api/capture?#batch-events

// sendEvents sends the given events to the API and returns the sent HTTP
// request. If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the API, a nil *http.Request
// and the error are returned.
func (posthog *PostHog) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {

	// bb contains newline-delimited JSON objects representing the events.
	bb := posthog.env.HTTPClient.GetBodyBuffer(contentEncoding)
	defer bb.Close()

	apiKey := posthog.settings.APIKey
	if preview {
		apiKey = "[REDACTED]"
	}

	_ = bb.WriteByte('{')
	_ = bb.EncodeKeyValue("api_key", apiKey)
	_, _ = bb.WriteString(`,"batch":[`)

	wroteEvent := false
	for ev := range events.All() {

		size := bb.Len()

		distinctID, ok := ev.Received.UserID()
		if !ok {
			distinctID = ev.Received.AnonymousID()
		}
		values := ev.Type.Values
		properties := values["properties"].(map[string]any)

		var event string
		switch ev.Type.ID {
		case "identify":
			event = "$identify"
			properties = map[string]any{
				"$set":              properties,
				"$anon_distinct_id": ev.Received.AnonymousID(),
			}
		case "group":
			event = "$groupidentify"
			groupID, ok := ev.Received.GroupID()
			if !ok {
				events.Discard(errors.New("event does not have a group ID"))
				continue
			}
			if utf8.RuneCountInString(groupID) > 400 {
				events.Discard(errors.New("event's group ID is longer than 400 characters"))
				continue
			}
			properties = map[string]any{
				"$group_type": values["group_type"],
				"$group_key":  groupID,
				"$group_set":  properties,
			}
		case "alias":
			event = "$create_alias"
			previousID, ok := ev.Received.PreviousID()
			if !ok {
				events.Discard(errors.New("event does not have a previous ID"))
				continue
			}
			properties["alias"] = distinctID
			distinctID = previousID
		case "page":
			event = "$pageview"
			if ctx, ok := ev.Received.Context(); ok {
				if page, ok := ctx.Page(); ok {
					if currentURL, ok := page.URL(); ok {
						properties["$current_url"] = currentURL
					}
				}
			}
		case "screen":
			event = "$screen"
			if screenName, ok := ev.Received.Name(); ok {
				properties["$screen_name"] = screenName
			}
		case "track":
			event = values["event"].(string)
			if strings.HasPrefix(event, "$") {
				events.Discard(errors.New("event name cannot start with «$», as this prefix is reserved by PostHog"))
				continue
			}
		}

		// IP address.
		if ctx, ok := ev.Received.Context(); ok {
			if ip, ok := ctx.IP(); ok {
				properties["$ip"] = ip
			}
		}
		if _, ok := properties["$ip"]; !ok {
			properties["$geoip_disable"] = true
		}

		if wroteEvent {
			_ = bb.WriteByte(',')
		}
		_ = bb.WriteByte('{')
		_ = bb.EncodeKeyValue("event", event)
		_ = bb.EncodeKeyValue("distinct_id", distinctID)
		_ = bb.EncodeKeyValue("properties", properties)
		_ = bb.EncodeKeyValue("timestamp", ev.Received.Timestamp().Format(time.RFC3339))
		_ = bb.EncodeKeyValue("uuid", ev.Received.MessageID())
		_ = bb.WriteByte('}')

		// Stop if body exceeds the API's size limit.
		if bb.Len()+len(`]}`) >= maxEventRequestSize {
			// From the PostHog documentation: «the entire request body must be less than 20MB by default»
			// https://posthog.com/docs/api/capture?#batch-events
			bb.Truncate(size)
			if wroteEvent {
				events.Postpone()
				break
			}
			events.Discard(errors.New("event exceeds PostHog's maximum request size"))
			continue
		}

		if err := bb.Flush(); err != nil {
			return nil, err
		}

		wroteEvent = true
	}
	if !wroteEvent {
		return nil, nil
	}

	_, _ = bb.WriteString("]}")

	var u string
	if cloud := posthog.settings.Cloud; cloud != nil {
		switch cloud.ProjectRegion {
		case "US":
			u = "https://us.i.posthog.com/batch/"
		case "EU":
			u = "https://eu.i.posthog.com/batch/"
		default:
			return nil, fmt.Errorf("expected projectRegion to be US or EU, got %q", cloud.ProjectRegion)
		}
	} else {
		u = posthog.settings.SelfHosted.URL + "batch/"
	}

	req, err := bb.NewRequest(ctx, "POST", u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header["Idempotency-Key"] = nil // mark the request as idempotent

	if preview {
		return req, nil
	}

	// Send the request.
	res, err := posthog.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	_ = res.Body.Close()
	if res.StatusCode < 300 {
		return req, nil
	}

	// Handle the error.
	return nil, fmt.Errorf("PostHog server responded with %d error code", res.StatusCode)
}
