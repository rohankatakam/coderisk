# Developer Workflows: Git, Vibe Coding, and CodeRisk Integration

**Last Updated:** October 3, 2025
**Purpose:** Map how CodeRisk (crisk) seamlessly integrates into modern developer workflows across team sizes, coding styles, and git patterns

---

## Overview

Modern software development exists on a spectrum from **traditional git workflows** to **AI-assisted "vibe coding"** with tools like Claude Code and Cursor. CodeRisk is designed to integrate seamlessly into both paradigms, becoming as reflexive as `git status` regardless of how developers write code.

**Key Insight:** Whether developers write code manually or use AI assistance, the **git workflow patterns remain identical**—but vibe coding accelerates iteration velocity, making pre-commit risk assessment even more critical.

---

## The Git Workflow Spectrum

### Traditional Manual Coding
```bash
# Developer writes code manually in IDE
vim src/auth.py
# Frequent commits, careful testing
git add src/auth.py
git commit -m "Add rate limiting to auth endpoint"
```

### Vibe Coding (AI-Assisted)
```bash
# Developer prompts Claude Code
> "Add rate limiting to the auth endpoint with Redis backend"
# AI generates 200 lines across 3 files in 30 seconds
git add src/auth.py src/middleware/rate_limit.py tests/test_auth.py
git commit -m "Add Redis-based rate limiting to auth"
```

**The Risk:** Vibe coding generates more code, faster, with less manual review—but uses the **exact same git workflow**.

---

## Core Git Patterns (Team Size Agnostic)

These patterns appear universally across solo developers, startups, and large enterprises:

### 1. Feature Branch Workflow (Universal)
```bash
# Start new feature
git checkout -b feature/auth-improvements
# Make changes (manual or AI-assisted)
# ... code changes ...
git add .
git commit -m "WIP: auth improvements"
# Continue iterating
git push origin feature/auth-improvements
# Create PR when ready
gh pr create --title "Auth improvements"
```

### 2. Main Branch Protection (Teams 5+)
- Main/master is protected
- All changes via pull requests
- CI checks required before merge
- Code review required (1-2 approvers)

### 3. Commit Frequency Patterns

| Developer Type | Commits/Day | Commit Style | Risk Profile |
|----------------|-------------|--------------|--------------|
| **Manual coder** | 5-15 | Small, incremental | Lower blast radius |
| **Vibe coder (beginner)** | 3-8 | Large, multi-file | Higher blast radius |
| **Vibe coder (experienced)** | 10-20 | Mixed (AI + manual fixes) | Variable |

---

## Team-Size Specific Workflows

### Solo Developer / Side Project (1 person)
**Git Pattern:**
```bash
# Often commits directly to main
git checkout main
# Make changes
git add .
git commit -m "Add feature X"
git push origin main
```

**CodeRisk Integration:**
```bash
# Pre-commit check (catches issues before they hit main)
crisk check
# Output: "MEDIUM risk: auth.py has 0% test coverage"
# Fix issues
git add tests/test_auth.py
git commit --amend
crisk check  # Verify fix
git push
```

**Value:** Acts as a personal code reviewer when working alone.

---

### Startup / Small Team (2-10 people)

**Git Pattern:**
```bash
# Feature branches, but fast iteration
git checkout -b feature/payments
# Vibe code with Claude Code (3-5 files changed)
> "Integrate Stripe payments with webhook handling"
# Quick self-review
git add .
git commit -m "Add Stripe integration"
git push origin feature/payments
# Create PR (often self-merge or single reviewer)
gh pr create --fill
```

**CodeRisk Integration (Pre-Commit):**
```bash
# Before committing vibe-generated code
crisk check
# Output:
# HIGH risk detected:
#   - payment_handler.py calls 8 other functions (high coupling)
#   - webhook.py has no tests
#   - payment_handler.py changed with database.py in 90% of commits
# Developer reviews AI code more carefully
# Adds tests before committing
```

**CodeRisk Integration (Pre-PR):**
```bash
# Before opening PR
git commit -am "Add Stripe integration"
crisk check --branch feature/payments
# Output: "50 files changed on this branch vs main"
# crisk creates branch delta graph (3-5s)
# Shows cumulative risk across all branch changes
```

**Value:** Prevents vibe-coded changes from breaking existing code, catches missing tests.

---

### Growth Company (10-50 people)

