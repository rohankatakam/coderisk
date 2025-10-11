package config

import (
	"testing"
)

func TestInferDomain_PythonWeb(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "Flask web app",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"Flask==2.3.0",
					"SQLAlchemy==2.0.0",
				},
				DirectoryNames: []string{"templates", "static", "app"},
			},
			expected: DomainWeb,
		},
		{
			name: "Django web app",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"Django==4.2.0",
					"djangorestframework==3.14.0",
				},
				DirectoryNames: []string{"templates", "static"},
			},
			expected: DomainWeb,
		},
		{
			name: "FastAPI backend service",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"fastapi==0.103.0",
					"uvicorn==0.23.0",
				},
				DirectoryNames: []string{"api", "services"},
			},
			expected: DomainWeb,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_GoBackend(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "Go backend service",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/backend",
					"require github.com/gin-gonic/gin v1.9.0",
				},
				DirectoryNames: []string{"api", "internal", "pkg"},
			},
			expected: DomainWeb,
		},
		{
			name: "Go microservice without web framework",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/service",
					"require github.com/go-redis/redis v8.11.0",
				},
				DirectoryNames: []string{"internal", "pkg", "services"},
			},
			expected: DomainBackend,
		},
		{
			name: "Go CLI tool",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/cli",
					"require github.com/spf13/cobra v1.7.0",
				},
				DirectoryNames: []string{"cmd", "internal"},
				FilePaths:      []string{"cmd/main.go", "cmd/root.go"},
			},
			expected: DomainCLI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_TypeScriptFrontend(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "React app with react-scripts",
			metadata: RepoMetadata{
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
			},
			expected: DomainFrontend,
		},
		{
			name: "Vue.js app",
			metadata: RepoMetadata{
				PrimaryLanguage: "JavaScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"vue": "^3.3.0",
					},
					"scripts": map[string]interface{}{
						"dev": "vue-cli-service serve",
					},
				},
				DirectoryNames: []string{"src", "components", "views"},
			},
			expected: DomainFrontend,
		},
		{
			name: "Angular app",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"@angular/core": "^16.0.0",
					},
					"scripts": map[string]interface{}{
						"start": "ng serve",
					},
				},
				DirectoryNames: []string{"src", "app"},
			},
			expected: DomainFrontend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_MachineLearning(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "TensorFlow ML project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"tensorflow==2.13.0",
					"numpy==1.24.0",
					"pandas==2.0.0",
				},
				DirectoryNames: []string{"models", "data", "notebooks"},
			},
			expected: DomainML,
		},
		{
			name: "PyTorch ML project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"torch==2.0.0",
					"scikit-learn==1.3.0",
					"matplotlib==3.7.0",
				},
				DirectoryNames: []string{"src", "experiments"},
			},
			expected: DomainML,
		},
		{
			name: "Data science project",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"pandas==2.0.0",
					"scikit-learn==1.3.0",
					"xgboost==1.7.0",
				},
			},
			expected: DomainML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_CLITools(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "Python CLI with Click",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"click==8.1.0",
				},
				FilePaths:      []string{"cli.py", "commands/init.py"},
				DirectoryNames: []string{"commands"},
			},
			expected: DomainCLI,
		},
		{
			name: "Node.js CLI with Commander",
			metadata: RepoMetadata{
				PrimaryLanguage: "JavaScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"commander": "^11.0.0",
					},
					"bin": map[string]interface{}{
						"mycli": "./bin/cli.js",
					},
				},
			},
			expected: DomainCLI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_MixedSignals(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
		reason   string
	}{
		{
			name: "Next.js (web takes priority over frontend)",
			metadata: RepoMetadata{
				PrimaryLanguage: "TypeScript",
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"next":  "^13.4.0",
						"react": "^18.2.0",
					},
				},
				DirectoryNames: []string{"pages", "api", "components"},
			},
			expected: DomainWeb,
			reason:   "Next.js is hybrid (server + client), classified as web",
		},
		{
			name: "Flask API (no templates, but still web)",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"flask==2.3.0",
					"flask-cors==4.0.0",
				},
				DirectoryNames: []string{"api", "models"},
			},
			expected: DomainWeb,
			reason:   "Flask presence indicates web, even without templates",
		},
		{
			name: "Pure backend service (no web framework)",
			metadata: RepoMetadata{
				PrimaryLanguage: "Go",
				GoMod: []string{
					"module github.com/example/processor",
					"require github.com/aws/aws-sdk-go v1.44.0",
				},
				DirectoryNames: []string{"internal", "pkg"},
			},
			expected: DomainBackend,
			reason:   "Go language + no web framework = backend service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.reason, tt.expected, result)
			}
		})
	}
}

