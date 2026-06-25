// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/uuid"
	"github.com/krenalis/analytics-go"
)

var defaultStrategy Strategy = "Conversion"

// This file contains support methods which reduce verbosity of tests.

func (k *Krenalis) AlterProfileSchemaAndWait(schema types.Type, primarySources map[string]string, rePaths map[string]any) {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	ts := time.Now().UTC()
	k.Call("PUT", "/v1/profiles/schema", nil, req, nil)
	// Waits for the alter schema that was started following the call to this
	// method to finish.
	for {
		time.Sleep(50 * time.Millisecond)
		startTime, endTime, alterError := k.LatestAlterProfileSchema()
		if alterError != nil {
			k.t.Fatalf("profile schema altering failed: %s", *alterError)
		}
		// On Windows, it may happen that 'startTime' is exactly equal to 'ts'
		// because the precision of timestamps is lower: for this reason, it is
		// necessary to check that 'startTime ≥ ts', not just that it is after.
		if (startTime.Equal(ts) || startTime.After(ts)) && endTime != nil {
			break
		}
	}
}

func (k *Krenalis) TryAlterProfileSchema(schema types.Type, primarySources map[string]string, rePaths map[string]any) error {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	return k.TryCall("PUT", "/v1/profiles/schema", nil, req, nil)
}

func (k *Krenalis) AbsolutePath(storage string, path string) string {
	var response struct {
		Path string `json:"path"`
	}
	endpointPath := fmt.Sprintf("/v1/connections/%s/files/absolute", storage)
	if path != "" {
		endpointPath += "?path=" + url.QueryEscape(path)
	}
	k.Call("GET", endpointPath, nil, nil, &response)
	return response.Path
}

func (k *Krenalis) PipelineSchemas(conn string, target core.Target, eventType string) map[string]any {
	path := fmt.Sprintf("/v1/connections/%s/pipelines/schemas/%s", conn, target)
	if eventType != "" {
		path += "?type=" + url.QueryEscape(eventType)
	}
	var schemas map[string]any
	k.Call("GET", path, nil, nil, &schemas)
	return schemas
}

func (k *Krenalis) ConnectionIdentities(conn string, first, limit int) ([]Identity, int) {
	var response struct {
		Identities []Identity `json:"identities"`
		Total      int        `json:"total"`
	}
	path := fmt.Sprintf("/v1/connections/%s/identities?first=%d&limit=%d", conn, first, limit)
	k.Call("GET", path, nil, nil, &response)
	return response.Identities, response.Total
}

func (k *Krenalis) ConnectionUI(connection string) map[string]any {
	path := fmt.Sprintf("/v1/connections/%s/ui", connection)
	var ui map[string]any
	k.Call("GET", path, nil, nil, &ui)
	return ui
}