**Git Pattern:**
```bash
# Stricter branch naming, PR templates
git checkout -b rohan/JIRA-123-add-caching
# Mix of manual and AI coding
# Multiple commits per feature
git add src/cache.py
git commit -m "[JIRA-123] Add Redis cache layer"
# More commits...
git commit -m "[JIRA-123] Add cache invalidation"
git push origin rohan/JIRA-123-add-caching
# PR requires CI + 2 reviewers
gh pr create --template feature.md
```

**CodeRisk Integration (CI Pipeline):**
```yaml
# .github/workflows/ci.yml
name: CI
on: [pull_request]
jobs:
  risk-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: CodeRisk Check
        run: |
          crisk check --branch ${{ github.head_ref }}
          # Fails if CRITICAL risk detected
          # Posts comment to PR with risk summary
```

**CodeRisk Integration (Local Pre-Commit Hook):**
```bash
# .git/hooks/pre-commit
#!/bin/bash
crisk check --quick
if [ $? -ne 0 ]; then
  echo "❌ CodeRisk detected issues. Run 'crisk check --explain' for details."
  exit 1
fi
```

**Value:** Automated quality gate before code review, reduces reviewer burden.

---

### Enterprise (50-500+ people)

**Git Pattern:**
```bash
# Monorepo with strict policies
git checkout -b users/rohan.katakam/feature/auth-rbac
# Extensive branch protection
# Multiple levels of review (team lead, security, architect)
# Slow PR cycle (1-3 days)
git add services/auth/rbac.go
git commit -m "feat(auth): Add RBAC middleware"
# Push triggers CI/CD pipeline
git push origin users/rohan.katakam/feature/auth-rbac
# PR requires 3 approvals + security scan + architect review
```

**CodeRisk Integration (Multi-Stage Gates):**

**Stage 1: Developer Local Check**
```bash
# Before committing
crisk check --mode=enterprise
# Uses self-hosted Neptune in VPC
# No data leaves corporate network
```

**Stage 2: Pre-Commit Hook**
```bash
# Enforced via git hooks
crisk check --policy=strict
# Blocks commit if:
# - Test coverage < 70%
# - Coupling score > 8
# - Touches incident-prone files
```

**Stage 3: CI Pipeline**
```yaml
# Required status check on PR
- name: CodeRisk Enterprise Scan
  run: |
    crisk check \
      --branch $BRANCH \
      --compare main \
      --output sarif \
      --fail-on critical
    # Upload to security dashboard
```

**Stage 4: PR Comment Bot**
```markdown
## CodeRisk Analysis

**Branch:** feature/auth-rbac
**Risk Level:** MEDIUM

### Findings:
- ✅ Test coverage: 78% (target: 70%)
- ⚠️  High coupling detected in `rbac.go` (8 dependencies)
- ⚠️  File `auth_service.go` has incident history (3 incidents in 90 days)

### Recommendations:
1. Review coupling with `user_service.go` (changed together in 85% of commits)
2. Add integration tests for RBAC edge cases
```

**Value:** Prevents security/reliability issues in critical systems, provides audit trail.

---

## Vibe Coding Integration Patterns

### Pattern 1: Iterative AI Generation

**Scenario:** Developer uses Claude Code to generate feature, then refines.

```bash
# Initial AI generation
> "Create a user authentication system with JWT tokens"
# Claude generates 5 files, 400 lines

# Before committing, run crisk
crisk check
# Output: CRITICAL - no tests, high complexity

# Refine with AI
> "Add comprehensive tests for the auth system"
# Claude adds test files

crisk check
# Output: MEDIUM - test coverage improved, but high coupling remains

# Manual refinement
vim src/auth.py  # Developer reduces coupling

crisk check
# Output: LOW - Ready to commit
git add .
git commit -m "Add JWT authentication with tests"
```

**CodeRisk Value:** Guides AI-assisted iteration toward safer code.

---

### Pattern 2: Rapid Prototyping → Production Hardening

**Vibe Coding Phase (Velocity Focused):**
```bash
git checkout -b spike/new-feature
# Generate prototype quickly with AI
> "Build a real-time chat feature with WebSockets"
# AI generates 10 files in 2 minutes
git add .
git commit -m "WIP: chat prototype"
crisk check
# Output: CRITICAL (expected for prototype)
# Developer ignores for now (prototyping)
```

