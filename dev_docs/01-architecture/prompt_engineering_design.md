# Prompt Engineering Architecture

**Version:** 1.0
**Last Updated:** October 2, 2025
**Purpose:** LLM prompt design and context management for agentic risk investigation
**Design Philosophy:** Own your prompts and context window for reliable, cost-effective agent behavior *(12-factor: Factors 2 & 3)*

---

## 1. Core Principles

### 1.1. Design Philosophy *(12-factor: Factor 2 - Own your prompts)*

**Own Your Prompts:**
- ✅ **Version-controlled prompts** - Stored in codebase, not hardcoded
- ✅ **Structured outputs** - JSON schemas for predictable LLM responses
- ✅ **Explicit instructions** - Clear decision trees, not vague "assess risk"
- ✅ **Testable prompts** - Can validate with BAML tests before deployment

**Own Your Context Window:**
- ✅ **Token budget management** - Track and limit context size (<4K tokens per call)
- ✅ **Evidence prioritization** - High-signal evidence first, details later
- ✅ **Hierarchical summaries** - Compress graph data for LLM consumption
- ✅ **Hop-by-hop context** - Add evidence incrementally, not all at once

**Avoid:**
- ❌ **Conversational prompts** - "Can you assess the risk?" (vague, unpredictable)
- ❌ **Unstructured outputs** - Free-form text (hard to parse, inconsistent)
- ❌ **Context stuffing** - Dumping entire graph into prompt (expensive, slow)

### 1.2. Three Prompt Types

**1. Decision Prompts (Hop 1-3)**
- **Purpose:** LLM decides next investigation action
- **Output:** JSON with action (CALCULATE_METRIC | EXPAND_GRAPH | FINALIZE)
- **Token Budget:** <2K tokens
- **Frequency:** 1-3 times per Phase 2 investigation

**2. Synthesis Prompts (Final)**
- **Purpose:** LLM combines all evidence into risk assessment
- **Output:** JSON with risk_level, confidence, evidence, recommendations
- **Token Budget:** <3K tokens
- **Frequency:** Once per Phase 2 investigation

**3. Feedback Prompts (Learning)**
- **Purpose:** LLM learns from user feedback to adjust future prompts
- **Output:** JSON with suggested threshold adjustments
- **Token Budget:** <1K tokens
- **Frequency:** Batch processing weekly

---

## 2. Decision Prompt Design (Phase 2 Investigation)

### 2.1. Prompt Template (Hop N)

**Purpose:** Guide LLM to decide next investigation step

**Template:**
```
You are a code risk investigator analyzing a proposed change for potential impact.
Your goal is to gather evidence to assess risk, not to make a final decision yet.

## Current Investigation State

**Changed File:** {file_path}
**Language:** {language}
**Lines of Code:** {loc}

**Tier 1 Metrics (always calculated):**
- Structural coupling: {coupling_count} files depend on this code
  - Signal: {coupling_signal} (threshold: >10 for HIGH)
  - Evidence: "{coupling_evidence}"
- Temporal co-change: {co_change_frequency} max frequency with {co_change_file}
  - Signal: {co_change_signal} (threshold: >0.7 for HIGH)
  - Evidence: "{co_change_evidence}"
- Test coverage ratio: {test_ratio}
  - Signal: {test_ratio_signal} (threshold: <0.3 for HIGH)
  - Evidence: "{test_ratio_evidence}"

**Evidence Gathered So Far (Hop {current_hop}/3):**
{evidence_chain}

**Available Actions:**
1. CALCULATE_METRIC - Request a Tier 2 metric (ownership_churn or incident_similarity)
2. EXPAND_GRAPH - Load 2-hop neighbors for broader context (expensive, use sparingly)
3. FINALIZE - You have enough evidence to make an assessment

## Decision Rules

- If Tier 1 signals are all LOW and no concerning patterns → FINALIZE
- If ownership or incident data would clarify risk → CALCULATE_METRIC
- If coupling is very high (>20) and you need dependency context → EXPAND_GRAPH
- If you have 3+ pieces of HIGH evidence → FINALIZE
- Max 3 hops total - if current_hop = 3, you MUST choose FINALIZE

## Output Format (JSON)

Respond ONLY with valid JSON in this exact format:
{{
  "action": "CALCULATE_METRIC" | "EXPAND_GRAPH" | "FINALIZE",
  "reasoning": "Brief explanation of why you chose this action (1-2 sentences)",
  "target": {{
    "metric_name": "ownership_churn" | "incident_similarity" | null,
    "file_path": "{file_path}" | null
  }}
}}

**Example 1 - Request ownership data:**
{{
  "action": "CALCULATE_METRIC",
  "reasoning": "High coupling (12 files) combined with low test ratio (0.3) suggests checking if code owner is experienced",
  "target": {{
    "metric_name": "ownership_churn",
    "file_path": "{file_path}"
  }}
}}

**Example 2 - Finalize early:**
{{
  "action": "FINALIZE",
  "reasoning": "All Tier 1 metrics are LOW (coupling=3, co_change=0.2, test_ratio=0.8), no further investigation needed",
  "target": null
}}
```

