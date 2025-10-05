package treesitter

import (
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractJavaScriptEntities extracts entities from JavaScript AST
func extractJavaScriptEntities(filePath string, root *sitter.Node, code []byte) ([]CodeEntity, error) {
	entities := []CodeEntity{}

	// Add file entity
	entities = append(entities, CodeEntity{
		Type:     "file",
		Name:     filepath.Base(filePath),
		FilePath: filePath,
		Language: "javascript",
	})

	// Walk the AST to extract entities
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}

		nodeType := node.Kind()

		switch nodeType {
		case "function_declaration":
			extractFunctionDeclaration(node, code, filePath, "javascript", &entities)

		case "arrow_function", "function_expression":
			extractArrowFunction(node, code, filePath, "javascript", &entities)

		case "class_declaration":
			extractClassDeclaration(node, code, filePath, "javascript", &entities)

		case "method_definition":
			extractMethodDefinition(node, code, filePath, "javascript", &entities)

		case "import_statement":
			extractImportStatement(node, code, filePath, "javascript", &entities)

		case "export_statement":
			// Export statements contain declarations, process children
			for i := uint(0); i < node.ChildCount(); i++ {
				walk(node.Child(uint(i)))
			}
			return
		}

		// Recurse to children
		for i := uint(0); i < node.ChildCount(); i++ {
			walk(node.Child(uint(i)))
		}
	}

	walk(root)
	return entities, nil
}

// extractFunctionDeclaration extracts a function declaration
func extractFunctionDeclaration(node *sitter.Node, code []byte, filePath, lang string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	funcName := getNodeText(nameNode, code)
	paramsNode := node.ChildByFieldName("parameters")
	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      funcName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  lang,
		Signature: fmt.Sprintf("function %s%s", funcName, params),
	})
}

// extractArrowFunction extracts arrow functions and function expressions
func extractArrowFunction(node *sitter.Node, code []byte, filePath, lang string, entities *[]CodeEntity) {
	// Arrow functions often appear in variable declarations
	parent := node.Parent()
	if parent == nil {
		return
	}

	// Check if parent is a variable declarator
	var funcName string
	if parent.Kind() == "variable_declarator" {
		nameNode := parent.ChildByFieldName("name")
		if nameNode != nil {
			funcName = getNodeText(nameNode, code)
		}
	} else if parent.Kind() == "assignment_expression" {
		leftNode := parent.ChildByFieldName("left")
		if leftNode != nil {
			funcName = getNodeText(leftNode, code)
		}
	}

	if funcName == "" {
		funcName = "<anonymous>"
	}

	paramsNode := node.ChildByFieldName("parameters")
	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      funcName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  lang,
		Signature: fmt.Sprintf("const %s = %s => ...", funcName, params),
	})
}

// extractClassDeclaration extracts a class declaration
func extractClassDeclaration(node *sitter.Node, code []byte, filePath, lang string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	className := getNodeText(nameNode, code)

	*entities = append(*entities, CodeEntity{
		Type:      "class",
		Name:      className,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  lang,
	})
}

// extractMethodDefinition extracts a method from a class
func extractMethodDefinition(node *sitter.Node, code []byte, filePath, lang string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	methodName := getNodeText(nameNode, code)
	paramsNode := node.ChildByFieldName("parameters")
	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	// Find parent class name
	className := findParentClassName(node, code)
	fullName := methodName
	if className != "" {
		fullName = fmt.Sprintf("%s.%s", className, methodName)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      fullName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  lang,
		Signature: fmt.Sprintf("%s%s", methodName, params),
	})
}

// extractImportStatement extracts import declarations
func extractImportStatement(node *sitter.Node, code []byte, filePath, lang string, entities *[]CodeEntity) {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		return
	}

	importPath := getNodeText(sourceNode, code)
	// Remove quotes
	importPath = strings.Trim(importPath, "\"'`")

	*entities = append(*entities, CodeEntity{
		Type:       "import",
		Name:       importPath,
		FilePath:   filePath,
		Language:   lang,
		ImportPath: importPath,
		StartLine:  int(node.StartPosition().Row) + 1,
		EndLine:    int(node.EndPosition().Row) + 1,
	})
}