**Production Hardening Phase (Safety Focused):**
```bash
git checkout -b feature/chat-production
git merge spike/new-feature
# Now harden with AI assistance
> "Add error handling, tests, and logging to chat system"
crisk check --fix-suggestions
# Output:
# - Add error handling in message_handler.py (line 45)
# - Test coverage for websocket_manager.py is 0%
# - connection_pool.py has high coupling (7 deps)

# Iteratively fix with AI + manual review
crisk check  # Re-run after each fix
# Eventually: LOW risk → Ready for PR
```

**CodeRisk Value:** Differentiates prototype code from production-ready code.

---

### Pattern 3: AI Code Review Assistant

**Scenario:** Developer reviews AI-generated code with CodeRisk.

```bash
# Claude Code generates complex refactor
> "Refactor the payment processing module to use async/await"
# 15 files changed, 800 lines modified

# Before accepting changes
crisk check --explain --branch feature/async-payments
# Output (detailed investigation trace):
#
# Investigation Summary (5 hops, 12s):
#
# Hop 1: payment_processor.py
#   - Complexity increased from 8 → 15
#   - Now async, but no error handling for network failures
#
# Hop 2: database.py (coupled 85% co-change rate)
#   - Not updated to async → BLOCKING BUG
#
# Hop 3: test_payments.py
#   - Tests not updated for async → will fail
#
# CRITICAL: database.py still uses sync calls, will deadlock

# Developer catches the issue
> "Update database.py to use async database driver"
# Fix applied, re-check
crisk check
# Output: MEDIUM (much better)
```

**CodeRisk Value:** Acts as second reviewer for AI-generated code.

---

## Open Source Workflow Integration

### OSS Maintainer Workflow

**Traditional OSS Flow:**
```bash
# Contributor opens PR
# Maintainer reviews manually (time-consuming)
# Back-and-forth on code quality issues
```

**With CodeRisk (Automated Quality Check):**
```yaml
# .github/workflows/pr-check.yml
name: CodeRisk OSS Check
on: [pull_request]
jobs:
  risk-check:
    runs-on: ubuntu-latest
    steps:
      - uses: coderisk/action@v1
        with:
          mode: oss
          # Uses shared public cache (instant, no build time)
          # Posts PR comment with risk assessment
```

**PR Comment (Automated):**
```markdown
## CodeRisk Analysis

**Status:** ✅ LOW risk (safe to merge)

### Changes:
- Added feature X (50 lines)
- Test coverage: 85%
- No coupling issues detected

**Maintainer Note:** This PR looks safe from an architectural perspective.
```

**Value for OSS:**
- Reduces maintainer burden
- Provides objective quality signal
- Educates contributors on code quality

---

### OSS Contributor Workflow

**Before Opening PR:**
```bash
# Fork and clone repo
git clone https://github.com/facebook/react
cd react

# Make changes
vim packages/react/src/ReactHooks.js

# Check impact before opening PR
crisk check --public
# Uses shared cache for React (instant access, no build)
# Output: MEDIUM - your change affects 50+ dependent functions

# Contributor adds tests to reduce risk
vim packages/react/src/__tests__/ReactHooks-test.js

crisk check --public
# Output: LOW - ready to submit PR

git push origin my-feature-branch
gh pr create
```

**Value:** Contributors self-assess quality before maintainer review.

---

## Team Size Adoption Patterns

### Startup → Scale-Up Evolution (1 → 50 people)

**Phase 1: Solo/Small Team (1-5 people)**
- **Usage:** `crisk check` before commits (optional, personal discipline)
- **Value:** Personal code reviewer
- **Adoption:** Organic (developers discover via word-of-mouth)

**Phase 2: Product-Market Fit (5-15 people)**
- **Usage:** Add to pre-commit hooks (recommended)
- **Value:** Prevents vibe-coding accidents
- **Adoption:** Team lead mandates for critical paths

**Phase 3: Scaling (15-50 people)**
- **Usage:** CI/CD integration (required status check)
- **Value:** Automated quality gate
- **Adoption:** Engineering manager enforces via branch protection

**Phase 4: Enterprise (50+ people)**
- **Usage:** Self-hosted, integrated with security tooling
- **Value:** Compliance, audit trail, risk reduction
- **Adoption:** Top-down (CTO/VP Eng decision)

---