**Token Budget Breakdown:**
- Static template: ~600 tokens
- File metadata: ~50 tokens
- Tier 1 metrics (3): ~300 tokens
- Evidence chain: ~500-1000 tokens (grows per hop)
- **Total: ~1500-2000 tokens per call**

### 2.2. Evidence Chain Formatting

**Purpose:** Present accumulated evidence in structured, scannable format

**Format:**
```
**Hop 1:**
- Action: CALCULATE_METRIC (ownership_churn)
- Result: Primary owner changed from bob@example.com to alice@example.com 14 days ago
- Signal: MEDIUM ownership stability
- Reasoning: "High coupling suggests checking ownership stability"

**Hop 2:**
- Action: CALCULATE_METRIC (incident_similarity)
- Result: Similar to Incident #123 "Auth timeout after permission check" (BM25 score: 12.3)
- Signal: HIGH incident similarity
- Reasoning: "Recent ownership change + past incident suggests elevated risk"
```

**Token Optimization:**
- Omit low-signal evidence (e.g., LOW co-change results)
- Summarize graph structure (e.g., "12 dependencies" not "file1.py, file2.py, ...")
- Use abbreviations (e.g., "MEDIUM" not "MEDIUM_RISK_SIGNAL")

### 2.3. Structured Output Schema (JSON)

**Purpose:** Ensure consistent, parseable LLM responses

**Schema:**
```json
{
  "type": "object",
  "required": ["action", "reasoning"],
  "properties": {
    "action": {
      "type": "string",
      "enum": ["CALCULATE_METRIC", "EXPAND_GRAPH", "FINALIZE"]
    },
    "reasoning": {
      "type": "string",
      "minLength": 10,
      "maxLength": 200
    },
    "target": {
      "type": "object",
      "properties": {
        "metric_name": {
          "type": "string",
          "enum": ["ownership_churn", "incident_similarity", null]
        },
        "file_path": {
          "type": "string",
          "pattern": "^[a-zA-Z0-9/_.-]+$"
        }
      }
    }
  }
}
```

**Validation:**
- Parse JSON response
- Validate against schema
- Reject invalid responses (retry once, then fallback to FINALIZE)

**Error Handling:**
```python
try:
    response = llm.call(decision_prompt)
    decision = json.loads(response)
    validate_schema(decision, DECISION_SCHEMA)
except (JSONDecodeError, ValidationError) as e:
    logger.warn(f"Invalid LLM response: {e}, retrying...")
    response = llm.call(decision_prompt + "\n\nIMPORTANT: Respond with valid JSON only.")
    decision = json.loads(response)
    if not validate_schema(decision, DECISION_SCHEMA):
        logger.error("LLM failed twice, defaulting to FINALIZE")
        decision = {"action": "FINALIZE", "reasoning": "Error in LLM response"}
```

---

## 3. Synthesis Prompt Design (Final Assessment)