func (k *Krenalis) CreatePipeline(conn string, target string, pipeline PipelineToSet) string {
	switch target {
	case "Event", "User", "Group":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	pipelineJSON, err := stdjson.Marshal(pipeline)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = stdjson.Unmarshal(pipelineJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = target
	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/pipelines", nil, body, &response)
	return response.ID
}

func (k *Krenalis) TryCreatePipeline(conn string, target string, pipeline PipelineToSet) (string, error) {
	switch target {
	case "Event", "User", "Group":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	pipelineJSON, err := stdjson.Marshal(pipeline)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = stdjson.Unmarshal(pipelineJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = target
	var response struct {
		ID string `json:"id"`
	}
	err = k.TryCall("POST", "/v1/pipelines", nil, body, &response)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

// DefaultFilterUserFromEvents is the filter that the admin adds by default to
// the pipelines that import users from events.
var DefaultFilterUserFromEvents = &Filter{
	Logical: "or",
	Conditions: []FilterCondition{
		{
			Property: "type",
			Operator: "is",
			Values:   []string{"identify"},
		},
		{
			Property: "traits",
			Operator: "is not empty",
			Values:   nil,
		},
	},
}

func (k *Krenalis) CreateConnection(connection ConnectionToCreate) string {
	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/connections", nil, connection, &response)
	return response.ID
}

func (k *Krenalis) TryCreateConnection(connection ConnectionToCreate) (string, error) {
	var response struct {
		ID string `json:"id"`
	}
	err := k.TryCall("POST", "/v1/connections", nil, connection, &response)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func (k *Krenalis) CreateDestinationFilesystem() string {
	return k.CreateConnection(ConnectionToCreate{
		Name:      "File System",
		Role:      Destination,
		Connector: "filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"simulateHighIOLatency": false,
		}),
	})
}

func (k *Krenalis) CreateDestinationPostgreSQL() string {
	return k.CreateConnection(ConnectionToCreate{
		Name:      "PostgreSQL (destination)",
		Role:      Destination,
		Connector: "postgresql",
		Settings: JSONEncodeSettings(map[string]any{
			"host":     testsSettings.Database.Host,
			"port":     testsSettings.Database.Port,
			"username": testsSettings.Database.Username,
			"password": testsSettings.Database.Password,
			"database": testsSettings.Database.Database,
			"schema":   testsSettings.Database.Schema,
		}),
	})
}

func (k *Krenalis) CreateDummy(name string, role Role) string {
	conn := ConnectionToCreate{
		Name:      name,
		Role:      role,
		Connector: "dummy",
		Settings:  json.Value("{}"),
	}
	if role == Destination {
		mode := Server
		conn.SendingMode = &mode
	}
	return k.CreateConnection(conn)
}

func (k *Krenalis) CreateDummyWithSettings(name string, role Role, settings DummySettings) string {
	conn := ConnectionToCreate{
		Name:      name,
		Role:      role,
		Connector: "dummy",
		Settings:  JSONEncodeSettings(settings),
	}
	if role == Destination {
		mode := Server
		conn.SendingMode = &mode
	}
	return k.CreateConnection(conn)
}

func (k *Krenalis) CreateEventPipeline(conn string, eventType string, pipeline PipelineToSet) string {
	pipelineJSON, err := stdjson.Marshal(pipeline)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = stdjson.Unmarshal(pipelineJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = "Event"
	body["eventType"] = eventType
	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/pipelines", nil, body, &response)
	return response.ID
}

func (k *Krenalis) CreateWebhookSource(name string, linkedConnections []string) string {
	return k.CreateConnection(ConnectionToCreate{
		Name:              name,
		Role:              Source,
		Connector:         "webhook",
		LinkedConnections: linkedConnections,
	})
}

func (k *Krenalis) CreateWorkspaceRestrictedAPIKey(name string) string {
	var response struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	body := struct {
		Name      string        `json:"name"`
		Workspace string        `json:"workspace"`
		Type      AccessKeyType `json:"type"`
	}{
		Name:      name,
		Workspace: k.WorkspaceID(),
		Type:      AccessKeyTypeAPI,
	}
	k.Call("POST", "/v1/keys", nil, body, &response)
	return response.Token
}

func organizationsHeaders() http.Header {
	return http.Header{
		"Krenalis-Workspace": nil, // so that Call does not add automatically the header.
		"Authorization":      []string{"Bearer " + testsSettings.OrganizationsAPIKey},
	}
}

func (k *Krenalis) CreateOrganization(name string, enabled bool) string {
	var response struct {
		ID string `json:"id"`
	}
	body := map[string]any{"name": name, "enabled": enabled}
	k.Call("POST", "/v1/organizations", organizationsHeaders(), body, &response)
	return response.ID
}

func (k *Krenalis) Organization(id string) Organization {
	var org Organization
	k.Call("GET", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, &org)
	return org
}

// CanGetOrganization reports whether the organization with the given ID can be
// retrieved, returning an error if it cannot.
func (k *Krenalis) CanGetOrganization(id string) error {
	return k.TryCall("GET", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, nil)
}

func (k *Krenalis) Organizations(first, limit int) []Organization {
	var response struct {
		Organizations []Organization `json:"organizations"`
	}
	path := fmt.Sprintf("/v1/organizations?first=%d&limit=%d", first, limit)
	k.Call("GET", path, organizationsHeaders(), nil, &response)
	return response.Organizations
}

func (k *Krenalis) UpdateOrganization(id string, name string) {
	k.Call("PUT", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), map[string]any{"name": name}, nil)
}

func (k *Krenalis) DeleteOrganization(id string) {
	k.Call("DELETE", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, nil)
}

func (k *Krenalis) CreateJavaScriptSource(name string, linkedConnections []string) string {
	return k.CreateConnection(ConnectionToCreate{
		Name:              name,
		Role:              Source,
		Connector:         "javascript",
		Strategy:          &defaultStrategy,
		LinkedConnections: linkedConnections,
	})
}

func (k *Krenalis) CreateSourceFileSystem() string {
	return k.CreateConnection(ConnectionToCreate{
		Name:      "File System",
		Role:      Source,
		Connector: "filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"simulateHighIOLatency": false,
		}),
	})
}

func (k *Krenalis) CreateSourcePostgreSQL() string {
	return k.CreateConnection(ConnectionToCreate{
		Name:      "PostgreSQL (destination)",
		Role:      Source,
		Connector: "postgresql",
		Settings: JSONEncodeSettings(map[string]any{
			"host":     testsSettings.Database.Host,
			"port":     testsSettings.Database.Port,
			"username": testsSettings.Database.Username,
			"password": testsSettings.Database.Password,
			"database": testsSettings.Database.Database,
			"schema":   testsSettings.Database.Schema,
		}),
	})
}

func (k *Krenalis) DeleteConnection(conn string) {
	path := fmt.Sprintf("/v1/connections/%s", conn)
	k.Call("DELETE", path, nil, nil, nil)
}

func (k *Krenalis) TryDeleteConnection(conn string) error {
	path := fmt.Sprintf("/v1/connections/%s", conn)
	return k.TryCall("DELETE", path, nil, nil, nil)
}

func (k *Krenalis) StartPipelineRun(pipeline string) string {
	var response struct {
		ID string
	}
	path := fmt.Sprintf("/v1/pipelines/%s/runs", pipeline)
	k.Call("POST", path, nil, map[string]any{}, &response)
	return response.ID
}

func (k *Krenalis) TryStartPipelineRun(pipeline string) (string, error) {
	var response struct {
		ID string
	}
	path := fmt.Sprintf("/v1/pipelines/%s/runs", pipeline)
	err := k.TryCall("POST", path, nil, map[string]any{}, &response)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func (k *Krenalis) ExternalEventURL() string {
	var metadata map[string]any
	k.Call("GET", "/v1/public/metadata", nil, nil, &metadata)
	return metadata["externalEventURL"].(string)
}

func (k *Krenalis) Events(properties []string) []map[string]any {
	queryString := url.Values{
		"properties": properties,
		"first":      []string{"0"},
		"limit":      []string{"10"},
	}
	var response struct {
		Events []map[string]any `json:"events"`
	}
	k.Call("GET", "/v1/events"+"?"+queryString.Encode(), nil, nil, &response)
	return response.Events
}

// CanGetEvents reports whether the events (passing the given properties) can be
// retrieved, returning an error if they cannot.
func (k *Krenalis) CanGetEvents(properties []string) error {
	queryString := url.Values{
		"properties": properties,
		"first":      []string{"0"},
		"limit":      []string{"10"},
	}
	return k.TryCall("GET", "/v1/events"+"?"+queryString.Encode(), nil, nil, nil)
}

func (k *Krenalis) File(storage string, path, format, sheet string, compression Compression, settings json.Value, limit int) ([]map[string]any, types.Type) {
	queryString := url.Values{
		"path":           []string{path},
		"format":         []string{format},
		"sheet":          []string{sheet},
		"compression":    []string{string(compression)},
		"formatSettings": []string{string(settings)},
		"limit":          []string{strconv.Itoa(limit)},
	}
	var response struct {
		Records []map[string]any `json:"records"`
		Schema  types.Type       `json:"schema"`
	}
	endpointPath := fmt.Sprintf("/v1/connections/%s/files", storage)
	k.Call("GET", endpointPath+"?"+queryString.Encode(), nil, nil, &response)
	return response.Records, response.Schema
}

func (k *Krenalis) JavaScriptSDKURL() string {
	var metadata map[string]any
	k.Call("GET", "/v1/public/metadata", nil, nil, &metadata)
	return metadata["javascriptSDKURL"].(string)
}

func (k *Krenalis) LatestAlterProfileSchema() (startTime, endTime *time.Time, alterError *string) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
		Error     *string    `json:"error"`
	}
	k.Call("GET", "/v1/profiles/schema/latest-alter", nil, nil, &response)
	return response.StartTime, response.EndTime, response.Error
}

func (k *Krenalis) LatestIdentityResolution() (startTime, endTime *time.Time) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
	}
	k.Call("GET", "/v1/identity-resolution/latest", nil, nil, &response)
	return response.StartTime, response.EndTime
}

