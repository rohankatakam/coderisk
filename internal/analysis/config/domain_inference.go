package config

import (
	"path/filepath"
	"strings"
)

// Domain represents the type of application/project
type Domain string

const (
	DomainWeb      Domain = "web"      // Web applications (Flask, Django, Rails, Express)
	DomainBackend  Domain = "backend"  // Backend services (APIs, microservices)
	DomainFrontend Domain = "frontend" // Frontend applications (React, Vue, Angular)
	DomainML       Domain = "ml"       // Machine learning projects
	DomainCLI      Domain = "cli"      // Command-line tools
	DomainUnknown  Domain = "unknown"  // Cannot determine domain
)

// RepoMetadata contains repository information for domain inference
// 12-factor: Factor 3 - Own your context window (minimal metadata needed)
type RepoMetadata struct {
	PrimaryLanguage string            // e.g., "Python", "Go", "TypeScript"
	FilePaths       []string          // Sample of file paths from repo
	Dependencies    map[string]string // Package dependencies (name -> version)
	DirectoryNames  []string          // Top-level directory names
	PackageJSON     map[string]interface{} // package.json content (if exists)
	RequirementsTxt []string          // requirements.txt lines (if exists)
	GoMod           []string          // go.mod lines (if exists)
}

// InferDomain determines the application domain based on repository metadata
// 12-factor: Factor 8 - Own your control flow (explicit decision tree)
func InferDomain(metadata RepoMetadata) Domain {
	// Priority 1: Check for web framework indicators (highest signal)
	if domain := checkWebFrameworks(metadata); domain != DomainUnknown {
		return domain
	}

	// Priority 2: Check for frontend frameworks
	if domain := checkFrontendFrameworks(metadata); domain != DomainUnknown {
		return domain
	}

	// Priority 3: Check for ML/data science indicators
	if domain := checkMLFrameworks(metadata); domain != DomainUnknown {
		return domain
	}

	// Priority 4: Check for CLI tool patterns
	if domain := checkCLIPatterns(metadata); domain != DomainUnknown {
		return domain
	}

	// Priority 5: Infer from directory structure
	if domain := checkDirectoryStructure(metadata); domain != DomainUnknown {
		return domain
	}

	// Default: Assume backend if language is typically used for services
	if isBackendLanguage(metadata.PrimaryLanguage) {
		return DomainBackend
	}

	return DomainUnknown
}

// checkWebFrameworks detects web framework usage
func checkWebFrameworks(metadata RepoMetadata) Domain {
	// Python web frameworks
	pythonWebFrameworks := []string{
		"flask", "django", "fastapi", "pyramid", "bottle",
		"tornado", "aiohttp", "sanic", "starlette",
	}
	for _, fw := range pythonWebFrameworks {
		if hasDependency(metadata, fw) {
			return DomainWeb
		}
	}

	// Ruby web frameworks
	rubyWebFrameworks := []string{"rails", "sinatra", "hanami"}
	for _, fw := range rubyWebFrameworks {
		if hasDependency(metadata, fw) {
			return DomainWeb
		}
	}

	// JavaScript/TypeScript web frameworks (server-side)
	jsWebFrameworks := []string{
		"express", "koa", "hapi", "nest", "fastify",
		"next", "nuxt", "remix",
	}
	for _, fw := range jsWebFrameworks {
		if hasDependency(metadata, fw) {
			// Next/Nuxt/Remix are hybrid (frontend + backend)
			if fw == "next" || fw == "nuxt" || fw == "remix" {
				return DomainWeb
			}
			return DomainWeb
		}
	}

	// PHP web frameworks
	phpWebFrameworks := []string{"laravel", "symfony", "codeigniter"}
	for _, fw := range phpWebFrameworks {
		if hasDependency(metadata, fw) {
			return DomainWeb
		}
	}

	// Java web frameworks
	javaWebFrameworks := []string{
		"spring-boot", "spring-web", "jakarta.servlet",
		"javax.servlet", "jersey", "dropwizard",
	}
	for _, fw := range javaWebFrameworks {
		if hasDependency(metadata, fw) {
			return DomainWeb
		}
	}

	// Go web frameworks
	goWebFrameworks := []string{
		"gin", "gin-gonic", "echo", "fiber", "chi",
		"gorilla/mux", "beego", "revel",
	}
	for _, fw := range goWebFrameworks {
		if hasDependency(metadata, fw) {
			return DomainWeb
		}
	}

	return DomainUnknown
}

// checkFrontendFrameworks detects frontend framework usage
func checkFrontendFrameworks(metadata RepoMetadata) Domain {
	// Check package.json scripts for frontend indicators
	if packageJSON, ok := metadata.PackageJSON["scripts"].(map[string]interface{}); ok {
		for scriptName, scriptCmd := range packageJSON {
			cmdStr, ok := scriptCmd.(string)
			if !ok {
				continue
			}

			// React indicators
			if strings.Contains(cmdStr, "react-scripts") ||
				strings.Contains(cmdStr, "vite") && strings.Contains(cmdStr, "react") {
				return DomainFrontend
			}

			// Vue indicators
			if strings.Contains(cmdStr, "vue-cli-service") ||
				strings.Contains(cmdStr, "@vue/cli") {
				return DomainFrontend
			}

			// Angular indicators
			if strings.Contains(cmdStr, "@angular/cli") ||
				strings.Contains(cmdStr, "ng serve") {
				return DomainFrontend
			}

			// Svelte indicators
			if strings.Contains(cmdStr, "svelte") && scriptName == "dev" {
				return DomainFrontend
			}
		}
	}

	// Check dependencies for frontend frameworks
	frontendFrameworks := []string{
		"react", "react-dom", "vue", "@angular/core",
		"svelte", "preact", "solid-js",
	}
	for _, fw := range frontendFrameworks {
		if hasDependency(metadata, fw) {
			return DomainFrontend
		}
	}

	return DomainUnknown
}

