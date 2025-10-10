package treesitter

import (
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractTypeScriptEntities extracts entities from TypeScript AST
// TypeScript shares most node types with JavaScript but adds type annotations
func extractTypeScriptEntities(filePath string, root *sitter.Node, code []byte) ([]CodeEntity, error) {
	entities := []CodeEntity{}

	// Add file entity
	entities = append(entities, CodeEntity{
		Type:     "file",
		Name:     filepath.Base(filePath),
		FilePath: filePath,
		Language: "typescript",
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
			extractTSFunctionDeclaration(node, code, filePath, &entities)

		case "arrow_function", "function_expression":
			extractTSArrowFunction(node, code, filePath, &entities)

		case "class_declaration":
			extractTSClassDeclaration(node, code, filePath, &entities)

		case "method_definition", "method_signature":
			extractTSMethodDefinition(node, code, filePath, &entities)

		case "interface_declaration":
			extractTSInterfaceDeclaration(node, code, filePath, &entities)

		case "type_alias_declaration":
			extractTSTypeAlias(node, code, filePath, &entities)

		case "import_statement":
			extractTSImportStatement(node, code, filePath, &entities)

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

// extractTSFunctionDeclaration extracts a TypeScript function declaration
func extractTSFunctionDeclaration(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	funcName := getNodeText(nameNode, code)
	paramsNode := node.ChildByFieldName("parameters")
	returnTypeNode := node.ChildByFieldName("return_type")

	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	signature := fmt.Sprintf("function %s%s", funcName, params)
	if returnTypeNode != nil {
		signature += ": " + getNodeText(returnTypeNode, code)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      funcName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "typescript",
		Signature: signature,
	})
}

// extractTSArrowFunction extracts TypeScript arrow functions
func extractTSArrowFunction(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	parent := node.Parent()
	if parent == nil {
		return
	}

	// Get function name from variable declaration
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
	returnTypeNode := node.ChildByFieldName("return_type")

	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	signature := fmt.Sprintf("const %s = %s =>", funcName, params)
	if returnTypeNode != nil {
		signature += ": " + getNodeText(returnTypeNode, code)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      funcName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "typescript",
		Signature: signature,
	})
}

// extractTSClassDeclaration extracts a TypeScript class
func extractTSClassDeclaration(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
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
		Language:  "typescript",
	})
}

// extractTSMethodDefinition extracts a method from a TypeScript class
func extractTSMethodDefinition(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	methodName := getNodeText(nameNode, code)
	paramsNode := node.ChildByFieldName("parameters")
	returnTypeNode := node.ChildByFieldName("return_type")

	params := ""
	if paramsNode != nil {
		params = getNodeText(paramsNode, code)
	}

	signature := fmt.Sprintf("%s%s", methodName, params)
	if returnTypeNode != nil {
		signature += ": " + getNodeText(returnTypeNode, code)
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
		Language:  "typescript",
		Signature: signature,
	})
}

// extractTSInterfaceDeclaration extracts TypeScript interface declarations
func extractTSInterfaceDeclaration(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	interfaceName := getNodeText(nameNode, code)

	*entities = append(*entities, CodeEntity{
		Type:      "class", // Treat interfaces as classes for graph purposes
		Name:      interfaceName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "typescript",
	})
}

// extractTSTypeAlias extracts TypeScript type aliases
func extractTSTypeAlias(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	typeName := getNodeText(nameNode, code)

	*entities = append(*entities, CodeEntity{
		Type:      "class", // Treat type aliases as classes for graph purposes
		Name:      typeName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "typescript",
	})
}

// extractTSImportStatement extracts TypeScript import declarations
func extractTSImportStatement(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
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
		Language:   "typescript",
		ImportPath: importPath,
		StartLine:  int(node.StartPosition().Row) + 1,
		EndLine:    int(node.EndPosition().Row) + 1,
	})
}
