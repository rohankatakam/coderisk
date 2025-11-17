package mcp

import (
	"context"

	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// Tool represents an MCP tool
type Tool interface {
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	GetSchema() map[string]interface{}
}

// Resource represents an MCP resource
type Resource interface {
	Read(ctx context.Context) (interface{}, error)
}

// Handler handles MCP protocol requests
type Handler struct {
	tools     map[string]Tool
	resources map[string]Resource
}

// NewHandler creates a new MCP handler
func NewHandler() *Handler {
	return &Handler{
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
	}
}

// RegisterTool registers a tool with the handler
func (h *Handler) RegisterTool(name string, tool Tool) {
	h.tools[name] = tool
}

// RegisterResource registers a resource with the handler
func (h *Handler) RegisterResource(name string, resource Resource) {
	h.resources[name] = resource
}

// Handle processes a JSON-RPC request
func (h *Handler) Handle(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req)
	case "tools/list":
		return h.handleToolsList(req)
	case "tools/call":
		return h.handleToolCall(req)
	case "resources/list":
		return h.handleResourcesList(req)
	case "resources/read":
		return h.handleResourceRead(req)
	default:
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles the initialize request
func (h *Handler) handleInitialize(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	return &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "1.0",
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "crisk-check-server",
				"version": "0.1.0",
			},
		},
	}
}

// handleToolsList handles the tools/list request
func (h *Handler) handleToolsList(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	toolsList := []map[string]interface{}{}

	for name, tool := range h.tools {
		toolsList = append(toolsList, map[string]interface{}{
			"name":   name,
			"schema": tool.GetSchema(),
		})
	}

	return &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": toolsList,
		},
	}
}

// handleToolCall handles the tools/call request
func (h *Handler) handleToolCall(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	// Extract tool name from params
	toolName, ok := req.Params["name"].(string)
	if !ok {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: 'name' is required",
			},
		}
	}

	// Get the tool
	tool, exists := h.tools[toolName]
	if !exists {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32602,
				Message: "Tool not found: " + toolName,
			},
		}
	}

	// Extract arguments
	args, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	// Execute the tool
	ctx := context.Background()
	result, err := tool.Execute(ctx, args)
	if err != nil {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32603,
				Message: "Tool execution error: " + err.Error(),
			},
		}
	}

	return &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleResourcesList handles the resources/list request
func (h *Handler) handleResourcesList(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	resourcesList := []map[string]interface{}{}

	for name := range h.resources {
		resourcesList = append(resourcesList, map[string]interface{}{
			"name": name,
		})
	}

	return &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"resources": resourcesList,
		},
	}
}

// handleResourceRead handles the resources/read request
func (h *Handler) handleResourceRead(req *tools.JSONRPCRequest) *tools.JSONRPCResponse {
	// Extract resource name from params
	resourceName, ok := req.Params["name"].(string)
	if !ok {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: 'name' is required",
			},
		}
	}

	// Get the resource
	resource, exists := h.resources[resourceName]
	if !exists {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32602,
				Message: "Resource not found: " + resourceName,
			},
		}
	}

	// Read the resource
	ctx := context.Background()
	result, err := resource.Read(ctx)
	if err != nil {
		return &tools.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &tools.JSONRPCError{
				Code:    -32603,
				Message: "Resource read error: " + err.Error(),
			},
		}
	}

	return &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}
