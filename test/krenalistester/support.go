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

func (c *Krenalis) AlterProfileSchema(schema types.Type, primarySources map[string]int, rePaths map[string]any) {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	ts := time.Now().UTC()
	c.MustCall("PUT", "/v1/profiles/schema", nil, req, nil)
	// Waits for the alter schema that was started following the call to this
	// method to finish.
	for {
		time.Sleep(50 * time.Millisecond)
		startTime, endTime, alterError := c.LatestAlterProfileSchema()
		if alterError != nil {
			c.t.Fatalf("profile schema altering failed: %s", *alterError)
		}
		// On Windows, it may happen that 'startTime' is exactly equal to 'ts'
		// because the precision of timestamps is lower: for this reason, it is
		// necessary to check that 'startTime ≥ ts', not just that it is after.
		if (startTime.Equal(ts) || startTime.After(ts)) && endTime != nil {
			break
		}
	}
}

// AlterProfileSchemaErr is like AlterProfileSchema but returns an error instead of
// panicking.
func (c *Krenalis) AlterProfileSchemaErr(schema types.Type, primarySources map[string]int, rePaths map[string]any) error {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	return c.Call("PUT", "/v1/profiles/schema", nil, req, nil)
}

func (c *Krenalis) AbsolutePath(storage int, path string) string {
	var response struct {
		Path string `json:"path"`
	}
	endpointPath := fmt.Sprintf("/v1/connections/%d/files/absolute", storage)
	if path != "" {
		endpointPath += "?path=" + url.QueryEscape(path)
	}
	c.MustCall("GET", endpointPath, nil, nil, &response)
	return response.Path
}

func (c *Krenalis) PipelineSchemas(conn int, target core.Target, eventType string) map[string]any {
	path := fmt.Sprintf("/v1/connections/%d/pipelines/schemas/%s", conn, target)
	if eventType != "" {
		path += "?type=" + url.QueryEscape(eventType)
	}
	var schemas map[string]any
	c.MustCall("GET", path, nil, nil, &schemas)
	return schemas
}

func (c *Krenalis) ConnectionIdentities(conn, first, limit int) ([]Identity, int) {
	var response struct {
		Identities []Identity `json:"identities"`
		Total      int        `json:"total"`
	}
	path := fmt.Sprintf("/v1/connections/%d/identities?first=%d&limit=%d", conn, first, limit)
	c.MustCall("GET", path, nil, nil, &response)
	return response.Identities, response.Total
}

func (c *Krenalis) ConnectionUI(connection int) map[string]any {
	path := fmt.Sprintf("/v1/connections/%d/ui", connection)
	var ui map[string]any
	c.MustCall("GET", path, nil, nil, &ui)
	return ui
}

func (c *Krenalis) CreatePipeline(conn int, target string, pipeline PipelineToSet) int {
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
		ID int `json:"id"`
	}
	c.MustCall("POST", "/v1/pipelines", nil, body, &response)
	return response.ID
}

// CreatePipelineErr is like CreatePipeline but returns an error instead of
// panicking.
func (c *Krenalis) CreatePipelineErr(conn int, target string, pipeline PipelineToSet) (int, error) {
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
		ID int `json:"id"`
	}
	err = c.Call("POST", "/v1/pipelines", nil, body, &response)
	if err != nil {
		return 0, err
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

func (c *Krenalis) CreateConnection(connection ConnectionToCreate) int {
	var response struct {
		ID int `json:"id"`
	}
	c.MustCall("POST", "/v1/connections", nil, connection, &response)
	return response.ID
}

func (c *Krenalis) CreateDestinationFilesystem() int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "File System",
		Role:      Destination,
		Connector: "filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"simulateHighIOLatency": false,
		}),
	})
}

