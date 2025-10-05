package treesitter

import (
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractPythonEntities extracts entities from Python AST
func extractPythonEntities(filePath string, root *sitter.Node, code []byte) ([]CodeEntity, error) {
	entities := []CodeEntity{}

	// Add file entity
	entities = append(entities, CodeEntity{
		Type:     "file",
		Name:     filepath.Base(filePath),
		FilePath: filePath,
		Language: "python",
	})

	// Walk the AST to extract entities
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}

		nodeType := node.Kind()

		switch nodeType {
		case "function_definition":
			extractPythonFunctionDefinition(node, code, filePath, &entities)

		case "class_definition":
			extractPythonClassDefinition(node, code, filePath, &entities)

		case "import_statement", "import_from_statement":
			extractPythonImportStatement(node, code, filePath, &entities)
		}

		// Recurse to children
		for i := uint(0); i < node.ChildCount(); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return entities, nil
}

// extractPythonFunctionDefinition extracts a Python function definition
func extractPythonFunctionDefinition(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
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

	signature := fmt.Sprintf("def %s%s", funcName, params)
	if returnTypeNode != nil {
		signature += " -> " + getNodeText(returnTypeNode, code)
	}

	// Check if this is a method (inside a class)
	className := findPythonParentClassName(node, code)
	fullName := funcName
	if className != "" {
		fullName = fmt.Sprintf("%s.%s", className, funcName)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "function",
		Name:      fullName,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "python",
		Signature: signature,
	})
}

// extractPythonClassDefinition extracts a Python class definition
func extractPythonClassDefinition(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	className := getNodeText(nameNode, code)

	// Get base classes if any
	superclassesNode := node.ChildByFieldName("superclasses")
	var signature string
	if superclassesNode != nil {
		signature = fmt.Sprintf("class %s%s", className, getNodeText(superclassesNode, code))
	} else {
		signature = fmt.Sprintf("class %s", className)
	}

	*entities = append(*entities, CodeEntity{
		Type:      "class",
		Name:      className,
		FilePath:  filePath,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Language:  "python",
		Signature: signature,
	})
}

// extractPythonImportStatement extracts Python import statements
func extractPythonImportStatement(node *sitter.Node, code []byte, filePath string, entities *[]CodeEntity) {
	nodeType := node.Kind()

	if nodeType == "import_statement" {
		// import module
		// import module.submodule
		// import module as alias
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			// Handle dotted_name
			if nameNode.Kind() == "dotted_name" {
				importPath := getNodeText(nameNode, code)
				*entities = append(*entities, CodeEntity{
					Type:       "import",
					Name:       importPath,
					FilePath:   filePath,
					Language:   "python",
					ImportPath: importPath,
					StartLine:  int(node.StartPosition().Row) + 1,
					EndLine:    int(node.EndPosition().Row) + 1,
				})
			} else {
				// Single identifier
				importPath := getNodeText(nameNode, code)
				*entities = append(*entities, CodeEntity{
					Type:       "import",
					Name:       importPath,
					FilePath:   filePath,
					Language:   "python",
					ImportPath: importPath,
					StartLine:  int(node.StartPosition().Row) + 1,
					EndLine:    int(node.EndPosition().Row) + 1,
				})
			}
		}
	} else if nodeType == "import_from_statement" {
		// from module import name
		// from module.submodule import name
		moduleNode := node.ChildByFieldName("module_name")
		if moduleNode != nil {
			importPath := getNodeText(moduleNode, code)

			// Also capture what's being imported
			for i := uint(0); i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child.Kind() == "dotted_name" && child != moduleNode {
					importPath = importPath + "." + getNodeText(child, code)
				}
			}

			*entities = append(*entities, CodeEntity{
				Type:       "import",
				Name:       importPath,
				FilePath:   filePath,
				Language:   "python",
				ImportPath: importPath,
				StartLine:  int(node.StartPosition().Row) + 1,
				EndLine:    int(node.EndPosition().Row) + 1,
			})
		}
	}
}

// findPythonParentClassName finds the containing class name for Python methods
func findPythonParentClassName(node *sitter.Node, code []byte) string {
	current := node.Parent()
	for current != nil {
		if current.Kind() == "class_definition" {
			nameNode := current.ChildByFieldName("name")
			if nameNode != nil {
				return getNodeText(nameNode, code)
			}
		}
		current = current.Parent()
	}
	return ""
}
