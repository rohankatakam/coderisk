# Developer Experience (MVP): Local-First Pre-Commit Checks

**Last Updated:** October 17, 2025
**Status:** Active - Simplified for MVP launch
**Focus:** Solo developers + small teams using AI coding assistants

> **📘 Strategic Simplification:** Focused on local-first pre-commit workflow for MVP. Complex enterprise features (CI/CD, team dashboards, AI Mode) archived to [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/) for v2-v4.

---

## Design Philosophy

**Core Principle:** CodeRisk should be **invisible when safe, visible when risky**—like spell-check for code safety.

**Key Tenets:**
1. **Zero-friction activation** - `brew install` → Done, no configuration
2. **Instant feedback** - <5 seconds for pre-commit check
3. **Actionable guidance** - Tell developers *what to fix*, not just *what's wrong*
4. **Adaptive verbosity** - Quiet by default, detailed on request
5. **AI coding native** - Designed for Claude Code, Cursor, Copilot users

---

## The AI Coding Challenge

### Problem: AI Generates Code Faster Than Humans Can Review

**Traditional Manual Coding:**
```
Developer writes code: 50-100 lines/hour
Developer reviews own code: Built-in (continuous)
Risk: Low (human writes what they understand)
```

**AI Coding (Claude Code/Cursor/Copilot):**
```
AI generates code: 500-1,000 lines/hour (10x faster)
Developer reviews AI code: Often cursory (trust AI)
Risk: HIGH (developer may not understand all implications)
```

**The Gap:** Developers commit AI-generated code with less scrutiny than hand-written code.

**CodeRisk's Role:** Automated pre-commit reviewer that matches AI velocity.

---

## Seamless Integration: Pre-Commit Hook

### Goal: Risk check happens automatically before every commit

**One-Time Setup:**
```bash
# Install CodeRisk
brew install coderisk

# Initialize local Neo4j + build graph
crisk init

# Install pre-commit hook
crisk hook install
```

**What `crisk init` Does:**
1. Starts local Neo4j in Docker container
2. Analyzes git history to build graph
3. Parses codebase with tree-sitter
4. Creates baseline metrics (takes 1-5 min for medium repos)

**What `crisk hook install` Does:**
1. Creates `.git/hooks/pre-commit` script
2. Configures hook to run `crisk check --quiet` before each commit
3. Hook respects git's `--no-verify` flag for overrides

---

## User Experience Flow

### Success Case (Low Risk)

```bash
# Developer commits (manual or AI-generated code)
git add src/auth.py
git commit -m "Add rate limiting to auth endpoint"

# Hook triggers automatically (developer sees):
🔍 CodeRisk: Analyzing 1 file... (1.2s)

✅ LOW risk - Safe to commit
   - Test coverage: 78%
   - Coupling score: 3/10
   - No incident history

[main abc1234] Add rate limiting to auth endpoint
 1 file changed, 45 insertions(+), 12 deletions(-)
```

**Key UX Points:**
- Runs automatically (no manual command)
- Fast (1-2 seconds for single file)
- Clear verdict (safe to commit)
- Minimal output (one-line summary)

---

### Risk Detected Case (Medium/High Risk)

```bash
git add src/payment.py src/stripe_client.py
git commit -m "Add payment processing"

# Hook triggers:
🔍 CodeRisk: Analyzing 2 files... (2.1s)

⚠️  MEDIUM risk detected:

   Issues:
   1. payment.py has 0% test coverage (CRITICAL for payments)
   2. stripe_client.py calls 8 other functions (high coupling)
   3. payment.py similar to past incident INC-453 (timeout issue)

   Recommendations:
   - Add tests for payment.py before committing
   - Review coupling with database layer
   - Add error handling for network timeouts

   Run 'crisk check --explain' for full investigation

❌ Commit blocked. Fix issues or override with:
   git commit --no-verify
```

**Key UX Points:**
- Specific issues listed (not vague warnings)
- Explains *why* it matters (CRITICAL for payments)
- Actionable recommendations (add tests, review coupling)
- Easy override path (standard git flag)
- Suggests next command (`--explain` for details)

---

### Developer Decision Tree