func (c *Krenalis) CreateDestinationPostgreSQL() int {
	return c.CreateConnection(ConnectionToCreate{
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

func (c *Krenalis) CreateDummy(name string, role Role) int {
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
	return c.CreateConnection(conn)
}

func (c *Krenalis) CreateDummyWithSettings(name string, role Role, settings DummySettings) int {
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
	return c.CreateConnection(conn)
}

func (c *Krenalis) CreateEventPipeline(conn int, eventType string, pipeline PipelineToSet) int {
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
		ID int `json:"id"`
	}
	c.MustCall("POST", "/v1/pipelines", nil, body, &response)
	return response.ID
}

func (c *Krenalis) CreateWebhookSource(name string, linkedConnections []int) int {
	return c.CreateConnection(ConnectionToCreate{
		Name:              name,
		Role:              Source,
		Connector:         "webhook",
		LinkedConnections: linkedConnections,
	})
}

func (c *Krenalis) CreateWorkspaceRestrictedAPIKey(name string) string {
	var response struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	body := struct {
		Name      string        `json:"name"`
		Workspace int           `json:"workspace"`
		Type      AccessKeyType `json:"type"`
	}{
		Name:      name,
		Workspace: c.WorkspaceID(),
		Type:      AccessKeyTypeAPI,
	}
	c.MustCall("POST", "/v1/keys", nil, body, &response)
	return response.Token
}

func organizationsHeaders() http.Header {
	return http.Header{
		"Krenalis-Workspace": nil, // so that MustCall does not add automatically the header.
		"Authorization":      []string{"Bearer " + testsSettings.OrganizationsAPIKey},
	}
}

func (c *Krenalis) CreateOrganization(name string) uuid.UUID {
	var response struct {
		ID uuid.UUID `json:"id"`
	}
	c.MustCall("POST", "/v1/organizations", organizationsHeaders(), map[string]any{"name": name}, &response)
	return response.ID
}

func (c *Krenalis) Organization(id uuid.UUID) Organization {
	var org Organization
	c.MustCall("GET", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, &org)
	return org
}

// OrganizationErr is like Organization but returns an error instead of failing the test.
func (c *Krenalis) OrganizationErr(id uuid.UUID) error {
	return c.Call("GET", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, nil)
}

func (c *Krenalis) Organizations(first, limit int) []Organization {
	var response struct {
		Organizations []Organization `json:"organizations"`
	}
	path := fmt.Sprintf("/v1/organizations?first=%d&limit=%d", first, limit)
	c.MustCall("GET", path, organizationsHeaders(), nil, &response)
	return response.Organizations
}

func (c *Krenalis) UpdateOrganization(id uuid.UUID, name string) {
	c.MustCall("PUT", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), map[string]any{"name": name}, nil)
}

func (c *Krenalis) DeleteOrganization(id uuid.UUID) {
	c.MustCall("DELETE", fmt.Sprintf("/v1/organizations/%s", id), organizationsHeaders(), nil, nil)
}

func (c *Krenalis) CreateJavaScriptSource(name string, linkedConnections []int) int {
	return c.CreateConnection(ConnectionToCreate{
		Name:              name,
		Role:              Source,
		Connector:         "javascript",
		Strategy:          &defaultStrategy,
		LinkedConnections: linkedConnections,
	})
}

func (c *Krenalis) CreateSourceFileSystem() int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "File System",
		Role:      Source,
		Connector: "filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"simulateHighIOLatency": false,
		}),
	})
}

