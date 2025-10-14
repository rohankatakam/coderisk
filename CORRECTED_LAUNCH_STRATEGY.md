# CORRECTED: CodeRisk Launch Strategy

**Created:** 2025-10-13
**Status:** CORRECTED - API Key is Essential
**Critical Correction:** Phase 0 is a pre-filter, NOT a standalone mode

---

## ⚠️ IMPORTANT CORRECTION

**Previous mistake:** Docs suggested Phase 0 works as a standalone static analysis tool without API key.

**Reality:** Phase 0 is a **pre-filter** (<50ms) that runs BEFORE the LLM agentic investigation. The LLM (OpenAI API key) is **essential** for CodeRisk's core value proposition.

---

## Correct Architecture

### Phase 0: Pre-Filter (No LLM, <50ms)

**Purpose:** Fast heuristics to optimize investigation, NOT replace it

**What it does:**
1. **Security keyword detection** → Force escalate to LLM
2. **Documentation-only detection** → Skip expensive LLM call (save $)
3. **Environment config detection** → Force escalate to LLM
4. **Domain-aware config selection** → Pass to LLM for adaptive thresholds

**Does NOT provide risk assessment on its own!**

```
Phase 0 output:
- force_escalate: true/false
- skip_analysis: true/false
- modification_type: "SECURITY" | "DOCS" | etc.
- config: { coupling_threshold: 15, ... }

→ Passes to Phase 1 (LLM baseline) + Phase 2 (LLM deep investigation)
```

### Phase 1: Baseline Assessment (LLM Required, ~1-2s)

**Purpose:** Quick LLM-guided metric calculation

**Requires:** OpenAI API key

**What it does:**
1. LLM analyzes Phase 0 results + git diff
2. Requests Tier 1 metrics (coupling, co-change, test ratio)
3. Makes initial risk assessment
4. Decides if Phase 2 investigation needed

### Phase 2: Deep Investigation (LLM Required, ~2-5s)

**Purpose:** Confidence-driven graph traversal

**Requires:** OpenAI API key + Graph database (Neptune/Neo4j)

**What it does:**
1. LLM navigates graph (max 3-5 hops)
2. Requests Tier 2 metrics on-demand
3. Synthesizes evidence
4. Stops when confidence >85% or max hops reached

---

## Correct Value Proposition

### Without API Key: NOTHING WORKS ❌

Phase 0 alone does NOT provide risk assessment. It's just a pre-filter.

```bash
crisk check
# ❌ Error: OPENAI_API_KEY not set
# CodeRisk requires OpenAI API for risk assessment
# Get key: https://platform.openai.com/api-keys
```

### With API Key: FULL VALUE ✅

```bash
export OPENAI_API_KEY="sk-..."
crisk check
# ✅ Phase 0: Pre-filter (0.05s) - Detected security keywords
# ✅ Phase 1: Baseline assessment (1.2s) - HIGH coupling detected
# ✅ Phase 2: Deep investigation (2.3s) - Confidence: 87%
#
# Risk: HIGH
# Reason: Auth file with 12 dependencies + security keywords + low test coverage
```

---

## Why API Key is Essential

### Core Value Props REQUIRE LLM:

1. **<3% False Positive Rate**
   - Achieved via LLM-guided selective metric calculation
   - Static analysis: 10-20% FP rate (SonarQube level)
   - Phase 0 alone: No assessment, just classification

2. **Agentic Graph Search**
   - LLM decides what to calculate and when to stop
   - Navigates 10,000-file graph intelligently
   - Phase 0 alone: No graph navigation

3. **Evidence-Based Risk Assessment**
   - LLM synthesizes multiple low-FP metrics
   - Explainable reasoning traces
   - Phase 0 alone: Just keywords, no synthesis

4. **Adaptive Investigation**
   - Confidence-driven stopping (85% threshold)
   - Domain-aware thresholds (Python web vs Go backend)
   - Phase 0 alone: Just selects config, doesn't use it

### From Your Test Output:

```
✅ Phase 0: Static analysis (0.2s)
   - Detected modification_types=[Security]
   - Set force_escalate=true

⚠️  Phase 2 unavailable: OPENAI_API_KEY not set
```

**What this means:**
- Phase 0 detected keywords → flagged for investigation
- But NO RISK ASSESSMENT happened
- Needs LLM (Phase 1/2) to actually assess risk

