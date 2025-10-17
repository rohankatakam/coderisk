# Developer Workflows (MVP): Local Git + AI Coding

**Last Updated:** October 17, 2025
**Status:** Active - Simplified for MVP launch
**Focus:** Solo developers + small teams using AI coding assistants

> **📘 Strategic Simplification:** Focused on local git workflows with AI coding integration. Enterprise workflows, CI/CD pipelines, and OSS patterns archived to [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/) for v2-v4.

---

## Overview

Modern software development has evolved with AI coding assistants (Claude Code, Cursor, Copilot). CodeRisk integrates seamlessly into **local git workflows** to validate AI-generated code before it becomes public.

**Key Insight:** AI coding generates code 5-10x faster, but git workflows remain the same. CodeRisk adds a safety check at the pre-commit stage—the earliest intervention point.

---

## The AI Coding Challenge

### Traditional vs AI-Assisted Workflows

**Traditional Manual Coding:**
```bash
# Developer writes code manually (slow, deliberate)
vim src/auth.py         # 30-60 min
git add src/auth.py
git commit -m "Add rate limiting"
# Developer understands every line (low risk)
```

**AI-Assisted Coding (Claude Code/Cursor/Copilot):**
```bash
# Developer prompts AI (fast, less review)
> "Add rate limiting to auth endpoint with Redis"
# AI generates 3 files, 200 lines in 30 seconds
git add .
git commit -m "Add Redis rate limiting"
# Developer may not understand every line (higher risk)
```

**The Gap:** AI generates code faster than developers can thoroughly review. Pre-commit checks catch issues before they become public.

---

## Core Git Workflow Patterns

### Pattern 1: Direct Commit to Main (Solo Developers)

```bash
# Solo developer commits directly to main
git checkout main
# Make changes (manual or AI-assisted)
vim src/feature.py
git add .
git commit -m "Add feature X"
git push origin main
```

**CodeRisk Integration:**

```bash
# Pre-commit hook runs automatically
git commit -m "Add feature X"

🔍 CodeRisk: Analyzing 1 file... (1.2s)

✅ LOW risk - Safe to commit
   - Test coverage: 78%
   - No coupling issues

[main abc1234] Add feature X
```

**Value:** Acts as personal code reviewer when working alone.

---

### Pattern 2: Feature Branch Workflow (Small Teams)

```bash
# Create feature branch
git checkout -b feature/auth-improvements

# Make changes (manual or AI-assisted)
# ... code changes ...

git add .
git commit -m "Improve auth logic"

# Push to remote
git push origin feature/auth-improvements

# Create PR
gh pr create --title "Auth improvements"
```

**CodeRisk Integration:**

```bash
# Pre-commit hook runs on each commit
git commit -m "Improve auth logic"

🔍 CodeRisk: Analyzing 2 files... (2.1s)

⚠️  MEDIUM risk detected:
   - auth.py: 0% test coverage
   - High coupling with user_service.py

Fix or override: git commit --no-verify

# Developer adds tests before committing
```

**Value:** Catches issues before PR review, reduces review cycles.

---

## AI Coding Workflow Integration

### Scenario 1: AI Generates Feature (Validation Loop)

**Problem:** Developer uses Claude Code to generate feature, uncertain if it's safe.

**Solution:** Pre-commit hook validates before commit.

**Workflow:**

