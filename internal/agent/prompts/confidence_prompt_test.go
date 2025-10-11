package prompts

import (
	"strings"
	"testing"
)

func TestConfidencePrompt(t *testing.T) {
	tests := []struct {
		name             string
		evidenceChain    []string
		currentRiskLevel string
		hopNumber        int
		wantContains     []string
	}{
		{
			name: "hop 1 with evidence",
			evidenceChain: []string{
				"High coupling: 12 dependencies",
				"Low test coverage: 0.25",
			},
			currentRiskLevel: "HIGH",
			hopNumber:        1,
			wantContains: []string{
				"Hop 1",
				"High coupling: 12 dependencies",
				"Low test coverage: 0.25",
				"CURRENT RISK ASSESSMENT: HIGH",
				"confidence",
				"reasoning",
				"next_action",
			},
		},
		{
			name:             "hop 2 no evidence",
			evidenceChain:    []string{},
			currentRiskLevel: "MODERATE",
			hopNumber:        2,
			wantContains: []string{
				"Hop 2",
				"(No evidence gathered yet)",
				"CURRENT RISK ASSESSMENT: MODERATE",
			},
		},
		{
			name: "hop 3 multiple evidence",
			evidenceChain: []string{
				"Security keywords detected: auth, token",
				"Past incident: INC-123 (21 days ago)",
				"Ownership transition: 14 days ago",
			},
			currentRiskLevel: "CRITICAL",
			hopNumber:        3,
			wantContains: []string{
				"Hop 3",
				"Security keywords detected",
				"Past incident: INC-123",
				"Ownership transition",
				"CURRENT RISK ASSESSMENT: CRITICAL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := ConfidencePrompt(tt.evidenceChain, tt.currentRiskLevel, tt.hopNumber)

			for _, want := range tt.wantContains {
				if !strings.Contains(prompt, want) {
					t.Errorf("ConfidencePrompt() missing expected substring %q", want)
				}
			}

			// Verify prompt structure
			if !strings.Contains(prompt, "CONFIDENCE ASSESSMENT TASK:") {
				t.Error("ConfidencePrompt() missing confidence task section")
			}
			if !strings.Contains(prompt, "NEXT ACTION OPTIONS:") {
				t.Error("ConfidencePrompt() missing next action section")
			}
		})
	}
}

func TestConfidencePromptWithModificationType(t *testing.T) {
	tests := []struct {
		name                 string
		evidenceChain        []string
		currentRiskLevel     string
		hopNumber            int
		modificationType     string
		modificationReason   string
		wantContains         []string
		wantTypeGuidance     bool
	}{
		{
			name: "security type",
			evidenceChain: []string{
				"Authentication logic modified",
				"High coupling detected",
			},
			currentRiskLevel:   "CRITICAL",
			hopNumber:          1,
			modificationType:   "SECURITY",
			modificationReason: "Security keywords: auth, login, token",
			wantContains: []string{
				"MODIFICATION TYPE: SECURITY",
				"Security keywords: auth, login, token",
				"authentication/authorization flows",
				"security edge cases",
				"sensitive data",
			},
			wantTypeGuidance: true,
		},
		{
			name: "documentation type",
			evidenceChain: []string{
				"README.md modified",
				"Comment additions only",
			},
			currentRiskLevel:   "VERY_LOW",
			hopNumber:          1,
			modificationType:   "DOCUMENTATION",
			modificationReason: "Documentation file (zero runtime impact)",
			wantContains: []string{
				"MODIFICATION TYPE: DOCUMENTATION",
				"zero runtime impact",
				"FINALIZE immediately",
				"Confidence should be very high",
			},
			wantTypeGuidance: true,
		},
		{
			name: "interface type",
			evidenceChain: []string{
				"API route added: POST /users",
				"Schema change detected",
			},
			currentRiskLevel:   "HIGH",
			hopNumber:          2,
			modificationType:   "INTERFACE",
			modificationReason: "Interface/API modification",
			wantContains: []string{
				"MODIFICATION TYPE: INTERFACE",
				"API contracts",
				"backward compatibility",
				"versioning strategy",
			},
			wantTypeGuidance: true,
		},
		{
			name: "configuration type production",
			evidenceChain: []string{
				".env.production modified",
				"DATABASE_URL changed",
			},
			currentRiskLevel:   "CRITICAL",
			hopNumber:          1,
			modificationType:   "CONFIGURATION",
			modificationReason: "Production environment configuration",
			wantContains: []string{
				"MODIFICATION TYPE: CONFIGURATION",
				"production environment",
				"rollback plan",
				"connection strings",
			},
			wantTypeGuidance: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := ConfidencePromptWithModificationType(
				tt.evidenceChain,
				tt.currentRiskLevel,
				tt.hopNumber,
				tt.modificationType,
				tt.modificationReason,
			)

			for _, want := range tt.wantContains {
				if !strings.Contains(prompt, want) {
					t.Errorf("ConfidencePromptWithModificationType() missing expected substring %q", want)
				}
			}

			if tt.wantTypeGuidance {
				if !strings.Contains(prompt, "TYPE-SPECIFIC CONSIDERATIONS:") {
					t.Error("ConfidencePromptWithModificationType() missing type-specific guidance section")
				}
			}
		})
	}
}