// checkMLFrameworks detects ML/data science frameworks
func checkMLFrameworks(metadata RepoMetadata) Domain {
	mlFrameworks := []string{
		"tensorflow", "torch", "pytorch", "sklearn", "scikit-learn",
		"keras", "pandas", "numpy", "scipy", "xgboost",
		"lightgbm", "catboost", "transformers", "jax",
	}

	for _, fw := range mlFrameworks {
		if hasDependency(metadata, fw) {
			return DomainML
		}
	}

	return DomainUnknown
}

// checkCLIPatterns detects CLI tool patterns
func checkCLIPatterns(metadata RepoMetadata) Domain {
	// Check for CLI framework dependencies
	cliFrameworks := []string{
		"cobra", "cli", "urfave/cli", "spf13/cobra", // Go
		"click", "argparse", "typer",               // Python
		"commander", "yargs", "oclif",              // JavaScript
	}

	for _, fw := range cliFrameworks {
		if hasDependency(metadata, fw) {
			return DomainCLI
		}
	}

	// Check for typical CLI directory structure
	for _, dirName := range metadata.DirectoryNames {
		if dirName == "cmd" || dirName == "commands" || dirName == "cli" {
			return DomainCLI
		}
	}

	// Check file paths for main executables
	hasMainExecutable := false
	for _, path := range metadata.FilePaths {
		base := filepath.Base(path)
		dir := filepath.Dir(path)

		// Go pattern: cmd/<name>/main.go
		if strings.HasPrefix(dir, "cmd/") && base == "main.go" {
			hasMainExecutable = true
		}

		// Python pattern: <name>/__main__.py or cli.py
		if base == "__main__.py" || base == "cli.py" {
			hasMainExecutable = true
		}
	}

	// If has CLI structure but no web/frontend frameworks, likely CLI tool
	if hasMainExecutable && !hasWebOrFrontendDeps(metadata) {
		return DomainCLI
	}

	return DomainUnknown
}

// checkDirectoryStructure infers domain from directory names
func checkDirectoryStructure(metadata RepoMetadata) Domain {
	dirSet := make(map[string]bool)
	for _, dir := range metadata.DirectoryNames {
		dirSet[strings.ToLower(dir)] = true
	}

	// Frontend indicators
	frontendDirs := []string{"components", "pages", "public", "static", "assets"}
	frontendScore := 0
	for _, dir := range frontendDirs {
		if dirSet[dir] {
			frontendScore++
		}
	}
	if frontendScore >= 2 {
		return DomainFrontend
	}

	// Backend indicators
	backendDirs := []string{"api", "server", "services", "handlers", "controllers"}
	backendScore := 0
	for _, dir := range backendDirs {
		if dirSet[dir] {
			backendScore++
		}
	}
	if backendScore >= 2 {
		return DomainBackend
	}

	// Web indicators (hybrid)
	webDirs := []string{"templates", "views", "routes"}
	webScore := 0
	for _, dir := range webDirs {
		if dirSet[dir] {
			webScore++
		}
	}
	if webScore >= 2 {
		return DomainWeb
	}

	return DomainUnknown
}

// hasDependency checks if a dependency exists (case-insensitive, partial match)
func hasDependency(metadata RepoMetadata, depName string) bool {
	depNameLower := strings.ToLower(depName)

	// Check Dependencies map
	for name := range metadata.Dependencies {
		if strings.Contains(strings.ToLower(name), depNameLower) {
			return true
		}
	}

	// Check requirements.txt lines
	for _, line := range metadata.RequirementsTxt {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, depNameLower) {
			return true
		}
	}

	// Check go.mod lines
	for _, line := range metadata.GoMod {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, depNameLower) {
			return true
		}
	}

	// Check package.json dependencies
	if deps, ok := metadata.PackageJSON["dependencies"].(map[string]interface{}); ok {
		for name := range deps {
			if strings.Contains(strings.ToLower(name), depNameLower) {
				return true
			}
		}
	}

	if devDeps, ok := metadata.PackageJSON["devDependencies"].(map[string]interface{}); ok {
		for name := range devDeps {
			if strings.Contains(strings.ToLower(name), depNameLower) {
				return true
			}
		}
	}

	return false
}

// hasWebOrFrontendDeps checks if repo has web/frontend dependencies
func hasWebOrFrontendDeps(metadata RepoMetadata) bool {
	return checkWebFrameworks(metadata) != DomainUnknown ||
		checkFrontendFrameworks(metadata) != DomainUnknown
}

// isBackendLanguage returns true if language is typically used for backend services
func isBackendLanguage(language string) bool {
	backendLanguages := map[string]bool{
		"Go":     true,
		"Java":   true,
		"Rust":   true,
		"C#":     true,
		"Kotlin": true,
		"Scala":  true,
		"Elixir": true,
	}

	return backendLanguages[language]
}
