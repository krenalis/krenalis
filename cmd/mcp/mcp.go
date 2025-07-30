//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"context"
	"net/http"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"

	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer returns a new MCP server, which servers HTTP requests from MCP
// clients.
//
// An MCP server can be initialized with a nil Meergo core. In that case, only
// operations that do not involve the core (eg. listing the tools, the prompts,
// etc...) are supported; otherwise, a panic may occur. This is useful in tests.
func NewMCPServer(core *core.Core) *MCPServer {

	// Instantiate an MCP server.
	m := server.NewMCPServer("Meergo MCP server", "0.0.0", server.WithRecovery())

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

// ServeHTTP serves HTTP requests from MCP clients.
func (mcp *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Retrieve the Meergo MCP token from the request, otherwise return a Bad
	// Request error to the MCP client.
	mcpToken, err := mcpTokenFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Inject the MCP token and the Meergo core into the request's context, so
	// it is made available to the MCP tools.
	ctx := r.Context()
	ctx = context.WithValue(ctx, mcpContextKey("mcp-token"), mcpToken)
	ctx = context.WithValue(ctx, mcpContextKey("meergo-core"), mcp.core)
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

// mcpTokenFromRequest reads the Meergo MCP token from the request's
// 'Authorization' header.
func mcpTokenFromRequest(r *http.Request) (string, error) {
	auth, ok := r.Header["Authorization"]
	if !ok {
		return "", errors.New("no Authorization header found in request")
	}
	token, found := strings.CutPrefix(auth[0], "Bearer ")
	if !found || token == "" {
		return "", errors.BadRequest("Authorization header is invalid. It should be in the format 'Authorization: Bearer <YOUR_MCP_KEY>'.")
	}
	return token, nil
}

// meergoCoreFromCtx extracts the Meergo core from the given context, returning
// error if it is not found.
func meergoCoreFromCtx(ctx context.Context) (*core.Core, error) {
	core, ok := ctx.Value(mcpContextKey("meergo-core")).(*core.Core)
	if !ok {
		return nil, errors.New("meergo core not found in context")
	}
	return core, nil
}
