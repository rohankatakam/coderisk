# Phase: Developer Experience (DX) Foundation

**Timeline:** October 4, 2025 (completed in ~1 day with 3 parallel sessions)
**Status:** ✅ Complete
**Priority:** P0 (Critical for user adoption)

> **✅ PHASE COMPLETE:** All deliverables implemented and tested. See [DX_FOUNDATION_COMPLETE.md](../DX_FOUNDATION_COMPLETE.md) for full summary.

> **Design Reference:** [developer_experience.md](../../00-product/developer_experience.md) - Seamless risk assessment for vibe coding

---

## Goals

Implement the foundational UX features that make CodeRisk **invisible when safe, visible when risky** - ensuring zero-friction adoption and AI assistant integration.

**Core Principle:** Developers should experience automatic, fast, actionable risk assessment without changing their workflow.

---

## Scope

### In Scope (This Phase)

**P0 - Seamless Integration:**
- [ ] Pre-commit hook installation (`crisk hook install`)
- [ ] Adaptive verbosity (4 levels: quiet, standard, explain, AI mode)
- [ ] Actionable error messages with recommendations

**P1 - AI Assistant Integration:**
- [ ] AI Mode JSON output (`--ai-mode`)
- [ ] Ready-to-execute AI prompts for auto-fixing
- [ ] Confidence scoring for auto-fix safety

**P2 - Team Collaboration:**
- [ ] Team size modes (solo/team/standard/enterprise)
- [ ] Override tracking and audit logging
- [ ] Configurable blocking thresholds

### Out of Scope (Future Phases)

- VS Code extension (Phase: IDE Integration)
- AI-powered fix suggestions (requires LLM integration - Phase 2)
- Confidence scoring & pattern learning (Phase: ML Enhancement)
- Team dashboard & analytics (Phase: Observability)

---

## Deliverables

| Deliverable | Owner | Status | Estimated Days | Notes |
|-------------|-------|--------|----------------|-------|
| **Core UX** |
| Pre-commit hook (`hook install`) | TBD | Not Started | 2 days | Includes install/uninstall, error handling |
| Adaptive verbosity (quiet mode) | TBD | Not Started | 1 day | One-line summary for hooks |
| Adaptive verbosity (standard mode) | TBD | Not Started | 1 day | Issues + recommendations |
| Adaptive verbosity (explain mode) | TBD | Not Started | 2 days | Full investigation trace |
| Actionable error messages | TBD | Not Started | 2 days | "What to do" guidance |
| **AI Integration** |
| AI Mode JSON schema | TBD | Not Started | 2 days | Machine-readable output |
| AI prompt generation | TBD | Not Started | 3 days | Generate fix prompts per issue type |
| Auto-fix confidence scoring | TBD | Not Started | 2 days | Confidence thresholds for auto-fix |
| **Team Features** |
| Team mode configuration | TBD | Not Started | 1 day | Solo/team/standard/enterprise |
| Override tracking & logging | TBD | Not Started | 2 days | Audit trail for --no-verify |
| Configurable blocking rules | TBD | Not Started | 1 day | Block on LOW/MEDIUM/HIGH |

**Total Estimated Duration:** 19 days (3-4 weeks with testing/refinement)

---

## Success Criteria

### Functional Requirements

- [ ] **Pre-commit hook works end-to-end**
  - Installs with single command
  - Runs automatically on `git commit`
  - Blocks on HIGH risk by default
  - Allows override with `--no-verify`

- [ ] **Verbosity levels work correctly**
  - Quiet: One-line summary (<2s)
  - Standard: Issues + recommendations (default)
  - Explain: Full trace with hop-by-hop reasoning
  - AI Mode: Valid JSON matching schema v1.0

- [ ] **Error messages are actionable**
  - Every error includes "What to do" section
  - Suggests specific commands to fix issues
  - Links to relevant documentation

- [ ] **AI Mode enables auto-fixing**
  - JSON includes `ai_assistant_actions[]` array
  - High-confidence actions (>0.85) marked `ready_to_execute`
  - Prompts are executable by Claude Code/Cursor

### Performance Requirements

- [ ] **Pre-commit check <2s** (p50, cached)
- [ ] **Pre-commit check <5s** (p95, cold start)
- [ ] **AI Mode generation overhead <200ms** (vs standard mode)
- [ ] **JSON output <10KB** (typical 3-file commit)

### Quality Requirements

- [ ] **80%+ unit test coverage** for formatters and hook logic
- [ ] **Integration tests** for each verbosity level
- [ ] **End-to-end test** for pre-commit hook workflow
- [ ] **JSON schema validation** for AI Mode output

---

## Dependencies

### Completed Prerequisites