func (k *Krenalis) PipelineRun(id string) PipelineRun {
	var run PipelineRun
	path := fmt.Sprintf("/v1/pipelines/runs/%s", id)
	k.Call("GET", path, nil, nil, &run)
	return run
}

func (k *Krenalis) PipelineRuns() []PipelineRun {
	var response struct {
		Runs []PipelineRun
	}
	k.Call("GET", "/v1/pipelines/runs", nil, nil, &response)
	return response.Runs
}

func (k *Krenalis) PreviewAlterProfileSchema(schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	k.Call("PUT", "/v1/profiles/schema/preview", nil, req, &response)
	return response.Queries
}

func (k *Krenalis) TryPreviewAlterProfileSchema(schema types.Type, rePaths map[string]any) ([]string, error) {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	err := k.TryCall("PUT", "/v1/profiles/schema/preview", nil, req, &response)
	if err != nil {
		return nil, err
	}
	return response.Queries, nil
}

func (k *Krenalis) RepairWarehouse() {
	k.Call("POST", "/v1/warehouse/repair", nil, nil, nil)
}

func (k *Krenalis) TryRepairWarehouse() error {
	return k.TryCall("POST", "/v1/warehouse/repair", nil, nil, nil)
}

// RunIdentityResolutionAndWait runs the identity resolution and waits for it to
// complete before returning.
func (k *Krenalis) RunIdentityResolutionAndWait() {
	ts := time.Now().UTC()
	k.Call("POST", "/v1/identity-resolution/start", nil, nil, nil)
	// Waits for the Identity Resolution that was started following the call to
	// this method to finish.
	for {
		time.Sleep(50 * time.Millisecond)
		startTime, endTime := k.LatestIdentityResolution()
		// On Windows, it may happen that 'startTime' is exactly equal to 'ts'
		// because the precision of timestamps is lower: for this reason, it is
		// necessary to check that 'startTime ≥ ts', not just that it is after.
		if (startTime.Equal(ts) || startTime.After(ts)) && endTime != nil {
			break
		}
	}
}

