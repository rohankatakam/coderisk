package config

import (
	"fmt"
	"testing"
)

// ExampleInferDomain demonstrates domain inference for various repository types
func ExampleInferDomain() {
	// Example 1: Python Flask web application
	pythonWebRepo := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{
			"Flask==2.3.0",
			"SQLAlchemy==2.0.0",
			"pytest==7.4.0",
		},
		DirectoryNames: []string{"app", "templates", "static"},
	}
	fmt.Printf("Python Flask app: %s\n", InferDomain(pythonWebRepo))

	// Example 2: Go backend microservice
	goBackendRepo := RepoMetadata{
		PrimaryLanguage: "Go",
		GoMod: []string{
			"module github.com/example/service",
			"require github.com/go-redis/redis v8.11.0",
		},
		DirectoryNames: []string{"internal", "pkg", "services"},
	}
	fmt.Printf("Go microservice: %s\n", InferDomain(goBackendRepo))

	// Example 3: React frontend application
	reactFrontendRepo := RepoMetadata{
		PrimaryLanguage: "TypeScript",
		PackageJSON: map[string]interface{}{
			"dependencies": map[string]interface{}{
				"react":     "^18.2.0",
				"react-dom": "^18.2.0",
			},
			"scripts": map[string]interface{}{
				"start": "react-scripts start",
				"build": "react-scripts build",
			},
		},
		DirectoryNames: []string{"src", "public", "components"},
	}
	fmt.Printf("React frontend: %s\n", InferDomain(reactFrontendRepo))

	// Example 4: Machine learning project
	mlRepo := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{
			"tensorflow==2.13.0",
			"pandas==2.0.0",
			"scikit-learn==1.3.0",
		},
		DirectoryNames: []string{"models", "data", "notebooks"},
	}
	fmt.Printf("ML project: %s\n", InferDomain(mlRepo))

	// Example 5: CLI tool
	cliRepo := RepoMetadata{
		PrimaryLanguage: "Go",
		GoMod: []string{
			"module github.com/example/cli",
			"require github.com/spf13/cobra v1.7.0",
		},
		DirectoryNames: []string{"cmd", "internal"},
		FilePaths:      []string{"cmd/root.go", "cmd/main.go"},
	}
	fmt.Printf("CLI tool: %s\n", InferDomain(cliRepo))

	// Output:
	// Python Flask app: web
	// Go microservice: backend
	// React frontend: frontend
	// ML project: ml
	// CLI tool: cli
}

// TestPrintDomainExamples prints detailed examples for manual review
func TestPrintDomainExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example output in short mode")
	}

	examples := []struct {
		name        string
		metadata    RepoMetadata
		description string
	}{
		{
			name: "omnara (test repository)",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"next":  "14.0.4",
						"react": "^18",
					},
					"scripts": map[string]interface{}{
						"dev":   "next dev",
						"build": "next build",
					},
				},
				DirectoryNames: []string{"app", "components", "lib", "public"},
			},
			description: "Next.js full-stack web application",
		},
		{
			name: "coderisk-go (this project)",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/rohankatakam/coderisk",
					"require github.com/spf13/cobra v1.7.0",
					"require github.com/neo4j/neo4j-go-driver/v5 v5.13.0",
				},
				DirectoryNames: []string{"cmd", "internal", "pkg"},
				FilePaths:      []string{"cmd/crisk/main.go"},
			},
			description: "CLI tool for code risk analysis",
		},
		{
			name: "Django REST API",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"Django==4.2.0",
					"djangorestframework==3.14.0",
					"celery==5.3.0",
				},
				DirectoryNames: []string{"api", "models", "serializers"},
			},
			description: "Python REST API backend",
		},
		{
			name: "Vue.js SPA",
			metadata: RepoMetadata{
				PrimaryLanguage: "JavaScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"vue":        "^3.3.0",
						"vue-router": "^4.2.0",
					},
					"scripts": map[string]interface{}{
						"serve": "vue-cli-service serve",
					},
				},
				DirectoryNames: []string{"src", "components", "views", "router"},
			},
			description: "Vue.js single-page application",
		},
		{
			name: "PyTorch research project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"torch==2.0.0",
					"torchvision==0.15.0",
					"transformers==4.30.0",
					"wandb==0.15.0",
				},
				DirectoryNames: []string{"models", "experiments", "data"},
			},
			description: "ML research with PyTorch",
		},
	}

	t.Log("\n=== Domain Inference Examples ===\n")
	for _, ex := range examples {
		domain := InferDomain(ex.metadata)
		t.Logf("Repository: %s\n", ex.name)
		t.Logf("  Description: %s\n", ex.description)
		t.Logf("  Language: %s\n", ex.metadata.PrimaryLanguage)
		t.Logf("  Inferred Domain: %s\n", domain)
		t.Logf("  Expected: This domain will get specific risk thresholds\n\n")
	}
}