func (c *Krenalis) CreateSourcePostgreSQL() int {
	return c.CreateConnection(ConnectionToCreate{
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

func (c *Krenalis) DeleteConnection(conn int) {
	path := fmt.Sprintf("/v1/connections/%d", conn)
	c.MustCall("DELETE", path, nil, nil, nil)
}

func (c *Krenalis) RunPipeline(pipeline int) int {
	var response struct {
		ID int
	}
	path := fmt.Sprintf("/v1/pipelines/%d/runs", pipeline)
	c.MustCall("POST", path, nil, map[string]any{}, &response)
	return response.ID
}

func (c *Krenalis) ExternalEventURL() string {
	var metadata map[string]any
	c.MustCall("GET", "/v1/public/metadata", nil, nil, &metadata)
	return metadata["externalEventURL"].(string)
}

func (c *Krenalis) Events(properties []string) []map[string]any {
	queryString := url.Values{
		"properties": properties,
		"first":      []string{"0"},
		"limit":      []string{"10"},
	}
	var response struct {
		Events []map[string]any `json:"events"`
	}
	c.MustCall("GET", "/v1/events"+"?"+queryString.Encode(), nil, nil, &response)
	return response.Events
}

func (c *Krenalis) File(storage int, path, format, sheet string, compression Compression, settings json.Value, limit int) ([]map[string]any, types.Type) {
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
	endpointPath := fmt.Sprintf("/v1/connections/%d/files", storage)
	c.MustCall("GET", endpointPath+"?"+queryString.Encode(), nil, nil, &response)
	return response.Records, response.Schema
}

func (c *Krenalis) JavaScriptSDKURL() string {
	var metadata map[string]any
	c.MustCall("GET", "/v1/public/metadata", nil, nil, &metadata)
	return metadata["javascriptSDKURL"].(string)
}

func (c *Krenalis) LatestAlterProfileSchema() (startTime, endTime *time.Time, alterError *string) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
		Error     *string    `json:"error"`
	}
	c.MustCall("GET", "/v1/profiles/schema/latest-alter", nil, nil, &response)
	return response.StartTime, response.EndTime, response.Error
}

func (c *Krenalis) LatestIdentityResolution() (startTime, endTime *time.Time) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
	}
	c.MustCall("GET", "/v1/identity-resolution/latest", nil, nil, &response)
	return response.StartTime, response.EndTime
}

func (c *Krenalis) PipelineRun(id int) PipelineRun {
	var run PipelineRun
	path := fmt.Sprintf("/v1/pipelines/runs/%d", id)
	c.MustCall("GET", path, nil, nil, &run)
	return run
}

func (c *Krenalis) PipelineRuns() []PipelineRun {
	var response struct {
		Runs []PipelineRun
	}
	c.MustCall("GET", "/v1/pipelines/runs", nil, nil, &response)
	return response.Runs
}

func (c *Krenalis) PreviewAlterProfileSchema(schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	c.MustCall("PUT", "/v1/profiles/schema/preview", nil, req, &response)
	return response.Queries
}

// PreviewAlterProfileSchemaErr is like PreviewAlterProfileSchema but returns an
// error instead of panicking.
func (c *Krenalis) PreviewAlterProfileSchemaErr(schema types.Type, rePaths map[string]any) ([]string, error) {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	err := c.Call("PUT", "/v1/profiles/schema/preview", nil, req, &response)
	if err != nil {
		return nil, err
	}
	return response.Queries, nil
}

func (c *Krenalis) RepairWarehouse() {
	c.MustCall("POST", "/v1/warehouse/repair", nil, nil, nil)
}

// RunIdentityResolution starts the identity resolution and waits for it to
// complete.
func (c *Krenalis) RunIdentityResolution() {
	ts := time.Now().UTC()
	c.MustCall("POST", "/v1/identity-resolution/start", nil, nil, nil)
	// Waits for the Identity Resolution that was started following the call to
	// this method to finish.
	for {
		time.Sleep(50 * time.Millisecond)
		startTime, endTime := c.LatestIdentityResolution()
		// On Windows, it may happen that 'startTime' is exactly equal to 'ts'
		// because the precision of timestamps is lower: for this reason, it is
		// necessary to check that 'startTime ≥ ts', not just that it is after.
		if (startTime.Equal(ts) || startTime.After(ts)) && endTime != nil {
			break
		}
	}
}