func (k *Krenalis) TryStartIdentityResolution() error {
	return k.TryCall("POST", "/v1/identity-resolution/start", nil, nil, nil)
}

func (k *Krenalis) SendEvent(writeKey string, message analytics.Message) {
	endpoint := "http://" + k.Addr() + "/v1/events"
	cb := sendEventCallback{ch: make(chan error, 1)}
	client, err := analytics.NewWithConfig(
		writeKey,
		analytics.Config{
			Endpoint: endpoint,
			Callback: cb,
		},
	)
	if err != nil {
		k.t.Fatalf("cannot create client: %s", err)
	}
	err = client.Enqueue(message)
	if err != nil {
		k.t.Fatalf("cannot enqueue event: %s", err)
	}
	err = client.Close()
	if err != nil {
		k.t.Fatalf("cannot send event: %s", err)
	}
	err = <-cb.ch
	if err != nil {
		k.t.Fatalf("cannot close client when sending events: %s", err)
	}
}

// sendEventCallback is used to handle the result of sending an event.
// It communicates success or failure through an error channel.
type sendEventCallback struct {
	ch chan error
}

func (s sendEventCallback) Success(msg analytics.Message) {
	s.ch <- nil
}

func (s sendEventCallback) Failure(msg analytics.Message, err error) {
	s.ch <- err
}