```
Risk detected → Developer has 3 choices:

1. Fix issues (RECOMMENDED)
   → Add tests, reduce coupling
   → Re-commit (auto-checks again)
   → Iterate until LOW risk

2. Override (low friction)
   → git commit --no-verify -m "..."
   → Logs override (local file for later review)

3. Get details
   → crisk check --explain
   → See full investigation trace
   → Make informed decision
```

---

## Adaptive Verbosity (3 Modes)

### Mode 1: Quiet (Pre-Commit Hook Default)

```bash
crisk check --quiet

# Output (success):
✅ LOW risk

# Output (issues):
⚠️ MEDIUM risk: 3 issues detected
Run 'crisk check' for details
```

**Use Case:** Pre-commit hook (minimal noise)

---

### Mode 2: Standard (CLI Default)

```bash
crisk check

# Output:
🔍 CodeRisk Analysis
Files changed: 2
Risk level: MEDIUM

Issues:
1. ⚠️  auth.py - No test coverage (0%)
2. ⚠️  auth_middleware.py - High coupling (8 dependencies)
3. ℹ️  user_service.py - Changed with auth.py in 85% of commits

Recommendations:
- Add tests for auth.py (prevents regression)
- Review dependencies in auth_middleware.py
- Consider extracting shared logic to reduce coupling

Run 'crisk check --explain' for full investigation
```

**Use Case:** Manual CLI check (developer wants context)

---

### Mode 3: Explain (Full Investigation)

```bash
crisk check --explain

# Output:
🔍 CodeRisk Investigation Report
Started: 2025-10-17 14:23:15
Completed: 2025-10-17 14:23:17 (2.1s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Changed File Analysis: auth.py
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Changed functions:
  - authenticate_user() (lines 45-67)
  - validate_token() (lines 89-102)

Metrics:
  ✅ Complexity: 6 (target: <10)
  ❌ Test coverage: 0% (target: >70%)
  ⚠️  Import count: 8 (high coupling)

Pattern match: Risky authentication pattern (shared session state)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Temporal Coupling Analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Coupled files:
  - user_service.py: Co-changed in 17 of 20 commits (85%)
  - database.py: Co-changed in 12 of 20 commits (60%)

Strong coupling detected (>75% threshold)
Recent co-changes: 5 in last 30 days

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Incident History (Optional - if DB enabled)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Similar past incidents:
  - INC-453 (2025-09-15): Auth timeout cascade failure
  - Root cause: auth.py + user_service.py coupling

Pattern: auth.py changes often cause user_service issues

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Final Assessment
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Risk Level: MEDIUM → HIGH (elevated due to incident history)

Evidence:
  1. Zero test coverage on critical auth functions
  2. Strong temporal coupling with user_service.py (85%)
  3. Similar to incident INC-453 (3 months ago)
  4. High coupling (8 dependencies)

Recommendations (priority order):
  1. Add integration tests for authenticate_user() + user_service interactions
  2. Add unit tests for validate_token()
  3. Review coupling: Can user_service use an interface?
  4. Consider circuit breaker pattern (given timeout history)
```

**Use Case:** Deep dive (developer wants full context before fixing)

---

## AI Coding Workflow Integration

### Pattern: AI-Generated Code Validation

**Problem:** Developer uses Claude Code to generate feature, uncertain if it's safe.

**Solution:** Pre-commit hook catches issues before they become public.

**UX Flow:**

```bash
# Developer prompts Claude Code
> "Build a payment processing system with Stripe"

# Claude generates 5 files, 500 lines in 30 seconds

# Developer reviews briefly, looks good
git add .

# About to commit
git commit -m "Add payment processing"

# Pre-commit hook triggers
🔍 Analyzing AI-generated code... (2.3s)

🔴 HIGH risk in AI-generated code:

   Security Issues:
   1. payment.py - No input validation (injection risk)
   2. stripe_client.py - No rate limiting (DoS risk)
   3. config.py - Hardcoded API key (secrets exposure)

   Quality Issues:
   4. 0% test coverage across 5 files
   5. High complexity (15-20 per function)

❌ Commit blocked

💡 Fix these issues before committing:
   - Add input validation to payment.py
   - Add rate limiting to stripe_client.py
   - Move API key to environment variable
   - Add basic test coverage (target: >70%)

# Developer fixes issues (manually or with Claude)
# Re-commits
git commit -m "Add payment processing (security hardened)"

🔍 Analyzing... (1.8s)

✅ LOW risk
   - Security issues resolved ✅
   - Test coverage: 72% ✅
   - Complexity acceptable ✅

[main abc1234] Add payment processing (security hardened)
```