```bash
# Developer prompts Claude Code
> "Build a payment processing system with Stripe"

# Claude generates 5 files, 500 lines in 30 seconds
# - payment.py
# - stripe_client.py
# - webhook_handler.py
# - config.py
# - tests/test_payment.py

# Developer reviews briefly, looks good
git add .
git commit -m "Add Stripe payment processing"

# Pre-commit hook triggers
🔍 Analyzing AI-generated code... (2.3s)

🔴 HIGH risk detected:

   Security Issues:
   1. payment.py - No input validation (injection risk)
   2. config.py - Hardcoded API key (secrets exposure)
   3. webhook_handler.py - No signature verification

   Quality Issues:
   4. 0% test coverage for payment logic
   5. High complexity (15-20 per function)

❌ Commit blocked

💡 Fix these before committing:
   - Add input validation with Pydantic
   - Move API key to environment variable
   - Add webhook signature verification
   - Add payment tests

# Developer fixes issues (manually or with AI)
> "Add input validation to payment.py"
> "Add signature verification to webhook_handler.py"
> "Move API key to .env file"

# Re-commit
git commit -m "Add Stripe payment (security hardened)"

🔍 Analyzing... (1.8s)

✅ LOW risk
   - Security issues resolved ✅
   - Test coverage: 72% ✅
   - Input validation added ✅

[main abc1234] Add Stripe payment (security hardened)
```

**Key UX Elements:**
- Catches AI mistakes automatically
- Specific, actionable recommendations
- Fast feedback loop (2-3 seconds)
- Allows iterative fixes

---

### Scenario 2: Rapid Prototyping → Production

**Prototype Phase (Velocity Focus):**

```bash
git checkout -b spike/new-feature

# Use AI to generate prototype quickly
> "Build a real-time chat feature with WebSockets"
# AI generates 10 files in 2 minutes

# Solo dev mode: warnings only, never blocks
git commit -m "WIP: chat prototype"

⚠️  CRITICAL risk detected (expected for prototype)
   - 0% test coverage
   - No error handling
   - High coupling

✅ Committed (solo mode - warnings only)
💡 Tip: Harden before production
```

**Production Hardening Phase:**

```bash
git checkout -b feature/chat-production
git merge spike/new-feature

# Harden with AI + manual fixes
> "Add error handling and tests to chat system"

git commit -m "Add production hardening"

# Team mode: blocks on HIGH risk
🔴 HIGH risk: 0% test coverage for WebSocket handler

❌ Commit blocked (team mode)

# Fix incrementally
> "Add tests for WebSocket handlers"

git commit -m "Add WebSocket tests"

✅ LOW risk - Production ready
```

---

## Team Size Workflows

### Solo Developer / Side Project

**Git Pattern:**
- Direct commits to main (no PR process)
- Frequent small commits
- Self-review

**CodeRisk Behavior:**
- **Warnings only** (never blocks)
- Educational feedback
- Builds good habits

**Example:**

```bash
git commit -m "Add feature X"

⚠️  MEDIUM risk:
   - Missing tests (consider adding)
   - High coupling detected

✅ Committed (solo mode - warnings only)
💡 Tip: Tests prevent future bugs
```

**Why Warning-Only:**
- Solo devs decide their own risk tolerance
- No team to protect from risky code
- Faster iteration more important
- Educational, not enforcement

---

### Small Team (2-10 people)

**Git Pattern:**
- Feature branches
- Pull requests (optional or required)
- Quick review cycles
- Fast iteration

**CodeRisk Behavior:**
- **Blocks on HIGH/CRITICAL** risk only
- Allows MEDIUM/LOW risk (velocity)
- Override available for urgent cases
- Logged overrides for visibility

**Example:**

```bash
git commit -m "Add Stripe integration"

🔴 HIGH risk detected:
   - payment.py handles money but has 0% tests
   - No error handling for network failures

❌ Commit blocked (team mode)

Fix or override with: git commit --no-verify
(Overrides logged for team review)
```

**Why Block on HIGH:**
- Protects team from major incidents
- Maintains velocity for MEDIUM/LOW risks
- Override escape hatch for urgent needs
- Creates team visibility via logs

---

## Common Git Patterns + CodeRisk

### Pattern 1: WIP Commits (Iterative Development)

```bash
# Developers commit frequently with WIP
git commit -m "WIP"
git commit -m "WIP - fix tests"
git commit -m "WIP - almost there"

# Each triggers CodeRisk check
🔍 Analyzing... (1.2s)
⚠️  MEDIUM risk - continuing...
```