func TestParseConfidenceAssessment(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		wantConfidence  float64
		wantReasoning   string
		wantNextAction  string
		wantErr         bool
	}{
		{
			name: "valid finalize response",
			response: `{
  "confidence": 0.92,
  "reasoning": "High coupling (12 deps) + low test coverage (0.25) + security file = clear HIGH risk. Additional evidence unlikely to change assessment.",
  "next_action": "FINALIZE"
}`,
			wantConfidence: 0.92,
			wantReasoning:  "High coupling (12 deps) + low test coverage (0.25) + security file = clear HIGH risk. Additional evidence unlikely to change assessment.",
			wantNextAction: "FINALIZE",
			wantErr:        false,
		},
		{
			name: "valid gather more evidence response",
			response: `{
  "confidence": 0.65,
  "reasoning": "Moderate coupling but need incident history to confirm risk level.",
  "next_action": "GATHER_MORE_EVIDENCE"
}`,
			wantConfidence: 0.65,
			wantReasoning:  "Moderate coupling but need incident history to confirm risk level.",
			wantNextAction: "GATHER_MORE_EVIDENCE",
			wantErr:        false,
		},
		{
			name: "valid expand graph response",
			response: `{
  "confidence": 0.45,
  "reasoning": "Need to explore 2-hop neighbors to understand cascading impact.",
  "next_action": "EXPAND_GRAPH"
}`,
			wantConfidence: 0.45,
			wantReasoning:  "Need to explore 2-hop neighbors to understand cascading impact.",
			wantNextAction: "EXPAND_GRAPH",
			wantErr:        false,
		},
		{
			name: "response with markdown code block",
			response: "```json\n" + `{
  "confidence": 0.88,
  "reasoning": "Clear security risk with supporting evidence.",
  "next_action": "FINALIZE"
}` + "\n```",
			wantConfidence: 0.88,
			wantReasoning:  "Clear security risk with supporting evidence.",
			wantNextAction: "FINALIZE",
			wantErr:        false,
		},
		{
			name: "response with extra text before JSON",
			response: "Based on the analysis, here is my assessment:\n" + `{
  "confidence": 0.75,
  "reasoning": "Good evidence, minor gaps.",
  "next_action": "FINALIZE"
}`,
			wantConfidence: 0.75,
			wantReasoning:  "Good evidence, minor gaps.",
			wantNextAction: "FINALIZE",
			wantErr:        false,
		},
		{
			name:     "invalid confidence out of range (too high)",
			response: `{"confidence": 1.5, "reasoning": "test", "next_action": "FINALIZE"}`,
			wantErr:  true,
		},
		{
			name:     "invalid confidence out of range (negative)",
			response: `{"confidence": -0.2, "reasoning": "test", "next_action": "FINALIZE"}`,
			wantErr:  true,
		},
		{
			name:     "invalid next action",
			response: `{"confidence": 0.8, "reasoning": "test", "next_action": "CONTINUE"}`,
			wantErr:  true,
		},
		{
			name:     "malformed JSON",
			response: `{confidence: 0.8, reasoning: "test"`,
			wantErr:  true,
		},
		{
			name:           "missing reasoning field (allowed)",
			response:       `{"confidence": 0.8, "next_action": "FINALIZE"}`,
			wantConfidence: 0.8,
			wantReasoning:  "", // reasoning can be empty
			wantNextAction: "FINALIZE",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfidenceAssessment(tt.response)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfidenceAssessment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Confidence != tt.wantConfidence {
				t.Errorf("ParseConfidenceAssessment() confidence = %.2f, want %.2f", got.Confidence, tt.wantConfidence)
			}

			if got.Reasoning != tt.wantReasoning {
				t.Errorf("ParseConfidenceAssessment() reasoning = %q, want %q", got.Reasoning, tt.wantReasoning)
			}

			if got.NextAction != tt.wantNextAction {
				t.Errorf("ParseConfidenceAssessment() next_action = %q, want %q", got.NextAction, tt.wantNextAction)
			}
		})
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "clean JSON",
			response: `{"confidence": 0.8}`,
			want:     `{"confidence": 0.8}`,
		},
		{
			name:     "JSON with markdown",
			response: "```json\n{\"confidence\": 0.8}\n```",
			want:     `{"confidence": 0.8}`,
		},
		{
			name:     "JSON with extra text",
			response: "Here is the result:\n{\"confidence\": 0.8}\nThank you.",
			want:     `{"confidence": 0.8}`,
		},
		{
			name:     "JSON with whitespace",
			response: "  \n  {\"confidence\": 0.8}  \n  ",
			want:     `{"confidence": 0.8}`,
		},
		{
			name:     "no JSON found",
			response: "This is not JSON",
			want:     "This is not JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanJSONResponse(tt.response)
			if got != tt.want {
				t.Errorf("cleanJSONResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTypeSpecificGuidance(t *testing.T) {
	tests := []struct {
		name             string
		modificationType string
		wantContains     []string
	}{
		{
			name:             "security guidance",
			modificationType: "SECURITY",
			wantContains:     []string{"authentication", "authorization", "sensitive data", "incidents"},
		},
		{
			name:             "interface guidance",
			modificationType: "INTERFACE",
			wantContains:     []string{"API contracts", "backward compatibility", "versioning"},
		},
		{
			name:             "documentation guidance",
			modificationType: "DOCUMENTATION",
			wantContains:     []string{"zero runtime impact", "FINALIZE immediately", "0.95"},
		},
		{
			name:             "configuration guidance",
			modificationType: "CONFIGURATION",
			wantContains:     []string{"production environment", "rollback plan", "connection strings"},
		},
		{
			name:             "structural guidance",
			modificationType: "STRUCTURAL",
			wantContains:     []string{"files are affected", "circular dependency", "import paths"},
		},
		{
			name:             "behavioral guidance",
			modificationType: "BEHAVIORAL",
			wantContains:     []string{"test coverage", "edge cases", "cyclomatic complexity"},
		},
		{
			name:             "temporal pattern guidance",
			modificationType: "TEMPORAL_PATTERN",
			wantContains:     []string{"hotspot", "high churn", "incidents linked", "co-change"},
		},
		{
			name:             "ownership guidance",
			modificationType: "OWNERSHIP",
			wantContains:     []string{"new owner", "familiar", "code owner review", "knowledge transfer"},
		},
		{
			name:             "performance guidance",
			modificationType: "PERFORMANCE",
			wantContains:     []string{"load/performance tests", "bottlenecks", "caching/concurrency"},
		},
		{
			name:             "test quality guidance",
			modificationType: "TEST_QUALITY",
			wantContains:     []string{"tests being added", "critical paths", "assertions", "coverage"},
		},
		{
			name:             "unknown type fallback",
			modificationType: "UNKNOWN",
			wantContains:     []string{"Standard risk assessment", "coupling", "coverage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTypeSpecificGuidance(tt.modificationType)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("getTypeSpecificGuidance(%q) missing expected substring %q", tt.modificationType, want)
				}
			}
		})
	}
}

