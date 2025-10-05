package treesitter

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// getNodeText extracts text from a node using byte offsets
func getNodeText(node *sitter.Node, code []byte) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	if int(end) > len(code) {
		end = uint(len(code))
	}
	return string(code[start:end])
}

// findParentClassName traverses up to find the containing class name
func findParentClassName(node *sitter.Node, code []byte) string {
	current := node.Parent()
	for current != nil {
		if current.Kind() == "class_declaration" {
			nameNode := current.ChildByFieldName("name")
			if nameNode != nil {
				return getNodeText(nameNode, code)
			}
		}
		current = current.Parent()
	}
	return ""
}