- ✅ Phase 1 metrics (coupling, co-change, test ratio)
- ✅ Layer 1 (Tree-sitter AST parsing)
- ✅ Layers 2-3 (GitHub API, graph construction)
- ✅ CLI foundation (`crisk check`, `crisk init`)

### External Dependencies

- None (all features can be implemented with existing infrastructure)

### Documentation Dependencies

- ✅ [ux_adaptive_verbosity.md](../integration_guides/ux_adaptive_verbosity.md) - Implementation guide
- ✅ [ux_pre_commit_hook.md](../integration_guides/ux_pre_commit_hook.md) - Hook integration
- ✅ [ux_ai_mode_output.md](../integration_guides/ux_ai_mode_output.md) - AI Mode schema

---

## Implementation Plan

### Week 1: Core UX

**Days 1-2: Pre-commit Hook**
- Implement `cmd/crisk/hook.go` (install/uninstall commands)
- Add `--pre-commit` mode to `cmd/crisk/check.go`
- Implement `internal/git/staged.go` (staged file detection)
- Create hook script template with error handling
- Test: Install hook, commit safe/risky code, verify blocking

**Days 3-5: Adaptive Verbosity (Levels 1-2)**
- Implement `internal/output/formatter.go` (interface)
- Implement `internal/output/quiet.go` (Level 1)
- Implement `internal/output/standard.go` (Level 2)
- Wire formatters into `cmd/crisk/check.go`
- Test: Each verbosity level with sample results

**Days 6-7: Adaptive Verbosity (Level 3)**
- Implement `internal/output/explain.go` (Level 3)
- Add investigation trace formatting (hop-by-hop)
- Add metric threshold display
- Test: Full trace output with real investigation

### Week 2: AI Integration

**Days 8-9: AI Mode JSON Schema**
- Implement `internal/output/ai_mode.go` (Level 4)
- Extend `internal/models/risk_result.go` with AI fields
- Implement schema serialization (10KB target)
- Test: JSON validation against schema v1.0

**Days 10-12: AI Prompt Generation**
- Create prompt templates for common fixes:
  - Generate tests (for zero coverage)
  - Add error handling (for network calls)
  - Reduce coupling (for high co-change)
  - Fix security issues (for vulnerabilities)
- Implement `internal/ai/prompt_generator.go`
- Test: Generate prompts for each issue type

**Days 13-14: Auto-fix Confidence Scoring**
- Implement confidence calculation:
  - Test generation: 0.9+ (high confidence)
  - Error handling: 0.85+ (medium confidence)
  - Refactoring: 0.6-0.8 (requires review)
- Mark `ready_to_execute` for confidence >0.85
- Test: Validate confidence thresholds

### Week 3: Team Features & Polish

**Day 15: Team Mode Configuration**
- Implement team mode detection (solo/team/standard/enterprise)
- Add `.coderisk.yml` config support
- Set blocking thresholds per mode
- Test: Each mode with appropriate behavior

**Days 16-17: Override Tracking**
- Implement `internal/audit/override.go`
- Log overrides to `.coderisk/hook_log.jsonl`
- Add `crisk hook stats` command (show override patterns)
- Test: Track overrides, view stats

**Days 18-19: Refinement & Documentation**
- Fix bugs from integration testing
- Add user-facing error messages
- Create user documentation (README section)
- Final end-to-end testing

---

## Testing Strategy

### Unit Tests (80%+ coverage target)

**Formatters:**
- `internal/output/quiet_test.go` - Test one-line summaries
- `internal/output/standard_test.go` - Test issue + recommendation formatting
- `internal/output/explain_test.go` - Test trace formatting
- `internal/output/ai_mode_test.go` - Test JSON schema generation

**Hook Logic:**
- `cmd/crisk/hook_test.go` - Test install/uninstall
- `internal/git/staged_test.go` - Test staged file detection
- `internal/audit/override_test.go` - Test override logging

**AI Features:**
- `internal/ai/prompt_generator_test.go` - Test prompt generation
- `internal/ai/confidence_test.go` - Test confidence scoring

### Integration Tests

**Pre-commit Hook:**
```bash
# test/integration/test_pre_commit.sh
1. Install hook in test repo
2. Commit safe code → verify allows
3. Commit risky code → verify blocks
4. Override with --no-verify → verify logs
5. Check audit log → verify entry
```

**Verbosity Levels:**
```bash
# test/integration/test_verbosity.sh
1. Run check --quiet → verify one-line
2. Run check (default) → verify issues + recs
3. Run check --explain → verify trace
4. Run check --ai-mode → verify JSON + validate schema
```