**Optimization:**

```bash
# Shell alias for fast WIP commits
alias gwip='git add . && crisk check --quiet && git commit -m "WIP"'

# Developer runs: gwip
# Gets instant feedback on each iteration
```

---

### Pattern 2: Hotfix Workflow (Production Issues)

```bash
# Production is down, urgent fix needed
git checkout -b hotfix/auth-bug

# Make quick fix (high pressure)
vim src/auth.py
git add src/auth.py
git commit -m "hotfix: fix null pointer"

# CodeRisk still checks (fast)
🔍 Analyzing... (0.8s)

⚠️  WARNING: auth.py has incident history (3 past issues)

✅ Committed (hotfix allows override)
💡 Consider adding test to prevent regression
```

**Value:** Even in hotfixes, shows incident history (helps avoid repeat issues).

---

### Pattern 3: Large Refactors (Many Files)

```bash
git commit -m "Refactor auth module"

🔍 CodeRisk: Analyzing 25 files...

[Progress]
▓▓▓▓▓▓▓▓▓▓░░░░░ 65% (analyzing coupling)

Estimated: 3s remaining

⏱️  Large changeset (25 files) - taking ~4 seconds

   Pro tip: Smaller commits = faster checks

✅ LOW risk (complete)
```

---

## Complete End-to-End Examples

### Example 1: Solo Developer + AI Coding

**Scenario:** Weekend side project, using Claude Code

```bash
# Saturday morning
cd ~/projects/my-saas
git checkout main

# Start new feature
> "Build a user analytics dashboard with Chart.js"

# Claude generates:
# - dashboard.html (120 lines)
# - api.py (80 lines)
# - analytics.js (150 lines)
# Total: 3 files, 350 lines in 45 seconds

# Quick review, looks good
crisk check

# Output:
⚠️  MEDIUM risk:
   - api.py: no input validation (security)
   - analytics.js: 12 API endpoints (high coupling)
   - 0% test coverage

# Fix with AI
> "Add input validation to api.py with Pydantic"
> "Add basic tests for api.py"

# Re-check
crisk check
✅ LOW risk

# Commit
git add .
git commit -m "Add analytics dashboard with validation"

[main abc1234] Add analytics dashboard
```

**Time:** 15 minutes (vs 2-3 hours manual)
**Risk:** Reduced via CodeRisk (caught security + testing gaps)

---

### Example 2: Small Team Using Cursor

**Scenario:** Startup with 5 developers, assigned Slack integration task

```bash
# Monday morning: new ticket
git checkout main
git pull
git checkout -b sara/slack-notifications

# Use Cursor to scaffold
> "Create Slack webhook integration for user signups"

# Cursor generates:
# - slack_client.py (100 lines)
# - webhook.py (60 lines)
# - config.py (30 lines)
# - tests/test_slack.py (80 lines)

# Manual tweaks (5 min)

# Pre-commit check
crisk check

⚠️  MEDIUM risk:
   - slack_client.py: no rate limit handling
   - webhook.py: high coupling with user_service.py

# Fix with AI
> "Add exponential backoff to slack_client.py"

crisk check
✅ LOW risk

# Commit
git commit -m "Add Slack notifications"
git push origin sara/slack-notifications

# Create PR
gh pr create --title "Slack notifications for signups"

# Tech lead reviews (15 min) → Approves → Merges
```

**Time:** 1 hour (vs 4-5 hours manual)
**Quality:** High (caught rate limiting issue before PR)

---

## Local-First Setup

### One-Time Repository Setup

