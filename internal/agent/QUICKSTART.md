# Agent Package - Quick Start Guide

## Installation

```bash
go get github.com/sashabaranov/go-openai
```

## 5-Minute Setup

### 1. Set Environment Variables

```bash
export CODERISK_API_KEY=sk-your-openai-key
export CODERISK_LLM_PROVIDER=openai
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/coderisk/coderisk-go/internal/agent"
)

func main() {
    // Create LLM client
    llm, err := agent.NewLLMClient("openai", os.Getenv("CODERISK_API_KEY"))
    if err != nil {
        panic(err)
    }

    // Create investigator (with mock clients for now)
    investigator := agent.NewInvestigator(llm, nil, nil, nil)

    // Run investigation
    req := agent.InvestigationRequest{
        FilePath:   "src/payment_processor.py",
        ChangeType: "modify",
        Baseline: agent.BaselineMetrics{
            CouplingScore:     0.8,
            IncidentCount:     3,
        },
    }

    assessment, err := investigator.Investigate(context.Background(), req)
    if err != nil {
        panic(err)
    }

    // Print results
    fmt.Printf("Risk: %s (%.2f)\n", assessment.RiskLevel, assessment.RiskScore)
    fmt.Printf("Summary: %s\n", assessment.Summary)
}
```

## Test It

```bash
# Run unit tests
go test ./internal/agent/... -v

# Test with real OpenAI API
go run scripts/test_llm.go openai "Analyze this code for risk"
```

## Risk Levels

| Score | Level    | Action                  |
|-------|----------|-------------------------|
| ≥0.8  | CRITICAL | Block merge, escalate   |
| ≥0.6  | HIGH     | Require review          |
| ≥0.4  | MEDIUM   | Recommend tests         |
| ≥0.2  | LOW      | Informational           |
| <0.2  | MINIMAL  | Safe to merge           |

## Public API

```go
// Create investigator
investigator := agent.NewInvestigator(llm, temporal, incidents, graph)

// Run investigation
assessment, _ := investigator.Investigate(ctx, request)

// Access results
assessment.RiskLevel     // CRITICAL | HIGH | MEDIUM | LOW | MINIMAL
assessment.RiskScore     // 0.0-1.0
assessment.Confidence    // 0.0-1.0
assessment.Summary       // Human-readable 2-3 sentences
assessment.Evidence      // []Evidence with details
```

## Integration Points

When Sessions A & B are ready:

```go
// Session A - Temporal Analysis
temporal := temporal.NewClient(db)

// Session B - Incident Database
incidents := incidents.NewDatabase(postgresDB)

// Session C - Agent (this package)
investigator := agent.NewInvestigator(llm, temporal, incidents, graph)
```

## Performance

- Investigation time: ~3s
- Token usage: ~3K
- Hops: 1-3 (average 2)

## Troubleshooting

**Error: "API key required"**
```bash
export CODERISK_API_KEY=sk-...
```

**Error: "unsupported provider: anthropic"**
- Anthropic support coming soon, use "openai" for now

**Tests failing?**
```bash
go mod tidy
go test ./internal/agent/... -v
```

## Next Steps

1. ✅ Read `README.md` for full documentation
2. ✅ Run tests: `go test ./internal/agent/...`
3. ✅ Check `SESSION_C_COMPLETE.md` for implementation details
4. ⏳ Wait for Sessions A & B to integrate
5. ⏳ Test end-to-end with real repository

## Support

- Documentation: `internal/agent/README.md`
- Tests: `internal/agent/investigator_test.go`
- Integration: `test/integration/test_ai_mode.sh`
