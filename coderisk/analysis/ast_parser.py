"""
AST Parser for Multiple Languages

Provides unified AST parsing capabilities for Python, JavaScript, TypeScript,
Java, and other languages using appropriate parsers.
"""

import ast
import re
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass
from pathlib import Path
from enum import Enum

class Language(Enum):
    PYTHON = "python"
    JAVASCRIPT = "javascript"
    TYPESCRIPT = "typescript"
    JAVA = "java"
    GO = "go"
    RUST = "rust"
    UNKNOWN = "unknown"


@dataclass
class ASTNode:
    """Unified AST node representation"""
    type: str
    name: Optional[str]
    line_number: int
    column: Optional[int]
    children: List['ASTNode']
    metadata: Dict[str, Any]

    def find_children_by_type(self, node_type: str) -> List['ASTNode']:
        """Find all children of a specific type"""
        result = []
        if self.type == node_type:
            result.append(self)

        for child in self.children:
            result.extend(child.find_children_by_type(node_type))

        return result


@dataclass
class ParseResult:
    """Result of AST parsing"""
    success: bool
    language: Language
    root_node: Optional[ASTNode]
    error_message: Optional[str]
    parse_time_ms: float

    def find_functions(self) -> List[ASTNode]:
        """Find all function definitions"""
        if not self.root_node:
            return []

        function_types = ['function', 'method', 'async_function', 'function_def']
        functions = []

        for func_type in function_types:
            functions.extend(self.root_node.find_children_by_type(func_type))

        return functions

    def find_classes(self) -> List[ASTNode]:
        """Find all class definitions"""
        if not self.root_node:
            return []

        return self.root_node.find_children_by_type('class_def')

    def find_imports(self) -> List[ASTNode]:
        """Find all import statements"""
        if not self.root_node:
            return []

        import_types = ['import', 'import_from', 'import_statement']
        imports = []

        for import_type in import_types:
            imports.extend(self.root_node.find_children_by_type(import_type))

        return imports