**AI Mode Output:**
```bash
# test/integration/test_ai_mode.sh
1. Generate AI Mode output
2. Validate JSON against schema (ajv validate)
3. Check ai_assistant_actions[] array
4. Verify confidence scores
5. Test prompt templates
```

### End-to-End Test

**Scenario: Developer commits AI-generated code**
1. AI generates risky code (no tests, high coupling)
2. Developer runs `git add . && git commit -m "Add feature"`
3. Hook triggers → CodeRisk runs in quiet mode
4. Risk detected → commit blocked
5. Developer runs `crisk check --explain` → sees full trace
6. Developer runs `crisk fix-with-ai --tests` (future: auto-fixes)
7. Re-commits → passes check → commit succeeds

---

## Risks & Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Pre-commit hook too slow (>5s)** | Medium | High | Implement aggressive caching, parallel file analysis, 5s timeout with fail-open |
| **AI Mode JSON too large (>50KB)** | Low | Medium | Truncate investigation trace, limit graph depth to 3 hops |
| **False positives block valid commits** | Medium | High | Ensure easy override (`--no-verify`), track FP rate, auto-disable metrics >3% FP |
| **AI prompt quality varies** | Medium | Medium | Start with high-confidence fixes only (tests, error handling), iterate based on feedback |
| **Team mode config too complex** | Low | Medium | Provide sensible defaults, document each mode clearly |

---

## Metrics & Success Indicators

### Adoption Metrics

- **Hook installation rate:** >80% of developers install pre-commit hook
- **Override rate:** <10% of commits use `--no-verify`
- **AI Mode usage:** >50% of Claude Code/Cursor users enable AI Mode

### Performance Metrics

- **p50 check latency:** <2s (cached)
- **p95 check latency:** <5s (cold start)
- **Cache hit rate:** >60% for repeat checks

### Quality Metrics

- **False positive rate:** <3% across all metrics
- **AI fix success rate:** >80% for high-confidence actions (>0.9)
- **User satisfaction:** NPS >40 for DX features

---

## User Education Plan

### Documentation to Create

1. **User Guide: Pre-commit Hook**
   - Installation instructions
   - Override procedures
   - Troubleshooting guide

2. **Developer Guide: Verbosity Levels**
   - When to use each level
   - Output examples
   - CLI flags reference

3. **AI Assistant Integration Guide**
   - Claude Code setup
   - Cursor integration
   - JSON schema reference

4. **Team Configuration Guide**
   - Choosing team mode
   - Configuring blocking rules
   - Override policies

### Onboarding Flow

**First-time user:**
```bash
# Step 1: Initialize CodeRisk
crisk init

# Step 2: Install pre-commit hook
crisk hook install

✅ Pre-commit hook installed!

💡 What happens next?
   • CodeRisk checks your code automatically before commits
   • HIGH risk commits are blocked
   • Override anytime with: git commit --no-verify

🎯 Try it now:
   git add <file>
   git commit -m "Test commit"

📚 Learn more: https://docs.coderisk.com/pre-commit-hook
```

---

## Timeline

```
Week 1          Week 2          Week 3
│               │               │
├─ Day 1-2: Pre-commit hook
├─ Day 3-5: Verbosity L1-L2
├─ Day 6-7: Verbosity L3
│               │
                ├─ Day 8-9: AI Mode JSON
                ├─ Day 10-12: AI Prompts
                ├─ Day 13-14: Confidence
                │               │
                                ├─ Day 15: Team modes
                                ├─ Day 16-17: Override tracking
                                ├─ Day 18-19: Refinement

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    Phase Complete (Day 19)
```

---

## Next Phase

After DX Foundation is complete:

**Phase: LLM Investigation (Phase 2)**
- Implement LLM decision loop
- Add Tier 2 metrics (ownership_churn, incident_similarity)
- Create investigation agent with hop limits
- Reference: [agentic_design.md](../../01-architecture/agentic_design.md)

---

## References

**Design Documents:**
- [developer_experience.md](../../00-product/developer_experience.md) - UX philosophy and requirements
- [spec.md](../../spec.md) - System requirements (NFR-26 to NFR-30: Usability)

**Integration Guides:**
- [ux_adaptive_verbosity.md](../integration_guides/ux_adaptive_verbosity.md) - Implementation details
- [ux_pre_commit_hook.md](../integration_guides/ux_pre_commit_hook.md) - Hook behavior
- [ux_ai_mode_output.md](../integration_guides/ux_ai_mode_output.md) - JSON schema

**Related Architecture:**
- [agentic_design.md](../../01-architecture/agentic_design.md) - Investigation flow
- [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) - Risk calculation

---

**Status:** Ready to start implementation after Layer 1 completion
**Owner:** TBD
**Last Updated:** October 4, 2025