### 3.1. Prompt Template

**Purpose:** Combine all evidence into final risk assessment with recommendations

**Template:**
```
You are a code risk assessor synthesizing investigation results into a final risk determination.

## Investigation Summary

**Changed File:** {file_path}
**Language:** {language}
**Lines of Code:** {loc}

## Evidence Collected

### Tier 1 Metrics (always calculated)
{tier1_summary}

### Tier 2 Metrics (calculated during investigation)
{tier2_summary}

### Investigation Trace
{investigation_hops}

## Your Task

Synthesize this evidence into a final risk assessment. Consider:
- **Coupling + Co-change:** High values suggest wide impact
- **Test ratio:** Low values suggest inadequate validation
- **Ownership churn:** Recent transitions increase risk
- **Incident similarity:** Past failures indicate vulnerability

## Output Format (JSON)

Respond ONLY with valid JSON:
{{
  "risk_level": "LOW" | "MEDIUM" | "HIGH",
  "confidence": 0.0-1.0,  // 0.8+ = strong evidence, <0.5 = weak/conflicting
  "key_evidence": [
    "First most important piece of evidence",
    "Second most important piece",
    "Third most important piece"
  ],
  "recommendations": [
    "Actionable suggestion for developer (if risk > LOW)",
    "Second suggestion (optional)",
    "Third suggestion (optional)"
  ],
  "reasoning": "1-2 sentence summary of how you combined the evidence"
}}

**Guidelines for risk_level:**
- LOW: All signals are LOW or MEDIUM, no concerning patterns
- MEDIUM: Mixed signals (some HIGH, some LOW), or single HIGH signal with mitigating factors
- HIGH: Multiple HIGH signals, or single HIGH signal with reinforcing evidence (e.g., incident history)

**Guidelines for confidence:**
- 0.9+: 3+ independent HIGH signals from different sources
- 0.7-0.9: 2 HIGH signals or 1 HIGH + supporting evidence
- 0.5-0.7: Mixed signals, some conflicting evidence
- <0.5: Insufficient data or contradictory signals

**Guidelines for recommendations:**
- Be specific (not "add tests", but "add integration tests for X scenario")
- Prioritize (most impactful first)
- Make actionable (developer can act immediately)
- Limit to 3 recommendations max

**Example Output:**
{{
  "risk_level": "HIGH",
  "confidence": 0.85,
  "key_evidence": [
    "12 files directly depend on this code (high coupling)",
    "Code owner changed 14 days ago (ownership instability)",
    "Similar to Incident #123: 'Auth timeout after permission check'"
  ],
  "recommendations": [
    "Add integration tests for permission check timeout scenarios",
    "Ensure new owner (alice@) reviews incident #123 post-mortem",
    "Consider adding circuit breaker to prevent timeout cascades"
  ],
  "reasoning": "High coupling + ownership churn + incident history indicate elevated risk of similar failure"
}}
```

**Token Budget Breakdown:**
- Static template: ~800 tokens
- Tier 1 summary: ~300 tokens
- Tier 2 summary: ~400 tokens (if calculated)
- Investigation hops: ~500-800 tokens
- **Total: ~2000-2500 tokens per call**

### 3.2. Tier 1 Summary Format

**Purpose:** Concise presentation of baseline metrics

**Format:**
```
**Structural Coupling:**
- Value: 12 files
- Signal: HIGH (threshold: >10)
- Evidence: "File is imported by auth.py, permissions.py, roles.py, and 9 others"

**Temporal Co-Change:**
- Value: 0.75 max frequency with auth.py
- Signal: HIGH (threshold: >0.7)
- Evidence: "auth.py and permissions.py changed together in 15 of last 20 commits"

**Test Coverage Ratio:**
- Value: 0.3
- Signal: MEDIUM (threshold: <0.3 for HIGH)
- Evidence: "auth.py (250 LOC) has auth_test.py (75 LOC)"
```

### 3.3. Tier 2 Summary Format

**Purpose:** Present LLM-requested metrics with context