func TestFormatEvidenceList(t *testing.T) {
	tests := []struct {
		name          string
		evidenceChain []string
		wantContains  []string
	}{
		{
			name:          "empty evidence",
			evidenceChain: []string{},
			wantContains:  []string{"(No evidence gathered yet)"},
		},
		{
			name: "single evidence",
			evidenceChain: []string{
				"High coupling: 12 dependencies",
			},
			wantContains: []string{
				"1. High coupling: 12 dependencies",
			},
		},
		{
			name: "multiple evidence",
			evidenceChain: []string{
				"High coupling: 12 dependencies",
				"Low test coverage: 0.25",
				"Past incident: INC-123",
			},
			wantContains: []string{
				"1. High coupling: 12 dependencies",
				"2. Low test coverage: 0.25",
				"3. Past incident: INC-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEvidenceList(tt.evidenceChain)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatEvidenceList() missing expected substring %q", want)
				}
			}
		})
	}
}

// Benchmark tests

func BenchmarkConfidencePrompt(b *testing.B) {
	evidence := []string{
		"High coupling: 12 dependencies",
		"Low test coverage: 0.25",
		"Past incident: INC-123 (21 days ago)",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConfidencePrompt(evidence, "HIGH", 2)
	}
}

func BenchmarkParseConfidenceAssessment(b *testing.B) {
	response := `{
  "confidence": 0.92,
  "reasoning": "High coupling (12 deps) + low test coverage (0.25) + security file = clear HIGH risk.",
  "next_action": "FINALIZE"
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseConfidenceAssessment(response)
	}
}
