package atomizer

import (
	"fmt"
	"sync"
)

// StateTracker maintains in-memory state of code blocks during chronological processing
// Reference: AGENT_P2B_PROCESSOR.md - State tracking for CREATE/MODIFY/DELETE resolution
type StateTracker struct {
	// Map: "file_path:block_name" -> CodeBlock PostgreSQL ID
	blocks map[string]int64
	mu     sync.RWMutex
}

// NewStateTracker creates a new state tracker for code blocks
func NewStateTracker() *StateTracker {
	return &StateTracker{
		blocks: make(map[string]int64),
	}
}

// GetBlockID retrieves the PostgreSQL ID for a code block
// Returns (id, true) if block exists, (0, false) if not found
func (s *StateTracker) GetBlockID(filePath, blockName string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeBlockKey(filePath, blockName)
	id, exists := s.blocks[key]
	return id, exists
}

// SetBlockID stores the PostgreSQL ID for a code block
// Used after CREATE_BLOCK operations to track new blocks
func (s *StateTracker) SetBlockID(filePath, blockName string, id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeBlockKey(filePath, blockName)
	s.blocks[key] = id
}

// DeleteBlock removes a code block from the state tracker
// Used for DELETE_BLOCK operations (though block remains in DB with status='deleted')
func (s *StateTracker) DeleteBlock(filePath, blockName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeBlockKey(filePath, blockName)
	delete(s.blocks, key)
}

// BlockExists checks if a block is currently tracked
// Equivalent to GetBlockID but returns only boolean
func (s *StateTracker) BlockExists(filePath, blockName string) bool {
	_, exists := s.GetBlockID(filePath, blockName)
	return exists
}

// GetBlockCount returns the number of blocks currently tracked
// Used for metrics and logging
func (s *StateTracker) GetBlockCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}

// makeBlockKey creates a unique key for a code block
// Format: "file_path:block_name"
// Example: "src/utils/parser.go:parseExpression"
func makeBlockKey(filePath, blockName string) string {
	return fmt.Sprintf("%s:%s", filePath, blockName)
}