**Your test worked because:**
- Your environment had `OPENAI_API_KEY` set
- Phases 1 & 2 ran successfully
- Full agentic investigation completed

---

## Corrected Positioning

### What to Emphasize:

**Primary Message:**
> "CodeRisk uses AI-powered agentic graph search to achieve <3% false positive rates. Requires OpenAI API key."

**Secondary Message:**
> "Phase 0 pre-filter optimizes performance (skips docs-only changes, escalates security), but LLM investigation is the core value."

**Cost Transparency:**
> "You control LLM costs (BYOK model). Typical: $0.03-0.05/check. We cover infrastructure ($2-3/user/month)."

### What NOT to Say:

❌ "Works without API key"
❌ "Phase 0 static analysis provides risk assessment"
❌ "Zero configuration needed"
❌ "Free tier without API key"

### What to Say:

✅ "Quick setup: Add OpenAI API key (30 seconds)"
✅ "Optional: Local graph for full analysis (Docker, 10 min)"
✅ "You control LLM spend ($0.03-0.05/check)"
✅ "Phase 0 optimizes performance, LLM provides intelligence"

---

## Corrected Installation Flow

### Install + Configure (2 minutes)

```bash
# 1. Install CLI (30 seconds)
brew install rohankatakam/coderisk/crisk

# 2. Add API key (30 seconds)
crisk configure
# Or manually:
export OPENAI_API_KEY="sk-..."

# 3. Use immediately (works with API key)
cd my-repo
crisk check
# ✅ Phase 0 + Phase 1 + Phase 2 complete
```

### Optional: Full Graph Mode (10 minutes)

```bash
# Start Docker infrastructure
docker compose up -d

# Ingest repository
crisk init-local  # 5-15 min for medium repos

# Full analysis with graph data
crisk check --explain
```

---

## Corrected Pricing

### Free Tier: ❌ DOES NOT EXIST

There is no "free CLI" tier because the CLI requires OpenAI API key.

### Actual Pricing:

**Self-Hosted (Local Mode):**
- Cost: $0.03-0.05/check (your OpenAI API costs)
- Infrastructure: Free (you run Docker locally)
- Setup: 10 minutes (Docker + API key)
- Best for: Individual developers, small teams

**Cloud Platform:**
- Cost: $10-50/user/month + $0.03-0.05/check (LLM)
- Infrastructure: We host Neptune, no Docker needed
- Setup: 30 seconds (just API key)
- Best for: Teams, enterprises

### Cost Comparison:

| Solution | Setup | Per-Check | Monthly (100 checks) |
|----------|-------|-----------|----------------------|
| **CodeRisk Local** | 10 min | $0.04 | **$4** |
| **CodeRisk Cloud** | 30 sec | $0.04 + $0.10 | **$14** ($10 + $4 LLM) |
| SonarQube | 30 min | $0 (bundled) | $150/user |
| Greptile | 5 min | $0 (bundled) | $70/user |

**CodeRisk advantage:** Transparent LLM costs + cheapest cloud option

---

## What Makes CodeRisk Special (Corrected)

### Not Special: "Works without API key"

❌ This is false. Nothing works without API key.

### Actually Special:

1. **<3% False Positive Rate**
   - vs 10-20% industry standard
   - Achieved via LLM agentic investigation
   - Requires API key

2. **Transparent Costs (BYOK)**
   - You see exactly what you pay OpenAI
   - No 2-3x markup (like competitors)
   - You control spend

3. **Fast Setup**
   - 30 seconds: Add API key, start using
   - vs 30 minutes: SonarQube config
   - vs 5-10 minutes: Greptile indexing

4. **Optional Local Mode**
   - Run entirely on your machine
   - No data sent to us (just OpenAI)
   - Full control

5. **Phase 0 Optimization**
   - Skips expensive LLM calls for docs-only (saves $)
   - Escalates security keywords immediately
   - Adaptive thresholds per domain

---

## Corrected Documentation Updates Needed

### Files to Update:

1. **configuration_management.md**
   - Remove "Tier 1: Phase 0 Only (Zero Config)"
   - Correct to: "Tier 1: Local Mode (API Key Required)"
   - Clarify Phase 0 is pre-filter, not standalone

2. **BACKEND_PACKAGING_PROMPT.md**
   - Remove: "Phase 0 works without API key"
   - Add: "API key setup is first-run requirement"
   - Update README instructions