**Key UX Elements:**
- Catches AI security mistakes automatically
- Specific, actionable recommendations
- Allows incremental improvement (not perfectionism)
- Fast feedback loop (2-3 seconds)

---

## Error Messages & Guidance

### Principle: Tell Developers What To DO, Not Just What's Wrong

**Bad Error Message:**
```
❌ High coupling detected
```

**Good Error Message:**
```
⚠️  High coupling detected: auth.py calls 8 other functions

Why this matters:
  - Changes to auth.py likely affect 8+ files
  - Increases chance of breaking unrelated features
  - Makes code harder to test in isolation

What to do:
  1. Review if all 8 dependencies are necessary
  2. Consider dependency injection for easier testing
  3. Extract shared logic to reduce coupling

Similar past issue: INC-453 (auth timeout cascade failure)
```

---

### Actionable Commands (Always Provide Next Step)

Every warning includes a suggested next command:

```bash
⚠️  No test coverage
   → Run 'crisk check --explain' to see full analysis

⚠️  High coupling
   → Review coupling with: git log --follow src/auth.py

⚠️  Incident history
   → See past incidents: crisk incident search "auth"

⚠️  Complex function
   → Consider refactoring authenticate_user() (complexity: 12)
```

---

## Performance & Timing UX

### The 5-Second Rule

**Principle:** Risk check must complete <5s to not disrupt flow.