class ASTParser:
    """Multi-language AST parser"""

    def __init__(self):
        self.language_detectors = {
            '.py': Language.PYTHON,
            '.js': Language.JAVASCRIPT,
            '.jsx': Language.JAVASCRIPT,
            '.ts': Language.TYPESCRIPT,
            '.tsx': Language.TYPESCRIPT,
            '.java': Language.JAVA,
            '.go': Language.GO,
            '.rs': Language.RUST,
        }

    def detect_language(self, file_path: str) -> Language:
        """Detect programming language from file path"""
        path = Path(file_path)
        suffix = path.suffix.lower()

        return self.language_detectors.get(suffix, Language.UNKNOWN)

    def parse_file(self, file_path: str) -> ParseResult:
        """Parse a file and return AST"""
        import time
        start_time = time.perf_counter()

        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()

            language = self.detect_language(file_path)
            parse_time_ms = (time.perf_counter() - start_time) * 1000

            return self.parse_content(content, language, parse_time_ms)

        except Exception as e:
            parse_time_ms = (time.perf_counter() - start_time) * 1000
            return ParseResult(
                success=False,
                language=Language.UNKNOWN,
                root_node=None,
                error_message=str(e),
                parse_time_ms=parse_time_ms
            )

    def parse_content(self, content: str, language: Language, parse_time_ms: float = 0.0) -> ParseResult:
        """Parse content string and return AST"""
        try:
            if language == Language.PYTHON:
                return self._parse_python(content, parse_time_ms)
            elif language in [Language.JAVASCRIPT, Language.TYPESCRIPT]:
                return self._parse_javascript(content, language, parse_time_ms)
            elif language == Language.JAVA:
                return self._parse_java(content, parse_time_ms)
            else:
                return self._parse_generic(content, language, parse_time_ms)

        except Exception as e:
            return ParseResult(
                success=False,
                language=language,
                root_node=None,
                error_message=str(e),
                parse_time_ms=parse_time_ms
            )

    def _parse_python(self, content: str, parse_time_ms: float) -> ParseResult:
        """Parse Python code using ast module"""
        try:
            tree = ast.parse(content)
            root_node = self._convert_python_ast(tree)

            return ParseResult(
                success=True,
                language=Language.PYTHON,
                root_node=root_node,
                error_message=None,
                parse_time_ms=parse_time_ms
            )

        except SyntaxError as e:
            return ParseResult(
                success=False,
                language=Language.PYTHON,
                root_node=None,
                error_message=f"Python syntax error: {e}",
                parse_time_ms=parse_time_ms
            )

    def _parse_javascript(self, content: str, language: Language, parse_time_ms: float) -> ParseResult:
        """Parse JavaScript/TypeScript using regex-based approach"""
        # This is a simplified parser - in production would use tree-sitter or similar
        root_node = self._parse_js_regex(content)

        return ParseResult(
            success=True,
            language=language,
            root_node=root_node,
            error_message=None,
            parse_time_ms=parse_time_ms
        )

    def _parse_java(self, content: str, parse_time_ms: float) -> ParseResult:
        """Parse Java using regex-based approach"""
        root_node = self._parse_java_regex(content)

        return ParseResult(
            success=True,
            language=Language.JAVA,
            root_node=root_node,
            error_message=None,
            parse_time_ms=parse_time_ms
        )

    def _parse_generic(self, content: str, language: Language, parse_time_ms: float) -> ParseResult:
        """Generic parsing for unsupported languages"""
        root_node = ASTNode(
            type="module",
            name=None,
            line_number=1,
            column=0,
            children=[],
            metadata={"language": language.value, "content_lines": len(content.split('\n'))}
        )

        return ParseResult(
            success=True,
            language=language,
            root_node=root_node,
            error_message=None,
            parse_time_ms=parse_time_ms
        )

    def _convert_python_ast(self, node: ast.AST, parent_line: int = 1) -> ASTNode:
        """Convert Python AST to unified format"""
        node_type = type(node).__name__.lower()
        line_number = getattr(node, 'lineno', parent_line)
        column = getattr(node, 'col_offset', None)

        # Extract node name if available
        name = None
        if hasattr(node, 'name'):
            name = node.name
        elif hasattr(node, 'id'):
            name = node.id
        elif isinstance(node, ast.FunctionDef):
            name = node.name
        elif isinstance(node, ast.ClassDef):
            name = node.name

        # Extract metadata
        metadata = {}
        if isinstance(node, ast.FunctionDef):
            metadata['args'] = [arg.arg for arg in node.args.args]
            metadata['decorators'] = [ast.get_source_segment("", decorator) or str(decorator)
                                    for decorator in node.decorator_list]
            metadata['is_async'] = isinstance(node, ast.AsyncFunctionDef)
        elif isinstance(node, ast.ClassDef):
            metadata['bases'] = [ast.get_source_segment("", base) or str(base) for base in node.bases]
            metadata['decorators'] = [ast.get_source_segment("", decorator) or str(decorator)
                                    for decorator in node.decorator_list]
        elif isinstance(node, (ast.Import, ast.ImportFrom)):
            if isinstance(node, ast.ImportFrom):
                metadata['module'] = node.module
                metadata['level'] = node.level
            metadata['names'] = [(alias.name, alias.asname) for alias in node.names]

        # Convert children
        children = []
        for child in ast.iter_child_nodes(node):
            children.append(self._convert_python_ast(child, line_number))

        return ASTNode(
            type=node_type,
            name=name,
            line_number=line_number,
            column=column,
            children=children,
            metadata=metadata
        )

    def _parse_js_regex(self, content: str) -> ASTNode:
        """Parse JavaScript using regex patterns"""
        lines = content.split('\n')
        children = []

        for line_num, line in enumerate(lines, 1):
            line = line.strip()

            # Function declarations
            func_match = re.match(r'(?:async\s+)?(?:function\s+)?(\w+)\s*\(([^)]*)\)', line)
            if func_match:
                func_name = func_match.group(1)
                args = [arg.strip() for arg in func_match.group(2).split(',') if arg.strip()]

                children.append(ASTNode(
                    type='function',
                    name=func_name,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'args': args, 'is_async': 'async' in line}
                ))

            # Class declarations
            class_match = re.match(r'class\s+(\w+)(?:\s+extends\s+(\w+))?', line)
            if class_match:
                class_name = class_match.group(1)
                extends = class_match.group(2)

                children.append(ASTNode(
                    type='class_def',
                    name=class_name,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'extends': extends}
                ))

            # Import statements
            import_match = re.match(r'import\s+.*?from\s+["\']([^"\']+)["\']', line)
            if import_match:
                module = import_match.group(1)

                children.append(ASTNode(
                    type='import',
                    name=None,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'module': module}
                ))

        return ASTNode(
            type='module',
            name=None,
            line_number=1,
            column=0,
            children=children,
            metadata={'language': 'javascript'}
        )

    def _parse_java_regex(self, content: str) -> ASTNode:
        """Parse Java using regex patterns"""
        lines = content.split('\n')
        children = []

        for line_num, line in enumerate(lines, 1):
            line = line.strip()

            # Class declarations
            class_match = re.match(r'(?:public\s+)?(?:abstract\s+)?class\s+(\w+)', line)
            if class_match:
                class_name = class_match.group(1)

                children.append(ASTNode(
                    type='class_def',
                    name=class_name,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'access': 'public' if 'public' in line else 'package'}
                ))

            # Method declarations
            method_match = re.match(r'(?:public|private|protected)?\s*(?:static\s+)?(?:final\s+)?\w+\s+(\w+)\s*\(([^)]*)\)', line)
            if method_match:
                method_name = method_match.group(1)
                args = [arg.strip() for arg in method_match.group(2).split(',') if arg.strip()]

                children.append(ASTNode(
                    type='method',
                    name=method_name,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'args': args}
                ))

            # Import statements
            import_match = re.match(r'import\s+(?:static\s+)?([^;]+);', line)
            if import_match:
                imported = import_match.group(1)

                children.append(ASTNode(
                    type='import',
                    name=None,
                    line_number=line_num,
                    column=0,
                    children=[],
                    metadata={'imported': imported}
                ))

        return ASTNode(
            type='module',
            name=None,
            line_number=1,
            column=0,
            children=children,
            metadata={'language': 'java'}
        )