## Common Git Behaviors → CodeRisk Injection Points

### Behavior 1: "WIP Commits" (All team sizes)
```bash
# Developers commit frequently with WIP messages
git add .
git commit -m "WIP"
git commit -m "WIP - still broken"
git commit -m "WIP - almost there"
```

**CodeRisk Injection:**
```bash
# Add to shell alias
alias gwip='git add . && crisk check --quick && git commit -m "WIP"'
# Developer now runs: gwip
# CodeRisk checks even on WIP commits (fast feedback)
```

---

### Behavior 2: "Squash Before Merge" (Teams 10+)
```bash
# Many WIP commits on branch
git log --oneline
# ab12 WIP
# cd34 WIP - fix tests
# ef56 WIP - final

# Before merge, squash commits
git rebase -i main
# Squash all into one commit
```

**CodeRisk Injection:**
```bash
# After squash, before push
crisk check --branch feature/my-feature
# Checks cumulative risk of all changes
# (Not just last commit, but entire branch delta)
```

---

### Behavior 3: "Hotfix Workflow" (All sizes, production issues)
```bash
# Production is down, need urgent fix
git checkout -b hotfix/auth-bug
# Make quick fix (often manual, high pressure)
vim src/auth.py
git add src/auth.py
git commit -m "hotfix: fix null pointer in auth"
git push origin hotfix/auth-bug
# Merge immediately (no review)
```

**CodeRisk Injection:**
```bash
# Even in hotfix, run quick check
crisk check --mode=hotfix
# Uses cached graph (fast, <2s)
# Output: "⚠️  WARNING: auth.py has 3 incident history items"
# Developer adds extra validation before merging
```

---

### Behavior 4: "Stacked Diffs" (Advanced teams, Meta/Google style)
```bash
# Developer creates multiple dependent PRs
git checkout -b feature/step1
# ... changes ...
git commit -m "Step 1"
git push

git checkout -b feature/step2
# ... depends on step1 ...
git commit -m "Step 2"
git push
```

**CodeRisk Injection:**
```bash
# Check each diff independently
crisk check --branch feature/step1
crisk check --branch feature/step2 --base feature/step1
# Checks step2 changes relative to step1 (not main)
```

---

## Vibe Coding Prevalence by Organization Type

### Startups & Small Teams (1-20 people)
**Vibe Coding Adoption: 60-80%**
- High AI tool usage (Claude Code, Cursor, Copilot)
- Velocity over process
- Less code review rigor
- **CodeRisk Value:** Highest—acts as missing code reviewer

### Mid-Size Tech Companies (20-200 people)
**Vibe Coding Adoption: 40-60%**
- Mixed adoption (some teams yes, some no)
- Transitioning from startup to process-driven
- **CodeRisk Value:** High—helps standardize quality across teams

### Large Tech Companies (200-5000 people)
**Vibe Coding Adoption: 20-40%**
- Slower adoption due to security/compliance
- Pockets of innovation (internal tools teams)
- **CodeRisk Value:** Medium—complements existing tooling

### Enterprises (5000+ people)
**Vibe Coding Adoption: 5-20%**
- Highly regulated, slow adoption
- Security concerns around AI code generation
- **CodeRisk Value:** Medium—more focused on compliance/audit

### Open Source (Public repos)
**Vibe Coding Adoption: 30-50%**
- Maintainers skeptical, contributors enthusiastic
- Quality variance (AI-generated PRs often lower quality)
- **CodeRisk Value:** Very High—helps maintainers filter PRs

---

## Complete End-to-End Workflow Examples

### Example 1: Solo Vibe Coder (Side Project)

```bash
# Saturday morning, coffee in hand
cd ~/projects/my-saas
git checkout main
git pull

# Start new feature with Claude Code
git checkout -b feature/user-dashboard
> "Build a user analytics dashboard with charts using Chart.js"

# Claude generates:
# - dashboard.html (120 lines)
# - dashboard_api.py (80 lines)
# - analytics.js (150 lines)
# Total: 3 files, 350 lines, 45 seconds

# Quick manual review in IDE
# Looks good, but check safety
crisk check

# Output:
# MEDIUM risk detected:
#   - dashboard_api.py has no input validation (security risk)
#   - analytics.js calls 12 API endpoints (high coupling)
#   - Test coverage: 0%

# Fix with AI assistance
> "Add input validation to dashboard_api.py with Pydantic"
> "Add unit tests for dashboard_api.py"

# Re-check
crisk check
# Output: LOW risk (ready to commit)

git add .
git commit -m "Add user analytics dashboard with validation and tests"
git push origin feature/user-dashboard

# Self-merge (solo project)
git checkout main
git merge feature/user-dashboard
git push origin main

# Deploy to production
./deploy.sh
```