**Why:**
- <2s feels instant (doesn't break flow)
- 2-5s acceptable (developer expects analysis)
- >5s frustrating (developer considers `--no-verify`)

**UX for Normal Checks:**

```bash
git commit -m "Update auth logic"

🔍 CodeRisk: Analyzing 2 files... (1.2s)

✅ LOW risk
```

**UX for Slow Checks (Large Changeset):**

```bash
git commit -m "Large refactor"

🔍 CodeRisk: Analyzing 45 files...

[Progress indicator]
▓▓▓▓▓▓▓▓▓▓░░░░░ 65% (analyzing temporal coupling)

Estimated: 3s remaining

# If >5s, show tip:
⏱️  Large changeset (45 files) - taking ~7 seconds

   Pro tip: Smaller commits = faster checks
```

---

## Team Size Adaptive UX

### Solo Developer / Side Project

**UX Goal:** Personal assistant, minimal friction

**Behavior:**
- Pre-commit hook: WARNING only (never blocks for solo devs)
- Verbosity: Standard
- Override: Always allowed
- Focus: Education + safety, not enforcement

**Example:**

```bash
git commit -m "Add feature X"

⚠️  MEDIUM risk:
   - Missing tests (consider adding)
   - High coupling detected

✅ Committed (solo mode - warnings only)
💡 Tip: Add tests to prevent regressions
```

**Why Warning-Only for Solo:**
- Solo devs decide their own risk tolerance
- No team to protect from risky changes
- Faster iteration more important
- Educational (learn what's risky) not enforcement

---

### Small Team (2-10 people)

**UX Goal:** Balance velocity with quality

**Behavior:**
- Pre-commit hook: Blocks on HIGH/CRITICAL only
- Verbosity: Standard with suggestions
- Override: Allowed but logged
- Focus: Prevent major issues, allow minor risks

**Example:**

```bash
git commit -m "Add Stripe integration"

🔴 HIGH risk detected:
   - payment.py handles money but has 0% tests
   - No error handling for network failures

❌ Commit blocked (team mode - HIGH risk)

Fix or override with: git commit --no-verify
(Overrides logged for team visibility)
```

**Why Block on HIGH for Teams:**
- Protects team from major incidents
- Allows velocity for MEDIUM/LOW risks
- Override available for urgent cases
- Logged overrides create visibility

---

## CLI Interaction Patterns

### 1. Progressive Disclosure

**Start simple, reveal complexity on demand:**

```bash
crisk check
# Shows: Risk level + summary

crisk check --explain
# Shows: Full investigation trace

crisk check --explain --json
# Shows: Machine-readable output (future)
```

### 2. Smart Defaults

**No configuration required:**

```bash
# Works immediately (smart defaults)
crisk check

# Customize if needed (optional)
crisk config set mode team          # solo | team
crisk config set block-on high      # low | medium | high
```

### 3. Standard Git Integration

**Follows git conventions:**

```bash
# Standard git override
git commit --no-verify

# Standard exit codes
crisk check
echo $?  # 0 = safe, 1 = risky, 2 = error

# Standard output (stdout/stderr)
crisk check > report.txt 2>&1
```

---

## Local-First Setup UX

### Initial Setup (One-Time)

```bash
# Install CodeRisk
brew install coderisk

# Initialize (one-time per repo)
cd /path/to/your/repo
crisk init

# What happens:
🔍 CodeRisk Init
→ Starting local Neo4j (Docker)... ✅
→ Analyzing git history (150 commits)... ✅
→ Building graph database... ✅
→ Parsing codebase (450 files)... ✅

Initialization complete! (2.5 minutes)

Next steps:
  1. Install pre-commit hook: crisk hook install
  2. Set your API key: crisk config set openai-api-key sk-...
  3. Test it: crisk check
```

**What's Created Locally:**
- `.coderisk/` directory (config, cache)
- Docker container for Neo4j (local graph DB)
- `.git/hooks/pre-commit` (if installed)

**Data Stored Locally:**
- Neo4j graph database (AST + git history)
- Metrics cache (test coverage, complexity)
- Configuration (.coderisk/config.yml)

**Data NOT Stored (Privacy):**
- Source code never sent to cloud (except LLM API call)
- No telemetry without opt-in
- All analysis happens locally

---

## Cost Transparency

### BYOK Model (Bring Your Own Key)

**User Experience:**

```bash
# One-time setup: Configure API key
crisk config set openai-api-key sk-...
# OR
crisk config set anthropic-api-key sk-ant-...

# Usage is transparent
crisk check
🔍 Analyzing... (1.2s)
💰 LLM cost: ~$0.01 (charged to your OpenAI account)

✅ LOW risk

# View monthly costs
crisk stats --costs
📊 Usage Stats (Last 30 days)
   - Checks run: 120
   - LLM API calls: 45 (only HIGH risk files analyzed)
   - Estimated cost: $1.20 (120 × $0.01/check)
   - Charged to: your OpenAI account
```

**Why BYOK:**
- Transparent costs (see exact LLM spend)
- No markup (pay OpenAI/Anthropic directly)
- User controls provider choice
- 95%+ cheaper than competitors

---

## Key UX Principles Summary

1. **Invisible when safe, visible when risky**
   - Low noise for clean code
   - Surface critical issues clearly

2. **Velocity-preserving**
   - Never block unnecessarily (solo: warnings only)
   - <5s for fast feedback
   - Easy override path (standard git flag)

3. **Actionable guidance**
   - Every warning includes "what to do"
   - Suggested next steps
   - Links to full explanation

4. **Local-first & private**
   - Runs in Docker on user's machine
   - No cloud dependency (except LLM API)
   - Fast, private, offline-capable (except LLM)

5. **AI coding native**
   - Designed for Claude Code, Cursor, Copilot users
   - Validates AI-generated code automatically
   - Fast feedback loop (2-5 seconds)

6. **Cost transparent**
   - BYOK model (user pays LLM directly)
   - Show estimated costs
   - No hidden fees

---

## Success Metrics (UX Quality)

**Adoption Metrics:**
- % of developers with pre-commit hook installed (target: >80%)
- % of commits checked (target: >90%)
- % of overrides (target: <10% for teams, any % for solo)

**Satisfaction Metrics:**
- Developer NPS (target: >40)
- "CodeRisk saved me from incident" stories (target: 5/month)
- Time to first value (target: <5 minutes from install)

**Performance Metrics:**
- Average check time (target: <3s)
- p95 check time (target: <5s)

---

## Related Documents

**Product:**
- [mvp_vision.md](mvp_vision.md) - MVP vision and scope
- [user_personas.md](user_personas.md) - Ben (solo dev), Clara (small team)
- [simplified_pricing.md](simplified_pricing.md) - Free BYOK model

**Workflows:**
- [developer_workflows.md](developer_workflows.md) - Git workflow integration

**Archived (Future):**
- [../99-archive/00-product-future-vision/](../99-archive/00-product-future-vision/) - Enterprise UX, CI/CD, dashboards (v2-v4)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP launch (Week 7-8), after 50+ user feedback
