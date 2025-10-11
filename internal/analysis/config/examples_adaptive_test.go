package config

import (
	"fmt"
	"testing"
)

// TestPrintConfigSelectionExamples demonstrates config selection for various repositories
func TestPrintConfigSelectionExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example output in short mode")
	}

	examples := []struct {
		name        string
		metadata    RepoMetadata
		description string
	}{
		{
			name: "Python Flask Web App",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"Flask==2.3.0",
					"SQLAlchemy==2.0.0",
				},
				DirectoryNames: []string{"app", "templates", "static"},
			},
			description: "Flask web application with templates",
		},
		{
			name: "Go Microservice",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/user-service",
					"require github.com/go-redis/redis v8.11.0",
				},
				DirectoryNames: []string{"internal", "pkg", "api"},
			},
			description: "Go backend microservice",
		},
		{
			name: "TypeScript Next.js App",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"next":  "14.0.4",
						"react": "^18.2.0",
					},
					"scripts": map[string]interface{}{
						"dev": "next dev",
					},
				},
				DirectoryNames: []string{"app", "components", "public"},
			},
			description: "Next.js full-stack web application (omnara)",
		},
		{
			name: "React SPA",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"react":     "^18.2.0",
						"react-dom": "^18.2.0",
					},
					"scripts": map[string]interface{}{
						"start": "react-scripts start",
					},
				},
				DirectoryNames: []string{"src", "components", "public"},
			},
			description: "React single-page application",
		},
		{
			name: "PyTorch ML Project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"torch==2.0.0",
					"transformers==4.30.0",
					"wandb==0.15.0",
				},
				DirectoryNames: []string{"models", "experiments", "data"},
			},
			description: "Machine learning research project",
		},
		{
			name: "Go CLI Tool",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/crisk",
					"require github.com/spf13/cobra v1.7.0",
				},
				DirectoryNames: []string{"cmd", "internal"},
			},
			description: "Command-line tool (coderisk-go)",
		},
		{
			name: "Java Spring Boot Service",
			metadata: RepoMetadata{
				PrimaryLanguage: "Java",
				Dependencies: map[string]string{
					"spring-boot-starter-web": "3.1.0",
					"spring-boot-starter-data-jpa": "3.1.0",
				},
				DirectoryNames: []string{"src/main/java", "src/main/resources"},
			},
			description: "Spring Boot backend service",
		},
	}

	t.Log("\n=== Config Selection Examples ===\n")
	for _, ex := range examples {
		config, reason := SelectConfigWithReason(ex.metadata)
		domain := InferDomain(ex.metadata)

		t.Logf("Repository: %s\n", ex.name)
		t.Logf("  Description: %s\n", ex.description)
		t.Logf("  Language: %s\n", ex.metadata.PrimaryLanguage)
		t.Logf("  Inferred Domain: %s\n", domain)
		t.Logf("  Selected Config: %s\n", config.ConfigKey)
		t.Logf("  Coupling Threshold: %d\n", config.CouplingThreshold)
		t.Logf("  Co-Change Threshold: %.2f\n", config.CoChangeThreshold)
		t.Logf("  Test Ratio Threshold: %.2f\n", config.TestRatioThreshold)
		t.Logf("  Selection Reason: %s\n", reason)
		t.Logf("\n")
	}
}

// ExampleSelectConfig demonstrates how to use SelectConfig
func ExampleSelectConfig() {
	// Example: Python Flask web application
	metadata := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{
			"Flask==2.3.0",
			"SQLAlchemy==2.0.0",
		},
		DirectoryNames: []string{"app", "templates", "static"},
	}

	config := SelectConfig(metadata)

	fmt.Printf("Selected config: %s\n", config.ConfigKey)
	fmt.Printf("Coupling threshold: %d\n", config.CouplingThreshold)

	// Output:
	// Selected config: python_web
	// Coupling threshold: 15
}

// ExampleSelectConfigWithReason demonstrates config selection with reasoning
func ExampleSelectConfigWithReason() {
	// Example: Go microservice
	metadata := RepoMetadata{
		PrimaryLanguage: "Go",
		DirectoryNames:  []string{"internal", "pkg", "services"},
	}

	config, reason := SelectConfigWithReason(metadata)

	fmt.Printf("Config: %s\n", config.ConfigKey)
	fmt.Printf("Reason: %s\n", reason)

	// Output:
	// Config: go_backend
	// Reason: Exact match: language=go, domain=backend → config=go_backend
}

// TestCompareConfigs_Examples shows how thresholds differ between domains
func TestCompareConfigs_Examples(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison examples in short mode")
	}

	comparisons := []struct {
		repo1 string
		repo2 string
		meta1 RepoMetadata
		meta2 RepoMetadata
	}{
		{
			repo1: "React Frontend",
			repo2: "Go Backend",
			meta1: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{"react": "^18.2.0"},
				},
			},
			meta2: RepoMetadata{
				PrimaryLanguage: "Go",
				DirectoryNames:  []string{"internal", "pkg"},
			},
		},
		{
			repo1: "Python Web",
			repo2: "Python ML",
			meta1: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"Flask==2.3.0"},
			},
			meta2: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{"tensorflow==2.13.0"},
			},
		},
	}

	t.Log("\n=== Config Comparisons ===\n")
	for _, comp := range comparisons {
		config1 := SelectConfig(comp.meta1)
		config2 := SelectConfig(comp.meta2)

		t.Logf("Comparing: %s vs %s\n", comp.repo1, comp.repo2)
		t.Logf("  %s (%s):\n", comp.repo1, config1.ConfigKey)
		t.Logf("    Coupling: %d, Co-Change: %.2f, Test Ratio: %.2f\n",
			config1.CouplingThreshold, config1.CoChangeThreshold, config1.TestRatioThreshold)
		t.Logf("  %s (%s):\n", comp.repo2, config2.ConfigKey)
		t.Logf("    Coupling: %d, Co-Change: %.2f, Test Ratio: %.2f\n",
			config2.CouplingThreshold, config2.CoChangeThreshold, config2.TestRatioThreshold)
		t.Logf("  Differences:\n")
		t.Logf("    Coupling: %+d (%s is %+.0f%% more permissive)\n",
			config1.CouplingThreshold-config2.CouplingThreshold,
			comp.repo1,
			float64(config1.CouplingThreshold-config2.CouplingThreshold)/float64(config2.CouplingThreshold)*100)
		t.Logf("    Test Ratio: %+.2f (%s expects %+.0f%% more coverage)\n",
			config1.TestRatioThreshold-config2.TestRatioThreshold,
			comp.repo1,
			(config1.TestRatioThreshold-config2.TestRatioThreshold)*100)
		t.Logf("\n")
	}
}