```bash
# Install CodeRisk
brew install coderisk

# Navigate to repo
cd /path/to/your/repo

# Initialize
crisk init

# What happens:
🔍 CodeRisk Init
→ Starting local Neo4j (Docker)... ✅
→ Analyzing git history... ✅
→ Building graph database... ✅
→ Parsing codebase... ✅

Initialization complete! (2.5 minutes)

# Install pre-commit hook
crisk hook install

Pre-commit hook installed!
→ Will run before each commit
→ Override with: git commit --no-verify

# Set API key (one-time)
crisk config set openai-api-key sk-...

# Test it
crisk check
✅ Ready to use
```

---

## Configuration (Optional)

### Solo Developer Config

```bash
# Set to solo mode (warnings only)
crisk config set mode solo

# Confirm
crisk config list
mode: solo (warnings only, never blocks)
```

### Small Team Config

```bash
# Set to team mode (blocks on HIGH)
crisk config set mode team
crisk config set block-on high

# Confirm
crisk config list
mode: team
block-on: high (blocks HIGH/CRITICAL, allows MEDIUM/LOW)
```

---

## Shell Integration (Power Users)

### Useful Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc

# Quick check before commit
alias gcheck='crisk check'

# Check + commit if safe
alias gcommit='crisk check && git commit'

# WIP commits with check
alias gwip='crisk check --quiet && git commit -m "WIP"'

# Safe push (check before pushing)
alias gpush='crisk check && git push'
```

### Git Hooks Setup

```bash
# Install pre-commit hook (automatic)
crisk hook install

# This creates .git/hooks/pre-commit:
#!/bin/bash
crisk check --quiet
exit $?
```

---

## Workflow Principles

### 1. Early Intervention
- **Pre-commit** (earliest point, private)
- Before code becomes public
- Before PR review
- Before CI/CD

### 2. Fast Feedback
- <2s for single file
- <5s for small changes
- Progress indicator for large changes

### 3. Minimal Friction
- Runs automatically (pre-commit hook)
- Easy override (standard git flag)
- Smart defaults (no config needed)

### 4. AI Coding Native
- Designed for Claude Code, Cursor, Copilot
- Validates AI-generated code automatically
- Guides AI toward safer patterns

### 5. Local-First
- Everything runs locally (Docker + Neo4j)
- No cloud dependency (except LLM API)
- Fast, private, offline-capable

---

## Key Insights

### 1. AI Velocity × Safety Multiplier
- AI coding increases velocity 5-10x
- CodeRisk maintains safety without slowing down
- Result: Fast AND safe code

### 2. Shift-Left Quality Gates
- Traditional: Find issues in PR review (late)
- CodeRisk: Find issues pre-commit (early)
- Result: Faster PR cycles, less rework

### 3. Workflow Agnostic
- Works with direct-to-main (solo)
- Works with feature branches (teams)
- Works with any git pattern
- Result: Fits any workflow

### 4. Habit Formation
- `crisk check` becomes as natural as `git status`
- Pre-commit hook makes it automatic
- Result: Zero-friction safety check

---

## Success Metrics

### Adoption Metrics
- % of commits checked (target: >90%)
- % of developers with hook installed (target: >80%)
- % of overrides (target: <10% for teams)

### Workflow Metrics
- Average check time (target: <3s)
- Time to first commit (target: <5 min after install)
- Developer satisfaction (target: NPS >40)

### Quality Metrics
- Incidents prevented (target: 60-80% reduction)
- PR review cycles reduced (target: 30-50% fewer)
- Test coverage increased (target: +10-20%)

---

## Related Documents

**Product:**
- [mvp_vision.md](mvp_vision.md) - MVP vision and scope
- [user_personas.md](user_personas.md) - Ben (solo dev), Clara (small team)
- [simplified_pricing.md](simplified_pricing.md) - Free BYOK model

**User Experience:**
- [developer_experience.md](developer_experience.md) - Detailed UX patterns

**Archived (Future):**
- [../99-archive/00-product-future-vision/](../99-archive/00-product-future-vision/) - Enterprise workflows, CI/CD, OSS patterns (v2-v4)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP launch (Week 7-8), after 50+ user feedback