**Time:** 15 minutes (vs 2-3 hours manual coding)
**Risk:** Reduced via CodeRisk (caught security issue + missing tests)

---

### Example 2: Startup Team with Vibe Coding (10 people)

```bash
# Product team requests new feature: "Slack notifications"
# Developer: Sarah, Full-stack engineer

# Monday 9 AM standup: assigned JIRA-456
git checkout main
git pull
git checkout -b sarah/JIRA-456-slack-integration

# Use Cursor to scaffold
> "Create Slack webhook integration that sends notifications on user signups"

# Cursor generates:
# - slack_client.py (100 lines)
# - webhook_handler.py (60 lines)
# - slack_config.py (30 lines)
# - tests/test_slack.py (80 lines)

# Manual tweaks (5 min)
# Add company-specific config

# Pre-commit check
crisk check
# Output: MEDIUM
#   - slack_client.py not handling rate limits
#   - webhook_handler.py has high coupling with user_service.py

# Fix rate limiting with AI
> "Add exponential backoff retry logic to slack_client.py"

crisk check
# Output: LOW - ready to commit

git add .
git commit -m "[JIRA-456] Add Slack notification integration"
git push origin sarah/JIRA-456-slack-integration

# Create PR
gh pr create --title "[JIRA-456] Slack notifications for user signups" --body "$(crisk check --format=pr-description)"

# PR description auto-populated with risk assessment:
# Risk Level: LOW
# Files Changed: 4
# Test Coverage: 75%
# No critical issues detected

# Tech lead reviews (15 min)
# Approves + merges

git checkout main
git pull
git branch -d sarah/JIRA-456-slack-integration
```

**Time:** 1 hour (vs 4-5 hours manual)
**Quality:** High (CodeRisk caught rate limiting issue)

---

### Example 3: Enterprise Developer (500-person company)

```bash
# Developer: Marcus, Senior Engineer at FinTech company
# Task: Add new compliance report feature

# Tuesday 10 AM
cd ~/work/fintech-monorepo
git checkout main
git pull origin main

# Create feature branch (strict naming convention)
git checkout -b users/marcus.jones/PLAT-2341-compliance-reports

# Use Claude Code (approved AI tool)
> "Create a compliance report generator that exports user audit logs to CSV with PII redaction"

# Claude generates complex code:
# - compliance/report_generator.py (200 lines)
# - compliance/pii_redactor.py (150 lines)
# - compliance/csv_exporter.py (100 lines)
# - tests/ (300 lines)
# Total: 4 files, 750 lines

# Pre-commit hook triggers automatically
crisk check --mode=enterprise --policy=strict

# Output: CRITICAL
# ❌ BLOCKING ISSUES:
#   - pii_redactor.py handles SSN but no encryption at rest
#   - report_generator.py missing audit log entry
#   - csv_exporter.py allows arbitrary file paths (security)
#   - Test coverage: 65% (policy requires 80%)

# Marcus reviews AI code more carefully
# Realizes these are serious compliance issues

# Fix manually (AI suggestions not trusted for security)
vim compliance/pii_redactor.py
# Add encryption layer

vim compliance/report_generator.py
# Add audit logging

vim compliance/csv_exporter.py
# Add path validation

# Add more tests manually
vim tests/test_compliance.py

# Re-run CodeRisk
crisk check --mode=enterprise --policy=strict
# Output: MEDIUM (better, but still issues)
#   - High coupling with user_service (8 dependencies)
#   - Test coverage: 78% (need 80%)

# Add 2 more test cases
vim tests/test_compliance.py

crisk check --mode=enterprise --policy=strict
# Output: LOW ✅
#   - All policies passed
#   - Test coverage: 82%
#   - Security checks passed

# Commit
git add compliance/ tests/
git commit -m "feat(compliance): Add PII-safe audit report generator [PLAT-2341]"

# Push triggers CI pipeline
git push origin users/marcus.jones/PLAT-2341-compliance-reports

# CI runs multiple checks:
# 1. Unit tests
# 2. Integration tests
# 3. CodeRisk scan (in CI)
# 4. SonarQube
# 5. Security scan (Snyk)

# Create PR (requires template)
gh pr create --template compliance-feature.md

# PR requires:
# - 2 engineer approvals
# - 1 security team approval
# - 1 compliance team approval

# CodeRisk posts automated comment:
# Risk Assessment: LOW ✅
# Compliance Policy: PASSED
# Security Checks: PASSED
# Audit Trail: https://coderisk.internal/reports/PLAT-2341

# Review process (2-3 days)
# Eventually merged to main
```