func (k *Krenalis) Sheets(storage string, path string, format string, compression Compression, settings json.Value) []string {
	queryString := url.Values{
		"path":           []string{path},
		"format":         []string{format},
		"compression":    []string{string(compression)},
		"formatSettings": []string{string(settings)},
	}
	var response struct {
		Sheets []string `json:"sheets"`
	}
	endpointPath := fmt.Sprintf("/v1/connections/%s/files/sheets", storage)
	k.Call("GET", endpointPath+"?"+queryString.Encode(), nil, nil, &response)
	return response.Sheets
}

func (k *Krenalis) TableSchema(conn string, table string) (types.Type, []string) {
	var response struct {
		Schema types.Type `json:"schema"`
		Issues []string   `json:"issues"`
	}
	path := fmt.Sprintf("/v1/connections/%s/tables", conn)
	if table != "" {
		path += "?name=" + url.QueryEscape(table)
	}
	k.Call("GET", path, nil, nil, &response)
	return response.Schema, response.Issues
}

func (k *Krenalis) TestWarehouseUpdate(settings json.Value) {
	body := map[string]any{
		"settings": settings,
	}
	k.Call("PUT", "/v1/warehouse/test", nil, body, nil)
}

func (k *Krenalis) TestWorkspaceCreation(name string, profileSchema types.Type,
	uiPreferences UIPreferences, whPlatform string, whSettings json.Value, mode WarehouseMode) error {
	headers := http.Header{
		"Krenalis-Workspace": nil,
	}
	body := map[string]any{
		"name":          name,
		"profileSchema": profileSchema,
		"warehouse": map[string]any{
			"platform": whPlatform,
			"mode":     mode,
			"settings": whSettings,
		},
		"uiPreferences": uiPreferences,
	}
	return k.TryCall("POST", "/v1/workspaces/test", headers, body, nil)
}

// DeletePipeline deletes a pipeline.
func (k *Krenalis) DeletePipeline(pipelineID string) {
	path := fmt.Sprintf("/v1/pipelines/%s", pipelineID)
	k.Call("DELETE", path, nil, nil, nil)
}

func (k *Krenalis) TryDeletePipeline(pipelineID string) error {
	path := fmt.Sprintf("/v1/pipelines/%s", pipelineID)
	return k.TryCall("DELETE", path, nil, nil, nil)
}

func (k *Krenalis) UpdatePipeline(pipelineID string, pipeline PipelineToSet) {
	path := fmt.Sprintf("/v1/pipelines/%s", pipelineID)
	k.Call("PUT", path, nil, pipeline, nil)
}

func (k *Krenalis) UpdateIdentityResolutionSettings(runOnBatchImport bool, identifiers []string) {
	body := map[string]any{
		"runOnBatchImport": runOnBatchImport,
		"identifiers":      identifiers,
	}
	k.Call("PUT", "/v1/identity-resolution/settings", nil, body, nil)
}

func (k *Krenalis) TryUpdateIdentityResolutionSettings(identifiers []string) error {
	body := map[string]any{
		"identifiers": identifiers,
	}
	return k.TryCall("PUT", "/v1/identity-resolution/settings", nil, body, nil)
}