3. **FRONTEND_WEBSITE_PROMPT.md**
   - Remove: "Free CLI, optional cloud"
   - Correct to: "Open source CLI (requires API key), optional cloud hosting"
   - Update pricing page (no "Free Forever" tier)

4. **CONFIG_MANAGEMENT_PROMPT.md**
   - Make API key setup REQUIRED, not optional
   - Remove "Phase 0 works immediately" messaging
   - Update to: "Quick setup: 30 seconds to add API key"

5. **packaging_and_distribution.md**
   - Clarify cost model (BYOK = $0.04/check, not free)
   - Update README template
   - Installation must include API key setup

6. **website_messaging.md**
   - Remove "Free Forever" tier
   - Correct to: "Open Source + BYOK" ($0.04/check)
   - Update pricing page content

7. **LAUNCH_SETUP_SUMMARY.md**
   - Correct "Works in 30 seconds" to "Setup in 2 min (install + API key)"
   - Remove "Phase 0 works immediately" throughout
   - Update success metrics

---

## Corrected Messaging Framework

### Headline (Homepage):

**Before (WRONG):**
```
Open Source AI Code Risk Assessment
Free CLI, optional cloud platform
```

**After (CORRECT):**
```
Open Source AI Code Risk Assessment
Transparent BYOK model - You control LLM costs
```

### Subheadline:

**Before (WRONG):**
```
Works immediately after install. No configuration needed.
```

**After (CORRECT):**
```
Quick setup: 2 minutes (install + API key). Optional cloud hosting.
```

### Value Props:

1. **<3% False Positives** (LLM-powered, not static analysis)
2. **Transparent Costs** (BYOK, $0.03-0.05/check, no markup)
3. **2-Minute Setup** (install + API key)
4. **Open Source** (audit code, contribute, self-host)

### Installation Instructions:

```markdown
## Installation

### 1. Install CLI (30 seconds)
\`\`\`bash
brew install rohankatakam/coderisk/crisk
\`\`\`

### 2. Add OpenAI API Key (30 seconds) - REQUIRED
\`\`\`bash
crisk configure
# Enter your OpenAI API key (sk-...)
\`\`\`

[Get API key →](https://platform.openai.com/api-keys)

**Cost:** $0.03-0.05 per check (you pay OpenAI directly, no markup)

### 3. Start Using (immediately)
\`\`\`bash
cd your-repo
crisk check
# ✅ Works! LLM-powered risk assessment
\`\`\`

### Optional: Docker for Full Graph Mode (10 minutes)
\`\`\`bash
docker compose up -d
crisk init-local
\`\`\`
```

---

## Updated Session Prompts Needed

All three session prompts need updates to reflect this correction:

### Session 1: Backend Packaging

**Key changes:**
- README must emphasize API key is required
- Remove "Phase 0 works without API key" notes
- Update install.sh to make API key setup more prominent
- `crisk --version` should check for API key on first run

### Session 2: Frontend Website

**Key changes:**
- No "Free Forever" pricing tier
- Pricing starts at "Open Source + BYOK" ($0.04/check)
- Headline: Remove "Free CLI"
- Installation: API key is step 2 (required)
- Value prop: Transparent costs, not free usage

### Session 3: Configuration Management

**Key changes:**
- `crisk configure` is REQUIRED first-run, not optional
- Remove "Phase 0 only" tier from docs
- `crisk check` without API key should error clearly
- `crisk doctor` should flag missing API key as critical

---

## Action Plan

1. **Immediate:** Update all documentation to reflect API key requirement
2. **Update prompts:** Correct all three session prompts
3. **Update messaging:** Remove "free tier" language from all materials
4. **Test flow:** Ensure first-run UX makes API key setup clear
5. **Pricing page:** Show "BYOK from $0.04/check" not "Free $0"

---

## Key Takeaway

**CodeRisk's value is the LLM agentic investigation, NOT Phase 0 static analysis.**

Phase 0 is an optimization that:
- Saves money (skips LLM for docs-only changes)
- Improves accuracy (escalates security immediately)
- Enables adaptivity (domain-aware configs)

But it does NOT provide standalone risk assessment.

**Corrected positioning:**
> "Open source AI code risk tool with transparent LLM costs. 2-minute setup. <3% false positives. $0.03-0.05/check (BYOK)."

---

**This correction is critical for accurate launch positioning and user expectations.**
