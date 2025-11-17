package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// StdioTransport handles JSON-RPC communication over stdio
type StdioTransport struct {
	scanner *bufio.Scanner
	handler *Handler
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(handler *Handler) *StdioTransport {
	return &StdioTransport{
		scanner: bufio.NewScanner(os.Stdin),
		handler: handler,
	}
}

// Start begins listening for JSON-RPC requests on stdin
func (t *StdioTransport) Start() error {
	for t.scanner.Scan() {
		line := t.scanner.Text()

		// Parse JSON-RPC request
		var req tools.JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			t.sendError(nil, -32700, "Parse error")
			continue
		}

		// Handle request
		response := t.handler.Handle(&req)

		// Send response
		respJSON, _ := json.Marshal(response)
		fmt.Println(string(respJSON))
	}
	return t.scanner.Err()
}

// sendError sends a JSON-RPC error response
func (t *StdioTransport) sendError(id interface{}, code int, message string) {
	response := &tools.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &tools.JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	respJSON, _ := json.Marshal(response)
	fmt.Println(string(respJSON))
}