func (k *Krenalis) UpdateWarehouse(mode string, settings json.Value) {
	body := map[string]any{
		"mode":     mode,
		"settings": settings,
	}
	k.Call("PUT", "/v1/warehouse", nil, body, nil)
}

func (k *Krenalis) ProfileEvents(kpid uuid.UUID, properties []string) []map[string]any {
	queryString := url.Values{
		"properties": properties,
		"first":      []string{"0"},
		"limit":      []string{"10"},
	}
	filter := Filter{
		Logical: OpAnd,
		Conditions: []FilterCondition{
			{Property: "kpid",
				Operator: OpIs,
				Values:   []string{kpid.String()}},
		},
	}
	jsonFilter, err := stdjson.Marshal(filter)
	if err != nil {
		panic(err)
	}
	queryString.Add("filter", string(jsonFilter))
	var response struct {
		Events []map[string]any `json:"events"`
	}
	k.Call("GET", "/v1/events"+"?"+queryString.Encode(), nil, nil, &response)
	return response.Events
}

func (k *Krenalis) Identities(kpid uuid.UUID, first, limit int) ([]Identity, int) {
	var response struct {
		Identities []Identity `json:"identities"`
		Total      int        `json:"total"`
	}
	path := fmt.Sprintf("/v1/profiles/%s/identities?first=%d&limit=%d", kpid, first, limit)
	k.Call("GET", path, nil, nil, &response)
	return response.Identities, response.Total
}

func (k *Krenalis) ProfilePropertiesSuitableAsIdentifiers() types.Type {
	var schema types.Type
	k.Call("GET", "/v1/profiles/schema/suitable-as-identifiers", nil, nil, &schema)
	return schema
}

func (k *Krenalis) Profiles(properties []string, order string, orderDesc bool, first, limit int) (users []Profile, schema types.Type, total int) {
	queryString := url.Values{
		"properties": properties,
		"order":      []string{order},
		"orderDesc":  []string{fmt.Sprintf("%t", orderDesc)},
		"first":      []string{strconv.Itoa(first)},
		"limit":      []string{strconv.Itoa(limit)},
	}
	var response struct {
		Profiles []Profile  `json:"profiles"`
		Schema   types.Type `json:"schema"`
		Total    int        `json:"total"`
	}
	k.Call("GET", "/v1/profiles?"+queryString.Encode(), nil, nil, &response)
	return response.Profiles, response.Schema, response.Total
}

// SetOrganizationStatus enables or disables an organization through the
// organizations API.
func (k *Krenalis) SetOrganizationStatus(id string, enabled bool) {
	body := map[string]any{"enabled": enabled}
	k.Call("PUT", fmt.Sprintf("/v1/organizations/%s/status", id), organizationsHeaders(), body, nil)
}

// TrySetOrganizationStatus is like SetOrganizationStatus but sends the request
// with the given headers and returns an error instead of failing the test.
func (k *Krenalis) TrySetOrganizationStatus(id string, enabled bool, headers http.Header) error {
	body := map[string]any{"enabled": enabled}
	return k.TryCall("PUT", fmt.Sprintf("/v1/organizations/%s/status", id), headers, body, nil)
}

func (k *Krenalis) WaitConnectionIdentitiesStoredIntoWarehouse(ctx context.Context, connection string, expected int) {
	bo := backoff.New(200)
	const attempts = 20
	bo.SetAttempts(attempts)
	bo.SetCap(2 * time.Second)
	bo.SetNextWaitTime(200 * time.Millisecond)
	for bo.Next(ctx) {
		_, count := k.ConnectionIdentities(connection, 0, 1)
		if count == expected {
			break
		}
		k.t.Logf("[attempt %d] %d connection identities stored in warehouse until now", bo.Attempt(), count)
		if bo.WaitTime() == 0 {
			k.t.Fatalf("too many failed attempts (%d identities were expected, but after %d attempts %d identities are returned by Krenalis)", expected, attempts, count)
		}
	}
}