**Format (if ownership_churn calculated):**
```
**Ownership Churn:**
- Current owner: alice@example.com (14 days)
- Previous owner: bob@example.com (45 commits over 2 years)
- Signal: MEDIUM ownership stability
- Why calculated: "High coupling suggested checking owner experience"
```

**Format (if incident_similarity calculated):**
```
**Incident Similarity:**
- Top match: Incident #123 "Auth timeout after permission check" (score: 12.3)
- Incident date: 2025-08-15 (47 days ago)
- Signal: HIGH incident similarity
- Why calculated: "Ownership churn + high coupling suggested checking incident history"
```

### 3.4. Structured Output Schema

**Schema:**
```json
{
  "type": "object",
  "required": ["risk_level", "confidence", "key_evidence", "recommendations", "reasoning"],
  "properties": {
    "risk_level": {
      "type": "string",
      "enum": ["LOW", "MEDIUM", "HIGH"]
    },
    "confidence": {
      "type": "number",
      "minimum": 0.0,
      "maximum": 1.0
    },
    "key_evidence": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 1,
      "maxItems": 5
    },
    "recommendations": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 0,
      "maxItems": 3
    },
    "reasoning": {
      "type": "string",
      "minLength": 20,
      "maxLength": 300
    }
  }
}
```

---

## 4. Context Window Management

### 4.1. Token Budget Strategy

**Per-Call Limits:**
| Prompt Type | Input Tokens | Output Tokens | Total Budget |
|-------------|--------------|---------------|--------------|
| Decision (Hop 1) | ~1500 | ~150 | ~1650 |
| Decision (Hop 2) | ~1800 | ~150 | ~1950 |
| Decision (Hop 3) | ~2000 | ~150 | ~2150 |
| Synthesis | ~2500 | ~500 | ~3000 |
| **Total per HIGH check** | **~7800** | **~950** | **~8750** |

**Cost Calculation (GPT-4o-mini):**
- Input: $0.15 per 1M tokens
- Output: $0.60 per 1M tokens
- **Total per HIGH check:** (7800 × $0.15 + 950 × $0.60) / 1M = **~$0.0018 per check**

**Daily Cost (100 checks, 20% HIGH):**
- 20 HIGH checks × $0.0018 = **$0.036/day**
- Well within budget (<$0.05 per HIGH check target)

### 4.2. Evidence Prioritization

**Include (High Signal):**
- ✅ Tier 1 metric results (always)
- ✅ HIGH signal Tier 2 metrics (if calculated)
- ✅ Incident matches with score >10
- ✅ Ownership transitions within 30 days

**Summarize (Medium Signal):**
- 📝 MEDIUM signal metrics (brief summary, not full details)
- 📝 Co-change relationships <0.5 frequency (aggregate count)
- 📝 Incident matches with score 5-10 (top 3 only)