**Time:** 4 hours (vs 2-3 days manual)
**Quality:** Very High (caught 3 critical security issues)
**Compliance:** Full audit trail via CodeRisk

---

## Key Insights: Where CodeRisk Adds Most Value

### 1. **AI Velocity × Safety Multiplier**
- Vibe coding increases velocity 3-5x
- CodeRisk maintains safety without slowing velocity
- Result: Fast AND safe code

### 2. **Shift-Left Quality Gates**
- Traditional: Find issues in PR review (late, expensive)
- CodeRisk: Find issues pre-commit (early, cheap)
- Result: Faster PR cycles, less review burden

### 3. **Team Size Scaling**
- Solo: Personal code reviewer
- Small team: Automated quality gate
- Enterprise: Compliance + audit
- Result: Value at every scale

### 4. **Git Workflow Agnostic**
- Works with trunk-based development
- Works with GitFlow
- Works with stacked diffs
- Result: Fits any workflow

### 5. **Vibe Coding Accelerator**
- Doesn't slow down AI coding
- Guides AI toward safer patterns
- Catches AI mistakes early
- Result: Better AI code quality

---

## Adoption Recommendations by Team Type

### For Startups (1-20 people)
**Recommended Integration:**
```bash
# Add to package.json or Makefile
"scripts": {
  "precommit": "crisk check --quick",
  "pre-push": "crisk check"
}
```

**Adoption Strategy:**
1. Introduce via CLI (low friction)
2. Share impressive catches in Slack
3. Add to pre-commit hooks (optional)
4. Add to CI after 2-4 weeks

---

### For Growth Companies (20-200 people)
**Recommended Integration:**
```yaml
# Required CI check
- name: CodeRisk
  run: crisk check --fail-on high
```

**Adoption Strategy:**
1. Pilot with 2-3 teams
2. Measure impact (fewer production incidents)
3. Roll out to all teams
4. Enforce via branch protection

---

### For Enterprises (200+ people)
**Recommended Integration:**
```yaml
# Multi-stage pipeline
stages:
  - pre-commit-hook
  - ci-check
  - security-scan
  - compliance-report
```

**Adoption Strategy:**
1. Security/compliance team evaluates
2. Self-hosted deployment in VPC
3. Pilot with internal tools team
4. Expand to customer-facing services
5. Integrate with existing dashboards

---

### For Open Source Projects
**Recommended Integration:**
```yaml
# GitHub Action (free for OSS)
- uses: coderisk/action@v1
  with:
    mode: oss
    comment: true
```

**Adoption Strategy:**
1. Add as optional check (don't block PRs)
2. Use for maintainer visibility
3. Over time, educate contributors
4. Eventually make recommended (not required)

---

## Conclusion

CodeRisk integrates seamlessly into modern development workflows by:

1. **Meeting developers where they are** (pre-commit, CI, PR comments)
2. **Supporting both manual and AI coding** (vibe coding accelerator)
3. **Scaling from solo to enterprise** (same tool, different deployment)
4. **Working with any git pattern** (workflow agnostic)

The rise of vibe coding makes CodeRisk **more valuable**, not less—because AI-generated code needs intelligent review, and CodeRisk provides that at machine speed.

**Key Principle:** `crisk check` should feel as natural as `git status`—a reflexive habit, not a burdensome gate.

---

**Related Documentation:**
- [User Personas](user_personas.md) - Detailed user profiles (Ben, Clara)
- [Vision & Mission](vision_and_mission.md) - Strategic positioning as "pre-flight check"
- [Success Metrics](success_metrics.md) - How we measure workflow integration success
- [spec.md](../spec.md) - Technical requirements (FR-6 to FR-10: Multi-branch workflows)