func (k *Krenalis) WaitEventsStoredIntoWarehouse(ctx context.Context, expected int) {
	bo := backoff.New(200)
	const attempts = 20
	bo.SetAttempts(attempts)
	bo.SetCap(2 * time.Second)
	bo.SetNextWaitTime(200 * time.Millisecond)
	for bo.Next(ctx) {
		count := k.CountEventsInWarehouse(ctx)
		if count == expected {
			break
		}
		k.t.Logf("[attempt %d] %d events stored in warehouse until now", bo.Attempt(), count)
		if bo.WaitTime() == 0 {
			k.t.Fatalf("too many failed attempts (%d events were expected, but after %d attempts %d events are returned by Krenalis)", expected, attempts, count)
		}
	}
}

// WaitRunsCompletion waits for the runs with the specified IDs, belonging to
// the connection, to be completed. In the event that a run ends with an error,
// or there is at least one "Failed", this method will result in an error.
//
// If you intend to proceed even in the case of one or more "Failed" (but not an
// error for the entire run), see the method WaitForRunsCompletionAllowFailed.
func (k *Krenalis) WaitRunsCompletion(conn string, runs ...string) {
	k.waitForRunsCompletion(false, runs...)
}

// WaitForRunsCompletionAllowFailed waits for the runs with the specified IDs,
// belonging to the connection, to be completed. In the event that a run ends
// with an error, this method will result in an error. If there is one or more
// Failed, they are ignored.
//
// If you want the method to result in an error even in the case of one or more
// "Failed", see the method WaitForRunsCompletion.
func (k *Krenalis) WaitForRunsCompletionAllowFailed(conn string, runs ...string) {
	k.waitForRunsCompletion(true, runs...)
}

func (k *Krenalis) EventWriteKeys(conn string) []string {
	var res struct {
		Keys []string `json:"keys"`
	}
	path := fmt.Sprintf("/v1/connections/%s/event-write-keys", conn)
	k.Call("GET", path, nil, nil, &res)
	return res.Keys
}

func (k *Krenalis) Workspace() Workspace {
	var ws Workspace
	k.Call("GET", "/v1/workspaces/current", nil, nil, &ws)
	return ws
}

func (k *Krenalis) waitForRunsCompletion(allowFailed bool, ids ...string) {
	time.Sleep(500 * time.Millisecond)
	for {
		if len(ids) == 1 {
			run := k.PipelineRun(ids[0])
			if run.EndTime != nil {
				// If the pipeline run ended with an error, make the test fail.
				if run.Error != "" {
					k.t.Fatalf("error running pipeline %s for run %s: %s", run.Pipeline, run.ID, run.Error)
				}
				if !allowFailed && run.Failed != [6]int{} {
					k.t.Fatalf("error running pipeline %s for run %s: %d failed", run.Pipeline, run.ID, run.Failed)
				}
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}
		completed := true
		for _, run := range k.PipelineRuns() {
			if !slices.Contains(ids, run.ID) {
				continue
			}
			if run.EndTime == nil {
				completed = false
				continue
			}
			// If the pipeline run ended with an error, make the test fail.
			if run.Error != "" {
				k.t.Fatalf("error running pipeline %s for run %s: %s", run.Pipeline, run.ID, run.Error)
			}
			if !allowFailed && run.Failed != [6]int{} {
				k.t.Fatalf("error running pipeline %s for run %s: %d failed", run.Pipeline, run.ID, run.Failed)
			}
		}
		if completed {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func JSONEncodeSettings(values any) json.Value {
	data, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection settings to JSON: %s", err))
	}
	return data
}

func SettingsProperties(properties map[string]bool) json.Value {
	var settings = struct {
		Properties []KV `json:"properties"`
	}{
		Properties: []KV{},
	}
	for name, required := range properties {
		kv := KV{Key: name}
		if required {
			kv.Value = "t"
		} else {
			kv.Value = "f"
		}
		settings.Properties = append(settings.Properties, kv)
	}
	return JSONEncodeSettings(settings)
}