func TestInferDomain_DirectoryStructure(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "Frontend structure without explicit dependencies",
			metadata: RepoMetadata{
				PrimaryLanguage: "JavaScript",
				DirectoryNames:  []string{"components", "pages", "public"},
				FilePaths:       []string{"components/Button.jsx", "pages/index.jsx"},
			},
			expected: DomainFrontend,
		},
		{
			name: "Backend structure without explicit dependencies",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				DirectoryNames:  []string{"api", "services", "models"},
				FilePaths:       []string{"api/handlers.py", "services/auth.py"},
			},
			expected: DomainBackend,
		},
		{
			name: "Web app structure",
			metadata: RepoMetadata{
				PrimaryLanguage: "Ruby",
				DirectoryNames:  []string{"views", "routes", "public"},
				FilePaths:       []string{"views/home.erb", "routes/api.rb"},
			},
			expected: DomainWeb,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferDomain_UnknownFallback(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		expected Domain
	}{
		{
			name: "Empty metadata returns unknown",
			metadata: RepoMetadata{
				PrimaryLanguage: "",
			},
			expected: DomainUnknown,
		},
		{
			name: "Generic library with no clear domain signals",
			metadata: RepoMetadata{
				PrimaryLanguage: "Python",
				RequirementsTxt: []string{
					"requests==2.31.0",
					"pytest==7.4.0",
				},
				DirectoryNames: []string{"src", "tests"},
			},
			expected: DomainUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDomain(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHasDependency(t *testing.T) {
	tests := []struct {
		name     string
		metadata RepoMetadata
		depName  string
		expected bool
	}{
		{
			name: "Direct dependency match",
			metadata: RepoMetadata{
				Dependencies: map[string]string{
					"flask": "2.3.0",
				},
			},
			depName:  "flask",
			expected: true,
		},
		{
			name: "Partial match (case insensitive)",
			metadata: RepoMetadata{
				Dependencies: map[string]string{
					"Flask-CORS": "4.0.0",
				},
			},
			depName:  "flask",
			expected: true,
		},
		{
			name: "Match in requirements.txt",
			metadata: RepoMetadata{
				RequirementsTxt: []string{
					"Flask==2.3.0",
					"requests==2.31.0",
				},
			},
			depName:  "flask",
			expected: true,
		},
		{
			name: "Match in go.mod",
			metadata: RepoMetadata{
				GoMod: []string{
					"require github.com/spf13/cobra v1.7.0",
				},
			},
			depName:  "cobra",
			expected: true,
		},
		{
			name: "Match in package.json dependencies",
			metadata: RepoMetadata{
				PackageJSON: map[string]interface{}{
					"dependencies": map[string]interface{}{
						"react": "^18.2.0",
					},
				},
			},
			depName:  "react",
			expected: true,
		},
		{
			name: "Match in package.json devDependencies",
			metadata: RepoMetadata{
				PackageJSON: map[string]interface{}{
					"devDependencies": map[string]interface{}{
						"typescript": "^5.0.0",
					},
				},
			},
			depName:  "typescript",
			expected: true,
		},
		{
			name: "No match",
			metadata: RepoMetadata{
				Dependencies: map[string]string{
					"requests": "2.31.0",
				},
			},
			depName:  "flask",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDependency(tt.metadata, tt.depName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsBackendLanguage(t *testing.T) {
	tests := []struct {
		language string
		expected bool
	}{
		{"Go", true},
		{"Java", true},
		{"Rust", true},
		{"C#", true},
		{"Kotlin", true},
		{"Scala", true},
		{"Elixir", true},
		{"Python", false}, // Python can be frontend, web, ML, etc.
		{"JavaScript", false},
		{"TypeScript", false},
		{"Ruby", false},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := isBackendLanguage(tt.language)
			if result != tt.expected {
				t.Errorf("language %s: expected %v, got %v", tt.language, tt.expected, result)
			}
		})
	}
}

// Benchmark domain inference performance
func BenchmarkInferDomain(b *testing.B) {
	metadata := RepoMetadata{
		PrimaryLanguage: "Python",
		RequirementsTxt: []string{
			"Flask==2.3.0",
			"SQLAlchemy==2.0.0",
			"pytest==7.4.0",
		},
		DirectoryNames: []string{"api", "models", "tests"},
		FilePaths:      []string{"api/handlers.py", "models/user.py"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InferDomain(metadata)
	}
}
