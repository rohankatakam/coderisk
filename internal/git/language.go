package git

import (
	"path/filepath"
	"strings"
)

// DetectLanguage returns the programming language based on file extension
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "Go",
		".py":    "Python",
		".js":    "JavaScript",
		".jsx":   "JavaScript",
		".ts":    "TypeScript",
		".tsx":   "TypeScript",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".cc":    "C++",
		".cxx":   "C++",
		".h":     "C/C++",
		".hpp":   "C++",
		".cs":    "C#",
		".rb":    "Ruby",
		".php":   "PHP",
		".rs":    "Rust",
		".swift": "Swift",
		".kt":    "Kotlin",
		".scala": "Scala",
		".sh":    "Shell",
		".bash":  "Shell",
		".sql":   "SQL",
		".r":     "R",
		".m":     "Objective-C",
		".pl":    "Perl",
		".lua":   "Lua",
		".vim":   "Vimscript",
		".dart":  "Dart",
		".ex":    "Elixir",
		".exs":   "Elixir",
		".clj":   "Clojure",
		".fs":    "F#",
		".ml":    "OCaml",
		".hs":    "Haskell",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	// Default to the extension without the dot
	if ext != "" {
		return strings.TrimPrefix(ext, ".")
	}

	return "unknown"
}