**Omit (Low Signal):**
- ❌ LOW signal metrics (don't add noise)
- ❌ Graph structure beyond 1-hop (unless EXPAND_GRAPH requested)
- ❌ Full file content (only metadata: path, language, LOC)
- ❌ Verbose git commit messages (use first line only)

**Example (Omit LOW co-change):**
```
# Don't include:
"Co-change with utils.py: 0.12 frequency (LOW)"
"Co-change with config.py: 0.08 frequency (LOW)"

# Instead:
"5 additional files with co-change frequency <0.3 (LOW signal)"
```

### 4.3. Hierarchical Summarization

**Level 1: File-Level Summary (Always Include)**
```
Changed file: src/auth/permissions.py (Python, 250 LOC)
```

**Level 2: Metric Summaries (Always Include)**
```
- Coupling: 12 files (HIGH)
- Co-change: 0.75 with auth.py (HIGH)
- Test ratio: 0.3 (MEDIUM)
```

**Level 3: Detailed Evidence (Include for HIGH signals only)**
```
Coupling details:
- Imported by: auth.py, roles.py, session.py, and 9 others
- Called by: check_permissions() (12 call sites)
```

**Level 4: Graph Context (Include only if EXPAND_GRAPH)**
```
2-hop neighbors: 47 files
- 12 direct dependencies
- 35 transitive dependencies (avg distance: 1.7 hops)
```

---

## 5. Prompt Versioning and Testing

### 5.1. Version Control Strategy

**File Structure:**
```
internal/prompts/
  v1/
    decision_hop.txt       # Decision prompt template
    synthesis.txt          # Synthesis prompt template
    feedback.txt           # Feedback prompt template
  v2/
    decision_hop.txt       # Improved version
    synthesis.txt
  current -> v1/           # Symlink to active version
```

**Prompt Metadata:**
```
# decision_hop.txt
# Version: 1.0
# Last Updated: 2025-10-02
# FP Rate: 8.2% (n=127)
# Success Rate: 94.3% (valid JSON responses)
# Avg Tokens: 1847 input, 142 output
```

**Version Upgrade Workflow:**
1. Create new version in `v2/` directory
2. A/B test with 10% of requests
3. Monitor FP rate, JSON parse success, token usage
4. If `v2 FP rate < v1 FP rate - 2%`, promote to current
5. Archive old version, update symlink

### 5.2. BAML Testing (12-factor: Factor 2)

**Test Cases:**
```python
# tests/prompts/test_decision_hop.py

def test_high_coupling_triggers_ownership_check():
    """High coupling should request ownership_churn metric"""
    context = InvestigationContext(
        file_path="auth.py",
        tier1_metrics={
            "coupling": {"value": 15, "signal": "HIGH"},
            "co_change": {"value": 0.4, "signal": "MEDIUM"},
            "test_ratio": {"value": 0.5, "signal": "MEDIUM"}
        },
        evidence_chain=[],
        hop_count=1
    )

    decision = call_decision_prompt(context)

    assert decision["action"] == "CALCULATE_METRIC"
    assert decision["target"]["metric_name"] == "ownership_churn"

def test_all_low_signals_finalize_early():
    """All LOW signals should finalize without further investigation"""
    context = InvestigationContext(
        file_path="utils.py",
        tier1_metrics={
            "coupling": {"value": 3, "signal": "LOW"},
            "co_change": {"value": 0.15, "signal": "LOW"},
            "test_ratio": {"value": 0.9, "signal": "LOW"}
        },
        evidence_chain=[],
        hop_count=1
    )

    decision = call_decision_prompt(context)

    assert decision["action"] == "FINALIZE"

def test_hop_3_must_finalize():
    """Hop 3 should always finalize (budget limit)"""
    context = InvestigationContext(
        file_path="complex.py",
        tier1_metrics={
            "coupling": {"value": 20, "signal": "HIGH"},
            "co_change": {"value": 0.8, "signal": "HIGH"},
            "test_ratio": {"value": 0.1, "signal": "HIGH"}
        },
        evidence_chain=[...],  # Previous hops
        hop_count=3  # Max hops reached
    )

    decision = call_decision_prompt(context)

    assert decision["action"] == "FINALIZE"
```

**Test Metrics:**
- **JSON Parse Success:** >95% (detect prompt drift)
- **Schema Validation:** 100% (all fields present)
- **Decision Consistency:** >90% (same input → same output)

### 5.3. Feedback-Driven Iteration

**User Feedback → Prompt Improvement Loop:**

**Step 1: Collect Feedback**
```sql
SELECT
    metric_name,
    feedback_reason,
    COUNT(*) as frequency
FROM metric_validations
WHERE user_feedback = 'false_positive'
GROUP BY metric_name, feedback_reason
ORDER BY frequency DESC
```

**Step 2: Identify Patterns**
```
Top false positive reasons for coupling:
1. "Intentional coupling in framework code" (40%)
2. "Test files inflating count" (30%)
3. "Generated code (migrations, protobufs)" (20%)
```

**Step 3: Update Prompt**
```diff
# decision_hop.txt v1.1

+ **Special Cases to Consider:**
+ - Framework code (e.g., Django models, React components) often has high intentional coupling
+ - Test files should not count toward coupling (check if file path contains 'test', '__tests__', 'spec')
+ - Generated code (migrations, protobufs) may have high coupling by design

IF coupling is HIGH AND file matches framework pattern:
  → Request ownership_churn to verify experienced developer
ELSE IF coupling is HIGH AND file is generated:
  → FINALIZE with MEDIUM risk (coupling is expected)
```

**Step 4: A/B Test**
- Deploy v1.1 to 10% of requests
- Compare FP rate: v1.0 (8.2%) vs v1.1 (target: <6%)
- If successful, promote to 100%

---

## 6. Integration with Agent Design

### 6.1. Prompt Flow (Phase 2)

**Step 1: Initialize Context**
```python
context = InvestigationContext(
    file_path=changed_file,
    tier1_metrics=baseline_results,  # From Phase 1
    evidence_chain=[],
    hop_count=0
)
```

**Step 2: Decision Loop (Max 3 Hops)**
```python
for hop in range(1, 4):  # Hops 1, 2, 3
    context.hop_count = hop

    # Format decision prompt
    prompt = format_decision_prompt(context)

    # Call LLM
    response = llm.call(prompt, max_tokens=200)
    decision = parse_and_validate_json(response)

    # Execute action
    if decision["action"] == "CALCULATE_METRIC":
        metric_result = calculate_tier2_metric(
            metric_name=decision["target"]["metric_name"],
            file_path=decision["target"]["file_path"]
        )
        context.evidence_chain.append({
            "hop": hop,
            "action": "CALCULATE_METRIC",
            "metric": decision["target"]["metric_name"],
            "result": metric_result,
            "reasoning": decision["reasoning"]
        })

    elif decision["action"] == "EXPAND_GRAPH":
        neighbors = load_2_hop_neighbors(context.file_path)
        context.working_memory.update(neighbors)
        context.evidence_chain.append({
            "hop": hop,
            "action": "EXPAND_GRAPH",
            "nodes_loaded": len(neighbors),
            "reasoning": decision["reasoning"]
        })

    elif decision["action"] == "FINALIZE":
        break  # Exit loop, proceed to synthesis
```

**Step 3: Synthesize Assessment**
```python
# Format synthesis prompt
synthesis_prompt = format_synthesis_prompt(context)

# Call LLM
response = llm.call(synthesis_prompt, max_tokens=500)
assessment = parse_and_validate_json(response)

# Return final result
return RiskAssessment(
    risk_level=assessment["risk_level"],
    confidence=assessment["confidence"],
    key_evidence=assessment["key_evidence"],
    recommendations=assessment["recommendations"],
    reasoning=assessment["reasoning"],
    investigation_trace=context.evidence_chain
)
```

### 6.2. Prompt + Metric Integration

**Metric Results → Prompt Format:**
```python
def format_tier1_summary(metrics: Dict[str, MetricResult]) -> str:
    """Convert metric results to LLM-friendly format"""
    summary = []

    for name, metric in metrics.items():
        summary.append(f"- {name.title()}: {metric.value} ({metric.signal})")
        summary.append(f"  Evidence: \"{metric.evidence}\"")
        summary.append(f"  FP rate: {metric.fp_rate:.1%}")

    return "\n".join(summary)

# Example output:
"""
- Coupling: 12 files (HIGH)
  Evidence: "File is imported by 12 other files"
  FP rate: 2.1%
- Co-change: 0.75 (HIGH)
  Evidence: "Changed with auth.py in 15 of 20 commits"
  FP rate: 4.3%
- Test Ratio: 0.3 (MEDIUM)
  Evidence: "auth.py (250 LOC) has auth_test.py (75 LOC)"
  FP rate: 7.2%
"""
```

**Evidence Chain → Prompt Format:**
```python
def format_evidence_chain(chain: List[EvidenceItem]) -> str:
    """Format investigation history for LLM context"""
    if not chain:
        return "No evidence gathered yet."

    formatted = []
    for item in chain:
        formatted.append(f"**Hop {item.hop}:**")
        formatted.append(f"- Action: {item.action}")
        formatted.append(f"- Result: {item.result}")
        formatted.append(f"- Reasoning: \"{item.reasoning}\"")
        formatted.append("")

    return "\n".join(formatted)
```

---

## 7. Error Handling and Fallbacks

### 7.1. LLM Response Errors

**JSON Parse Failures:**
```python
def parse_llm_response(response: str, retry_count: int = 1) -> dict:
    """Parse LLM response with retry logic"""
    try:
        return json.loads(response)
    except JSONDecodeError as e:
        logger.warn(f"JSON parse error: {e}")

        if retry_count > 0:
            # Retry with explicit JSON instruction
            retry_prompt = original_prompt + "\n\nIMPORTANT: Respond with valid JSON only, no markdown formatting."
            response = llm.call(retry_prompt)
            return parse_llm_response(response, retry_count - 1)
        else:
            # Fallback to safe default
            logger.error("LLM failed to produce valid JSON after retry")
            return {"action": "FINALIZE", "reasoning": "Error in LLM response"}
```

**Schema Validation Failures:**
```python
def validate_decision(decision: dict) -> bool:
    """Validate decision against schema"""
    required_fields = ["action", "reasoning"]

    if not all(field in decision for field in required_fields):
        logger.error(f"Missing required fields: {decision}")
        return False

    if decision["action"] not in ["CALCULATE_METRIC", "EXPAND_GRAPH", "FINALIZE"]:
        logger.error(f"Invalid action: {decision['action']}")
        return False

    if decision["action"] == "CALCULATE_METRIC":
        if "target" not in decision or "metric_name" not in decision["target"]:
            logger.error("CALCULATE_METRIC requires target.metric_name")
            return False

    return True
```

### 7.2. Timeout Handling

**Per-Hop Timeout:**
```python
def call_llm_with_timeout(prompt: str, timeout_seconds: int = 10) -> str:
    """Call LLM with timeout protection"""
    try:
        response = asyncio.wait_for(
            llm.call_async(prompt),
            timeout=timeout_seconds
        )
        return response
    except asyncio.TimeoutError:
        logger.error(f"LLM call timed out after {timeout_seconds}s")
        return json.dumps({
            "action": "FINALIZE",
            "reasoning": "Investigation timed out, finalizing with current evidence"
        })
```

**Total Investigation Timeout:**
```python
MAX_PHASE2_DURATION = 10  # seconds

start_time = time.time()
for hop in range(1, 4):
    elapsed = time.time() - start_time
    if elapsed > MAX_PHASE2_DURATION:
        logger.warn(f"Phase 2 timeout ({elapsed:.1f}s), finalizing early")
        break

    # Call decision prompt with remaining time budget
    remaining_time = MAX_PHASE2_DURATION - elapsed
    decision = call_llm_with_timeout(prompt, timeout_seconds=remaining_time)
    ...
```

### 7.3. Degraded Mode (No LLM Available)

**Fallback to Heuristic-Only:**
```python
def assess_risk_degraded_mode(tier1_metrics: Dict) -> RiskAssessment:
    """Assess risk without LLM (heuristic only)"""
    high_signals = [m for m in tier1_metrics.values() if m.signal == "HIGH"]

    if len(high_signals) >= 2:
        risk_level = "HIGH"
        key_evidence = [m.evidence for m in high_signals]
        recommendations = [
            "Review dependencies and test coverage",
            "Consider breaking down this change into smaller parts"
        ]
    elif len(high_signals) == 1:
        risk_level = "MEDIUM"
        key_evidence = [high_signals[0].evidence]
        recommendations = ["Add tests for changed code"]
    else:
        risk_level = "LOW"
        key_evidence = []
        recommendations = []

    return RiskAssessment(
        risk_level=risk_level,
        confidence=0.0,  # No confidence without LLM
        key_evidence=key_evidence,
        recommendations=recommendations,
        reasoning="LLM unavailable, using heuristic fallback",
        mode="DEGRADED"
    )
```

---

## 8. Performance Optimization

### 8.1. Prompt Caching (Future)

**Cache Decision Prompts (Prompt Caching API):**
```python
# Cache static parts of prompt
cached_template = llm.cache_prompt(
    template=DECISION_PROMPT_TEMPLATE,
    ttl=3600  # 1 hour
)

# Inject dynamic values
full_prompt = cached_template.format(
    file_path=context.file_path,
    tier1_metrics=format_tier1_summary(context.tier1_metrics),
    evidence_chain=format_evidence_chain(context.evidence_chain)
)

# Call LLM with cached prefix (50% token cost reduction)
response = llm.call(full_prompt, use_cache=True)
```

**Savings:**
- Cached tokens (static): ~600 tokens
- Cost reduction: 50% on input tokens
- **New cost per call:** ~$0.0012 (was $0.0018)

### 8.2. Batch Processing (Non-Interactive)

**For CI/CD Pre-Commit Hooks (batch 10+ files):**
```python
async def batch_assess_risk(changed_files: List[str]) -> List[RiskAssessment]:
    """Assess multiple files in parallel"""
    tasks = []
    for file_path in changed_files:
        task = assess_risk_async(file_path)
        tasks.append(task)

    # Run up to 5 concurrent investigations
    results = await asyncio.gather(*tasks, max_concurrency=5)
    return results
```

**Performance:**
- Sequential: 10 files × 3.5s = 35s total
- Parallel (5 concurrent): 10 files / 5 × 3.5s = 7s total
- **Speedup: 5x**

---

## 9. Key Design Decisions (ADR Summary)

### 9.1. Why Structured JSON Outputs Over Free Text?

**Decision:** Require JSON schema for all LLM responses

**Rationale:**
- **Parseable:** Can extract risk_level, confidence programmatically
- **Testable:** Can validate against schema in tests
- **Consistent:** Avoid "Risk Level: HIGH" vs "HIGH RISK" vs "High" variations

**Trade-off:**
- Slightly more verbose prompts (+100 tokens for schema examples)
- Acceptable for reliability gain

### 9.2. Why Max 3 Hops?

**Decision:** Hard limit of 3 LLM decision loops per investigation

**Rationale:**
- **Cost control:** Unbounded loops could cost $0.50+ per check
- **Latency control:** More hops = slower response (user waits)
- **Diminishing returns:** Most HIGH risk is clear by hop 2

**Trade-off:**
- May miss nuanced patterns requiring 4+ hops
- Acceptable for V1, revisit if FP rate crosses threshold

### 9.3. Why Decision + Synthesis Prompts (Not Single Prompt)?

**Decision:** Separate prompts for investigation and final assessment

**Rationale:**
- **Focused instructions:** Decision prompts guide exploration, synthesis prompts guide conclusion
- **Token efficiency:** Don't repeat investigation logic in synthesis prompt
- **Flexibility:** Can improve one prompt without affecting the other

**Trade-off:**
- More LLM calls (4 total: 3 decision + 1 synthesis)
- Acceptable given cost (<$0.002 per HIGH check)

---

## 10. References

**Architecture:**
- [agentic_design.md](agentic_design.md) - Two-phase investigation flow
- [risk_assessment_methodology.md](risk_assessment_methodology.md) - Metric definitions and thresholds
- [graph_ontology.md](graph_ontology.md) - Graph data sources for evidence

**12-Factor Principles:**
- [Factor 2: Own Your Prompts](../12-factor-agents-main/content/factor-02-own-your-prompts.md) - Prompt versioning, testing, and iteration
- [Factor 3: Own Your Context Window](../12-factor-agents-main/content/factor-03-own-your-context-window.md) - Token budget management, evidence prioritization
- [Factor 4: Tools Are Structured Outputs](../12-factor-agents-main/content/factor-04-tools-are-structured-outputs.md) - JSON schemas for LLM responses

---

**Last Updated:** October 2, 2025
**Next Review:** After V1 MVP deployment, analyze LLM success rates and iterate prompts
