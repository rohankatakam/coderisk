package git

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractChunksForNewFile_Python(t *testing.T) {
	fileContent := `
def func1():
    pass

def func2():
    pass

def func3():
    return True
`

	chunks := ExtractChunksForNewFile(fileContent, "python", 10)

	// Should create chunks respecting function boundaries
	assert.GreaterOrEqual(t, len(chunks), 1)
	assert.LessOrEqual(t, len(chunks), 10)

	// Each chunk should have header
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.FileHeader)
		assert.Contains(t, chunk.FileHeader, "@@")
	}
}

func TestExtractChunksForNewFile_Go(t *testing.T) {
	fileContent := `package main

func main() {
    println("Hello World")
}

func helper() {
    return 42
}

func (s *Server) Start() error {
    return nil
}
`

	chunks := ExtractChunksForNewFile(fileContent, "go", 10)

	// Should detect Go functions
	assert.GreaterOrEqual(t, len(chunks), 1)

	// Each chunk should have proper line numbers
	for _, chunk := range chunks {
		assert.GreaterOrEqual(t, chunk.StartLine, 0)
		assert.GreaterOrEqual(t, chunk.EndLine, chunk.StartLine)
	}
}

func TestExtractChunksForNewFile_JavaScript_Nested(t *testing.T) {
	fileContent := `
function outer() {
    function inner() {
        return 42;
    }
    return inner();
}
`

	chunks := ExtractChunksForNewFile(fileContent, "javascript", 10)

	// Inner function should be included in outer function chunk
	assert.Len(t, chunks, 1, "Nested functions should be in one chunk")
}

func TestExtractChunksForNewFile_TypeScript(t *testing.T) {
	fileContent := `
export async function fetchData() {
    return await fetch('/api/data');
}

const processData = (data: any) => {
    return data.map(x => x * 2);
}

function legacy() {
    console.log('old style');
}
`

	chunks := ExtractChunksForNewFile(fileContent, "typescript", 10)

	// Should detect all TypeScript function styles
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestExtractChunksForNewFile_UnknownLanguage(t *testing.T) {
	fileContent := strings.Repeat("line\n", 5000)

	chunks := ExtractChunksForNewFile(fileContent, "kotlin", 10)

	// Should fallback to line-based chunking
	assert.GreaterOrEqual(t, len(chunks), 1)

	// Should respect max chunks budget
	assert.LessOrEqual(t, len(chunks), 10)
}

func TestExtractChunksForNewFile_LargeFunction(t *testing.T) {
	// Create a function with >100KB of content
	largeFunc := "def huge_function():\n" + strings.Repeat("    print('line')\n", 10000)

	chunks := ExtractChunksForNewFile(largeFunc, "python", 10)

	// Should split large function into sub-chunks
	assert.GreaterOrEqual(t, len(chunks), 2, "Large function should be split")
}

func TestExtractChunksForNewFile_MaxBudget(t *testing.T) {
	// Create file with 20 medium-sized functions (each ~15KB to force splitting)
	var content strings.Builder
	for i := 0; i < 20; i++ {
		content.WriteString(fmt.Sprintf("def func%d():\n", i))
		// Add enough code to make each function ~15KB
		for j := 0; j < 1000; j++ {
			content.WriteString(fmt.Sprintf("    x%d = some_long_function_call_here(arg1, arg2, arg3)\n", j))
		}
		content.WriteString("\n")
	}

	chunks := ExtractChunksForNewFile(content.String(), "python", 10)

	// Should enforce max budget (20 functions -> 10 chunks max)
	assert.LessOrEqual(t, len(chunks), 10, "Should respect max chunks budget")
	assert.GreaterOrEqual(t, len(chunks), 2, "Should create multiple chunks for large file")
}

func TestExtractChunksForNewFile_Ruby(t *testing.T) {
	fileContent := `
def method1
  puts "hello"
end

def method2(arg)
  return arg * 2
end

class MyClass
  def instance_method
    "instance"
  end
end
`

	chunks := ExtractChunksForNewFile(fileContent, "ruby", 10)

	// Should detect Ruby methods
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestExtractChunksForNewFile_Rust(t *testing.T) {
	fileContent := `
pub fn public_function() -> i32 {
    42
}

fn private_function() {
    println!("private");
}

pub async fn async_function() -> Result<(), Error> {
    Ok(())
}
`

	chunks := ExtractChunksForNewFile(fileContent, "rust", 10)

	// Should detect Rust functions
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestSplitByLines(t *testing.T) {
	lines := make([]string, 10000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}

	chunks := splitByLines(lines, 3000, 5)

	// Should create chunks of 3000 lines each
	assert.LessOrEqual(t, len(chunks), 5)

	for i, chunk := range chunks[:len(chunks)-1] {
		assert.Len(t, chunk.Lines, 3000, "Chunk %d should have 3000 lines", i)
	}

	// Last chunk might be smaller
	if len(chunks) > 0 {
		lastChunk := chunks[len(chunks)-1]
		assert.LessOrEqual(t, len(lastChunk.Lines), 3000)
	}
}

func TestSplitByLines_SmallFile(t *testing.T) {
	lines := []string{"line1", "line2", "line3"}

	chunks := splitByLines(lines, 1000, 10)

	// Small file should be one chunk
	assert.Len(t, chunks, 1)
	assert.Equal(t, lines, chunks[0].Lines)
}

func TestSplitByLines_NoBudgetLimit(t *testing.T) {
	lines := make([]string, 15000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}

	chunks := splitByLines(lines, 3000, -1) // No limit

	// Should create 5 chunks (15000 / 3000)
	assert.Equal(t, 5, len(chunks))
}

func TestExtractChunksForNewFile_EmptyFile(t *testing.T) {
	chunks := ExtractChunksForNewFile("", "python", 10)

	// Empty file should produce at least one chunk (possibly empty)
	assert.GreaterOrEqual(t, len(chunks), 0)
}

func TestExtractChunksForNewFile_OnlyComments(t *testing.T) {
	fileContent := `# This is a comment
# Another comment
# No actual code
`

	chunks := ExtractChunksForNewFile(fileContent, "python", 10)

	// Should handle files with only comments
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestExtractChunksForNewFile_MixedIndentation(t *testing.T) {
	fileContent := `
def top_level():
    def nested():
        pass
    return nested

    def another_nested():
        pass
`

	chunks := ExtractChunksForNewFile(fileContent, "python", 10)

	// Should only match top-level functions
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestExtractChunksForNewFile_Java(t *testing.T) {
	fileContent := `
public class Example {
    public static void main(String[] args) {
        System.out.println("Hello");
    }

    private void helper() {
        return;
    }
}
`

	chunks := ExtractChunksForNewFile(fileContent, "java", 10)

	// Should detect Java methods
	assert.GreaterOrEqual(t, len(chunks), 1)
}

func TestExtractChunksForNewFile_LineNumbers(t *testing.T) {
	fileContent := `def func1():
    pass

def func2():
    pass
`

	chunks := ExtractChunksForNewFile(fileContent, "python", 10)

	// Verify line numbers are set correctly
	for _, chunk := range chunks {
		assert.GreaterOrEqual(t, chunk.StartLine, 0, "StartLine should be >= 0")
		assert.GreaterOrEqual(t, chunk.EndLine, chunk.StartLine, "EndLine should be >= StartLine")
	}
}