func (c *Krenalis) SendEvent(writeKey string, message analytics.Message) {
	endpoint := "http://" + c.Addr() + "/v1/events"
	cb := sendEventCallback{ch: make(chan error, 1)}
	client, err := analytics.NewWithConfig(
		writeKey,
		analytics.Config{
			Endpoint: endpoint,
			Callback: cb,
		},
	)
	if err != nil {
		c.t.Fatalf("cannot create client: %s", err)
	}
	err = client.Enqueue(message)
	if err != nil {
		c.t.Fatalf("cannot enqueue event: %s", err)
	}
	err = client.Close()
	if err != nil {
		c.t.Fatalf("cannot send event: %s", err)
	}
	err = <-cb.ch
	if err != nil {
		c.t.Fatalf("cannot close client when sending events: %s", err)
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

func (c *Krenalis) Sheets(storage int, path string, format string, compression Compression, settings json.Value) []string {
	queryString := url.Values{
		"path":           []string{path},
		"format":         []string{format},
		"compression":    []string{string(compression)},
		"formatSettings": []string{string(settings)},
	}
	var response struct {
		Sheets []string `json:"sheets"`
	}
	endpointPath := fmt.Sprintf("/v1/connections/%d/files/sheets", storage)
	c.MustCall("GET", endpointPath+"?"+queryString.Encode(), nil, nil, &response)
	return response.Sheets
}

func (c *Krenalis) TableSchema(conn int, table string) (types.Type, []string) {
	var response struct {
		Schema types.Type `json:"schema"`
		Issues []string   `json:"issues"`
	}
	path := fmt.Sprintf("/v1/connections/%d/tables", conn)
	if table != "" {
		path += "?name=" + url.QueryEscape(table)
	}
	c.MustCall("GET", path, nil, nil, &response)
	return response.Schema, response.Issues
}

func (c *Krenalis) TestWarehouseUpdate(settings json.Value) {
	body := map[string]any{
		"settings": settings,
	}
	c.MustCall("PUT", "/v1/warehouse/test", nil, body, nil)
}

func (c *Krenalis) TestWorkspaceCreation(name string, profileSchema types.Type,
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
	return c.Call("POST", "/v1/workspaces/test", headers, body, nil)
}

func (c *Krenalis) UpdatePipeline(pipelineID int, pipeline PipelineToSet) {
	path := fmt.Sprintf("/v1/pipelines/%d", pipelineID)
	c.MustCall("PUT", path, nil, pipeline, nil)
}

func (c *Krenalis) UpdateIdentityResolution(runOnBatchImport bool, identifiers []string) {
	body := map[string]any{
		"runOnBatchImport": runOnBatchImport,
		"identifiers":      identifiers,
	}
	c.MustCall("PUT", "/v1/identity-resolution/settings", nil, body, nil)
}

func (c *Krenalis) UpdateIdentityResolutionErr(identifiers []string) error {
	body := map[string]any{
		"identifiers": identifiers,
	}
	return c.Call("PUT", "/v1/identity-resolution/settings", nil, body, nil)
}

func (c *Krenalis) UpdateWarehouse(mode string, settings json.Value) {
	body := map[string]any{
		"mode":     mode,
		"settings": settings,
	}
	c.MustCall("PUT", "/v1/warehouse", nil, body, nil)
}

func (c *Krenalis) ProfileEvents(kpid uuid.UUID, properties []string) []map[string]any {
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
	c.MustCall("GET", "/v1/events"+"?"+queryString.Encode(), nil, nil, &response)
	return response.Events
}

func (c *Krenalis) Identities(kpid uuid.UUID, first, limit int) ([]Identity, int) {
	var response struct {
		Identities []Identity `json:"identities"`
		Total      int        `json:"total"`
	}
	path := fmt.Sprintf("/v1/profiles/%s/identities?first=%d&limit=%d", kpid, first, limit)
	c.MustCall("GET", path, nil, nil, &response)
	return response.Identities, response.Total
}

func (c *Krenalis) ProfilePropertiesSuitableAsIdentifiers() types.Type {
	var schema types.Type
	c.MustCall("GET", "/v1/profiles/schema/suitable-as-identifiers", nil, nil, &schema)
	return schema
}

func (c *Krenalis) Profiles(properties []string, order string, orderDesc bool, first, limit int) (users []Profile, schema types.Type, total int) {
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
	c.MustCall("GET", "/v1/profiles?"+queryString.Encode(), nil, nil, &response)
	return response.Profiles, response.Schema, response.Total
}

func (c *Krenalis) WaitEventsStoredIntoWarehouse(ctx context.Context, expected int) {
	bo := backoff.New(200)
	bo.SetAttempts(20)
	bo.SetCap(2 * time.Second)
	bo.SetNextWaitTime(200 * time.Millisecond)
	for bo.Next(ctx) {
		count := c.CountEventsInWarehouse(ctx)
		if count == expected {
			break
		}
		c.t.Logf("[attempt %d] %d event(s) stored in warehouse until now", bo.Attempt(), count)
		if bo.WaitTime() == 0 {
			c.t.Fatalf("too many failed attempts")
		}
	}
}

// WaitRunsCompletion waits for the runs with the specified IDs, belonging to
// the connection, to be completed. In the event that a run ends with an error,
// or there is at least one "Failed", this method will result in an error.
//
// If you intend to proceed even in the case of one or more "Failed" (but not an
// error for the entire run), see the method WaitForRunsCompletionAllowFailed.
func (c *Krenalis) WaitRunsCompletion(conn int, runs ...int) {
	c.waitForRunsCompletion(false, runs...)
}

// WaitForRunsCompletionAllowFailed waits for the runs with the specified IDs,
// belonging to the connection, to be completed. In the event that a run ends
// with an error, this method will result in an error. If there is one or more
// Failed, they are ignored.
//
// If you want the method to result in an error even in the case of one or more
// "Failed", see the method WaitForRunsCompletion.
func (c *Krenalis) WaitForRunsCompletionAllowFailed(conn int, runs ...int) {
	c.waitForRunsCompletion(true, runs...)
}

func (c *Krenalis) EventWriteKeys(conn int) []string {
	var res struct {
		Keys []string `json:"keys"`
	}
	path := fmt.Sprintf("/v1/connections/%d/event-write-keys", conn)
	c.MustCall("GET", path, nil, nil, &res)
	return res.Keys
}

func (c *Krenalis) Workspace() Workspace {
	var ws Workspace
	c.MustCall("GET", "/v1/workspaces/current", nil, nil, &ws)
	return ws
}

func (c *Krenalis) waitForRunsCompletion(allowFailed bool, ids ...int) {
	time.Sleep(500 * time.Millisecond)
	for {
		if len(ids) == 1 {
			run := c.PipelineRun(ids[0])
			if run.EndTime != nil {
				// If the pipeline run ended with an error, make the test fail.
				if run.Error != "" {
					c.t.Fatalf("an error occurred when running pipeline %d on connection %d: %s", run.Pipeline, run.ID, run.Error)
				}
				if !allowFailed && run.Failed != [6]int{} {
					c.t.Fatalf("an error occurred when running pipeline %d on connection %d: %d failed", run.Pipeline, run.ID, run.Failed)
				}
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}
		completed := true
		for _, run := range c.PipelineRuns() {
			if !slices.Contains(ids, run.ID) {
				continue
			}
			if run.EndTime == nil {
				completed = false
				continue
			}
			// If the pipeline run ended with an error, make the test fail.
			if run.Error != "" {
				c.t.Fatalf("an error occurred when running pipeline %d on connection %d: %s", run.Pipeline, run.ID, run.Error)
			}
			if !allowFailed && run.Failed != [6]int{} {
				c.t.Fatalf("an error occurred when running pipeline %d on connection %d: %d failed", run.Pipeline, run.ID, run.Failed)
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
