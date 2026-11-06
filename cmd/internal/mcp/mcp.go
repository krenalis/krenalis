// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/validation"

	"github.com/mark3labs/mcp-go/server"
)

const serverInstructions = "Krenalis MCP tools must be used only through this MCP server." +
	" For profiles and segments, use query-profiles." +
	" Read profile-filter-doc before constructing a filter." +
	" Discover business events with list-event-definitions." +
	" Inspect get-event-definition before choosing a track event." +
	" Catalog schemas do not prove historical conformity or enable unsupported filtering capabilities." +
	" query-data-warehouse is retained for discovering event names, inspecting JSON properties and other analysis." +
	" Never use SQL to emulate an unsupported Unified capability." +
	" Report unsupported capabilities explicitly and do not reinterpret them as empty results or UNKNOWN." +
	" query-profiles requires configured MCP settings but uses the main Krenalis warehouse connection." +
	" The separate MCP credentials do not restrict its data access." +
	" On Snowflake it uses the main virtual warehouse, not the MCP virtual warehouse." +
	" Count remains unavailable on Snowflake until its backend gate is enabled." +
	" The workspace data warehouse is managed directly by Krenalis." +
	" Writes and schema changes for Krenalis-managed tables and views must be performed only by Krenalis itself." +
	" If a user wants to modify data, schema, or configuration, explain clearly and professionally that user-driven changes must be done only through the Krenalis Admin console or the Krenalis APIs, not through direct SQL on the warehouse." +
	" The warehouse access exposed by this MCP server is strictly read-only and intended only for analytical retrieval of data." +
	" You must not execute, suggest, or help craft INSERT, UPDATE, DELETE, MERGE, TRUNCATE, DDL, maintenance, locking, privilege, or any other query that could modify the warehouse or create operational problems." +
	" You must also avoid read-only queries that are unnecessarily expensive, abusive, or likely to place avoidable operational strain on the warehouse." +
	" When this Krenalis MCP server is configured, you must not search for, read, infer, or use warehouse credentials, connection strings, secrets, or configuration from the repository, environment, or any other source." +
	" You must not connect directly to the Krenalis data warehouse in any way, even if the user explicitly asks you to do so."

// NewMCPServer returns a new MCP server, which servers HTTP requests from MCP
// clients.
//
// An MCP server can be initialized with a nil Krenalis core. In that case, only
// operations that do not involve the core (eg. listing the tools, the prompts,
// etc...) are supported; otherwise, a panic may occur. This is useful in tests.
//
// The caller must call Close on the returned instance to release any associated
// resources when it is no longer needed.
func NewMCPServer(core *core.Core) *MCPServer {

	// Instantiate an MCP server.
	m := server.NewMCPServer(
		"Krenalis MCP server",
		"0.0.0",
		server.WithRecovery(),
		server.WithInstructions(serverInstructions),
	)

	// Register the prompts.
	m.AddPrompts(prompts...)

	// Register the tools.
	m.AddTools(tools...)

	return &MCPServer{
		server: server.NewStreamableHTTPServer(m),
		core:   core,
	}
}

// MCPServer serves HTTP requests from MCP clients.
type MCPServer struct {
	server *server.StreamableHTTPServer
	core   *core.Core
}

// Close mcp.
func (mcp *MCPServer) Close(ctx context.Context) error {
	return mcp.server.Shutdown(ctx)
}

const maxRequestBytes = 1024 * 1024 // 1 MiB; includes the JSON-RPC envelope

// ServeHTTP serves HTTP requests from MCP clients.
func (mcp *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Retrieve the Krenalis MCP token from the request, otherwise return a Bad
	// Request error to the MCP client.
	mcpToken, err := mcpTokenFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Bound the whole POST before the SDK reads and decodes it.
	if r.Method == http.MethodPost {
		if r.ContentLength > maxRequestBytes {
			http.Error(w, "MCP request body exceeds 1 MiB", http.StatusRequestEntityTooLarge)
			return
		}
		if r.Body != nil {
			limited := http.MaxBytesReader(w, r.Body, maxRequestBytes)
			defer limited.Close()
			body, err := io.ReadAll(limited)
			if err != nil {
				if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
					http.Error(w, "MCP request body exceeds 1 MiB", http.StatusRequestEntityTooLarge)
					return
				}
				http.Error(w, "cannot read MCP request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	// Inject the MCP token and the Krenalis core into the request's context, so
	// it is made available to the MCP tools.
	ctx := r.Context()
	ctx = context.WithValue(ctx, mcpContextKey("mcp-token"), mcpToken)
	ctx = context.WithValue(ctx, mcpContextKey("krenalis-core"), mcp.core)
	r = r.Clone(ctx)

	mcp.server.ServeHTTP(w, r)

}

type mcpContextKey string

// mcpTokenFromCtx extracts the MCP token from the given context, returning
// error if it is not found.
func mcpTokenFromCtx(ctx context.Context) (string, error) {
	mcpToken, ok := ctx.Value(mcpContextKey("mcp-token")).(string)
	if !ok {
		return "", errors.New("MCP token not found in context")
	}
	return mcpToken, nil
}

// mcpTokenFromRequest reads the Krenalis MCP token from the request's
// 'Authorization' header.
func mcpTokenFromRequest(r *http.Request) (string, error) {
	auth, ok := r.Header["Authorization"]
	if !ok {
		return "", errors.New("no Authorization header found in request")
	}
	if len(auth) > 1 {
		return "", errors.BadRequest("request contains multiple Authorization headers")
	}
	token, found := validation.ParseBearer(auth[0])
	if !found {
		return "", errors.BadRequest("Authorization header is invalid. It should be in the format 'Authorization: Bearer <YOUR_MCP_KEY>'.")
	}
	if !strings.HasPrefix(token, "mcp_") {
		return "", errors.BadRequest("MCP key is not valid; MCP keys start with the mcp_ prefix.")
	}
	return token, nil
}

// krenalisCoreFromCtx extracts the Krenalis core from the given context, returning
// error if it is not found.
func krenalisCoreFromCtx(ctx context.Context) (*core.Core, error) {
	core, ok := ctx.Value(mcpContextKey("krenalis-core")).(*core.Core)
	if !ok {
		return nil, errors.New("krenalis core not found in context")
	}
	return core, nil
}
