package atomizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSignature(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(user: string, pass: string)", "(user:string,pass:string)"},
		{"( ctx context.Context , id int64 ): error", "(ctxcontext.Context,idint):error"},
		{"()", "()"},
		{"(x:str,y:int)", "(x:string,y:int)"},
		{"(data: Map<string, number>)", "(data:Map<string,number>)"},
		{"", ""},
		{"(a: int32, b: int64)", "(a:int,b:int)"},
		{"(flag: bool)", "(flag:boolean)"},
	}

	for _, tt := range tests {
		result := NormalizeSignature(tt.input)
		assert.Equal(t, tt.expected, result, "Failed for input: %s", tt.input)
	}
}

func TestExtractParameterCount(t *testing.T) {
	tests := []struct {
		sig      string
		expected int
	}{
		{"(user:string)", 1},
		{"(user:string,pass:string)", 2},
		{"()", 0},
		{"", 0},
		{"(a,b,c)", 3},
		{"(x: int, y: int, z: string)", 3},
		{"no parens", 0},
	}

	for _, tt := range tests {
		result := ExtractParameterCount(tt.sig)
		assert.Equal(t, tt.expected, result, "Failed for sig: %s", tt.sig)
	}
}

func TestSignaturesMatch(t *testing.T) {
	// Exact matches
	assert.True(t, SignaturesMatch("(user:string)", "(user:string)", false))
	assert.True(t, SignaturesMatch("(user: string)", "(user:string)", false), "Should normalize before comparing")

	// Fuzzy matches
	assert.True(t, SignaturesMatch("(user:string)", "(usr:string)", true))
	assert.True(t, SignaturesMatch("(user:string,pass:string)", "(user:string,password:string)", true))

	// Non-matches
	assert.False(t, SignaturesMatch("(user:string)", "(user:int)", false))
	assert.False(t, SignaturesMatch("(user:string)", "(user:string,pass:string)", true)) // Different param count

	// Type alias normalization
	assert.True(t, SignaturesMatch("(x:str)", "(x:string)", false))
	assert.True(t, SignaturesMatch("(id:int64)", "(id:int)", false))
	assert.True(t, SignaturesMatch("(flag:bool)", "(flag:boolean)", false))
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"user", "usr", 1}, // Delete 'e' from position 2
	}

	for _, tt := range tests {
		result := levenshteinDistance(tt.s1, tt.s2)
		assert.Equal(t, tt.expected, result, "Failed for s1=%s, s2=%s", tt.s1, tt.s2)
	}
}

func TestMinFunction(t *testing.T) {
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(3, 1))
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, -1, min(-1, 0))
}
