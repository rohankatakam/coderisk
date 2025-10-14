# Agent Orchestration Guide: Running Multiple Claude Code Sessions

**Created:** 2025-10-13
**Purpose:** Guide for coordinating multiple Claude Code agents to work on CodeRisk in parallel
**Status:** Active - Ready to use

---

## Quick Start

You have 3 main workflows to choose from:

### Option 1: Sequential Testing → Implementation (Recommended)
**Use when:** You want to validate current functionality before making changes

```
Session 1 (Testing): Test omnara repo → Gather results
Session 2 (Backend): Implement packaging based on test results
Session 3 (Frontend): Update website with validated metrics
Session 4 (Config): Improve onboarding based on test insights
```

### Option 2: Parallel Implementation (Fast)
**Use when:** You're confident in current functionality and want speed

```
All 3 sessions run simultaneously:
- Session 1 (Backend): Packaging + GoReleaser
- Session 2 (Frontend): Website updates
- Session 3 (Config): Configuration management
```

### Option 3: Testing Only (Validation)
**Use when:** You just want to validate before launch

```
Single session: Comprehensive omnara testing
Result: TESTING_REPORT_OMNARA.md for review
```

---

## Available Prompts and Their Dependencies

### 1. Testing Prompt (Independent)
**File:** [TESTING_PROMPT_OMNARA.md](TESTING_PROMPT_OMNARA.md)
**Repository:** coderisk-go (main backend)
**Duration:** 2-3 hours
**Dependencies:** None
**Can run:** Independently, anytime

**What it does:**
- Tests all 10 modification types
- Measures false positive rate (<3% target)
- Benchmarks performance across repo sizes
- Tests edge cases and error handling

**When to use:**
- Before launch to validate claims
- After major changes to Phase 1/2
- When adding new modification types
- Before updating website metrics

**Output:** `TESTING_REPORT_OMNARA.md` with validation results

---

### 2. Backend Packaging Prompt (Independent)
**File:** [BACKEND_PACKAGING_PROMPT.md](dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md)
**Repository:** coderisk-go (main backend)
**Duration:** 2-3 hours
**Dependencies:** None
**Can run:** In parallel with Frontend and Config sessions

**What it does:**
- Sets up GoReleaser for multi-platform builds
- Creates Homebrew formula
- Configures GitHub Actions for releases
- Builds Docker image
- Injects version info

**When to use:**
- Ready to package for distribution
- Need automated release pipeline
- Want Homebrew installation

**Output:**
- `.goreleaser.yml`
- GitHub Actions workflow
- Homebrew formula
- Docker configuration

---

### 3. Frontend Website Prompt (Independent)
**File:** [FRONTEND_WEBSITE_PROMPT.md](dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md)
**Repository:** coderisk-frontend (Next.js app)
**Duration:** 1-2 hours
**Dependencies:** None
**Can run:** In parallel with Backend and Config sessions

**What it does:**
- Updates homepage with open source messaging
- Creates pricing page (Self-hosted vs Cloud)
- Creates open source page
- Updates installation instructions

**When to use:**
- Ready to launch website
- After finalizing pricing strategy
- After validating metrics from testing

**Output:**
- Updated `/app/page.tsx`
- New `/app/pricing/page.tsx`
- New `/app/open-source/page.tsx`

---

### 4. Config Management Prompt (Depends on Backend) - PROFESSIONAL TIER
**File:** [CONFIG_MANAGEMENT_PROMPT.md](dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md)
**Repository:** coderisk-go (main backend)
**Duration:** 4-5 hours (includes OS keychain integration)
**Dependencies:** Should run AFTER backend packaging if modifying CLI structure
**Can run:** In parallel with Frontend, but sequential with Backend

**What it does:**
- Implements `crisk configure` interactive wizard with OS keychain support
- Implements `crisk config` get/set commands with keychain integration
- Adds config file support (YAML via Viper)
- **Integrates go-keyring for OS-level secure storage**
- Improves API key setup experience (no plaintext storage)
- Implements secure credential management
- Improves error messages

**Security Features (Tier 3 - Best Practice):**
- ✅ API keys stored in OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- ✅ No plaintext storage in config files (optional plaintext for CI/CD)
- ✅ Multi-level precedence: env var > keychain > config file > .env
- ✅ Professional developer experience with security-first approach

**When to use:**
- After basic CLI structure is stable
- Want professional, secure credential management
- Need persistent configuration without security risks
- Current shell config editing is not developer-friendly

**Related Documentation:**
- [IMPROVED_API_KEY_SETUP.md](dev_docs/03-implementation/IMPROVED_API_KEY_SETUP.md) - Complete API key management strategy
- [KEYCHAIN_INTEGRATION_GUIDE.md](dev_docs/03-implementation/KEYCHAIN_INTEGRATION_GUIDE.md) - OS keychain implementation details

**Output:**
- `internal/config/keyring.go` (OS keychain interface)
- `cmd/crisk/configure.go` (interactive wizard with keychain support)
- `cmd/crisk/config.go` (get/set/show commands with keychain)
- `cmd/crisk/migrate.go` (migrate-to-keychain command)
- Updated `internal/llm/client.go` (uses config system with keychain fallback)
- Updated `install.sh` (offers keychain setup during installation)
- Enhanced error messages with security guidance
- Cross-platform keychain support (macOS, Windows, Linux)

---

## Recommended Workflows

### Workflow A: Pre-Launch Validation (Safest)

**Goal:** Validate everything works before public launch

**Steps:**

1. **Session 1: Testing (coderisk-go repo)**
   ```bash
   # Open Claude Code in coderisk-go
   # Provide this prompt:

   I need you to systematically test CodeRisk on the omnara repository.

   Context:
   - CodeRisk is already installed and working
   - Graph has been initialized (init-local completed)
   - Docker infrastructure is running
   - OpenAI API key is set

   Tasks:
   1. Read TESTING_PROMPT_OMNARA.md for full details
   2. Test all 10 modification types
   3. Measure false positive rate (target <3%)
   4. Benchmark performance
   5. Test edge cases
   6. Create TESTING_REPORT_OMNARA.md with results

   Please start with Task 1: Modification Type Coverage.
   ```

2. **Review test results** (30 minutes)
   - Read `TESTING_REPORT_OMNARA.md`
   - Verify <3% false positive rate achieved
   - Confirm all 10 modification types work
   - Check performance benchmarks (2-5 seconds)

3. **If tests pass → Proceed to parallel implementation**

4. **Sessions 2-4: Parallel Implementation**
   - Backend packaging (Session 2)
   - Frontend website (Session 3)
   - Config management (Session 4)

**Total time:** 2-3 hours testing + 2-3 hours implementation = 5-6 hours

---

### Workflow B: Fast Parallel Launch (Riskier but Faster)

**Goal:** Launch quickly, assume current functionality works

**Steps:**

1. **Launch 3 sessions simultaneously:**

   **Session 1 - Backend (coderisk-go):**
   ```bash
   I need you to set up packaging and distribution for CodeRisk CLI.

   Read these documents first:
   1. CORRECTED_LAUNCH_STRATEGY_V2.md - Understand requirements
   2. dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md - Your tasks
   3. dev_docs/03-implementation/packaging_and_distribution.md - Strategy

   Then implement:
   1. GoReleaser configuration
   2. Homebrew formula
   3. GitHub Actions workflow
   4. Docker image
   5. Version injection

   IMPORTANT: Both API key AND Docker required (not optional).
   Setup time: 17 minutes one-time per repo.
   Cost: $0.03-0.05/check (BYOK).
   ```

   **Session 2 - Frontend (/tmp/coderisk-frontend):**
   ```bash
   I need you to update the CodeRisk website with correct messaging.

   Read these documents first:
   1. coderisk-go/CORRECTED_LAUNCH_STRATEGY_V2.md - Requirements
   2. coderisk-go/dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md - Tasks
   3. coderisk-go/dev_docs/03-implementation/website_messaging.md - Content

   Then implement:
   1. Update homepage (app/page.tsx)
   2. Create pricing page (app/pricing/page.tsx)
   3. Create open source page (app/open-source/page.tsx)

   KEY MESSAGING:
   - NOT "free tier" - costs $3-5/month (LLM API)
   - NOT "works immediately" - 17-minute setup
   - Both API key AND Docker required
   - From $0.04/check (BYOK model)
   ```

   **Session 3 - Config (coderisk-go) - PROFESSIONAL SECURITY:**
   ```bash
   I need you to implement secure configuration management for CodeRisk with OS keychain integration.

   Read these documents first:
   1. CORRECTED_LAUNCH_STRATEGY_V2.md - Requirements
   2. dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md - Tasks
   3. dev_docs/03-implementation/IMPROVED_API_KEY_SETUP.md - Security strategy
   4. dev_docs/03-implementation/KEYCHAIN_INTEGRATION_GUIDE.md - Keychain implementation

   Then implement (Tier 3 - Professional Security):
   1. Add go-keyring dependency (github.com/zalando/go-keyring)
   2. internal/config/keyring.go (OS keychain interface)
   3. crisk configure command (interactive wizard with keychain support)
   4. crisk config command (get/set/show with keychain)
   5. crisk migrate-to-keychain command (for existing users)
   6. Update internal/llm/client.go to use config system with keychain
   7. Update install.sh to offer keychain setup
   8. Enhanced error messages with security guidance

   SECURITY REQUIREMENTS:
   - API keys stored in OS keychain (macOS/Windows/Linux)
   - No plaintext storage in config files (optional for CI/CD)
   - Precedence: env var > keychain > config file > .env
   - Professional developer experience

   CRITICAL: Both API key AND Docker required (not optional).
   ```

2. **Monitor all 3 sessions** (2-3 hours)
   - Sessions work independently
   - No conflicts (different files/repos)

3. **Review and merge** (30 minutes)
   - Test each implementation
   - Merge changes
   - Test integration

**Total time:** 2-3 hours + 30 min review = 2.5-3.5 hours

---

### Workflow C: Testing Only (Validation Before Changes)

**Goal:** Just validate current functionality

**Steps:**

1. **Single session: Testing (coderisk-go)**
   ```bash
   I need you to systematically test CodeRisk on the omnara repository.

   Please read TESTING_PROMPT_OMNARA.md and execute all tasks:
   1. Test all 10 modification types
   2. Measure false positive rate
   3. Benchmark performance
   4. Test edge cases
   5. Create comprehensive report

   Create TESTING_REPORT_OMNARA.md with findings.
   ```

2. **Review results** (30 minutes)
   - Validate <3% FP rate claim
   - Check performance (2-5 seconds)
   - Identify any issues

3. **Make fixes if needed** (variable)
   - Address any failing tests
   - Improve problematic areas

4. **Re-test** (if fixes were made)

**Total time:** 2-3 hours testing + fixes (if needed)

---

## How to Use Each Prompt

### Method 1: Direct Prompt Paste (Simplest)

1. Open Claude Code in the appropriate repository
2. Copy the entire prompt file contents
3. Paste into Claude Code
4. Claude reads referenced docs automatically

**Example:**
```bash
# 1. Open Claude Code in coderisk-go repo
cd ~/Documents/brain/coderisk-go
# Open Claude Code UI

# 2. Paste this:
"I need you to implement backend packaging for CodeRisk.

Read these documents first:
- CORRECTED_LAUNCH_STRATEGY_V2.md
- dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md
- dev_docs/03-implementation/packaging_and_distribution.md

Then follow all tasks in BACKEND_PACKAGING_PROMPT.md."
```

---

### Method 2: Reference File (Cleaner)

1. Tell Claude to read the prompt file
2. Claude reads and executes

**Example:**
```bash
# Open Claude Code
cd ~/Documents/brain/coderisk-go

# Send this message:
"Please read dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md
and execute all tasks listed there."
```

---

### Method 3: Slash Command (Future - If Implemented)

Create custom slash commands for each workflow:

```bash
# .claude/commands/test-omnara.md
Please read TESTING_PROMPT_OMNARA.md and execute all testing tasks.

# .claude/commands/package-backend.md
Please read dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md
and implement backend packaging.

# .claude/commands/update-website.md
Please read dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md
and update the website.
```

Then run:
```bash
/test-omnara
/package-backend
/update-website
```

---

## Managing Multiple Sessions

### Terminal Setup

**Option 1: Multiple Terminal Tabs**
```bash
# Tab 1: Backend session
cd ~/Documents/brain/coderisk-go
# Open Claude Code for backend

# Tab 2: Frontend session
cd /tmp/coderisk-frontend
# Open Claude Code for frontend

# Tab 3: Testing session (if running)
cd ~/Documents/brain/coderisk-go
# Open Claude Code for testing
```

**Option 2: tmux/screen**
```bash
# Create tmux session with 3 panes
tmux new-session -s coderisk

# Split panes
Ctrl+b %  # Vertical split
Ctrl+b "  # Horizontal split

# In each pane:
cd <repo>
# Open Claude Code
```

---

### Session Coordination

**When running parallel sessions:**

1. **Start all sessions at once**
   - Prevents one session from blocking others
   - Maximum parallelism

2. **Monitor progress**
   - Check each session every 15-20 minutes
   - Watch for blockers or questions

3. **Don't interrupt until complete**
   - Let each agent finish their prompt
   - Interrupting can cause context loss

4. **Review outputs sequentially**
   - Backend → Test it
   - Frontend → Test it
   - Config → Test it

---

### Handling Dependencies

**If Config session needs Backend changes:**

```bash
# Option 1: Run sequentially
1. Complete Backend session first
2. Test backend changes
3. Start Config session

# Option 2: Run in parallel, merge carefully
1. Both sessions complete
2. Merge backend changes first
3. Then merge config changes
4. Test integration
```

---

## Common Issues and Solutions

### Issue 1: Agent Gets Stuck

**Symptoms:**
- Agent stops responding
- Agent asks unclear questions
- Agent goes in circles

**Solutions:**
```bash
# Option 1: Clarify the goal
"Focus on Task 1 from the prompt: GoReleaser configuration.
Read the example in packaging_and_distribution.md."

# Option 2: Skip and move on
"Skip that task for now. Move to Task 2: Homebrew formula."

# Option 3: Restart with clearer prompt
"Let me rephrase. Please read BACKEND_PACKAGING_PROMPT.md
and implement ONLY the GoReleaser configuration (Task 1)."
```

---

### Issue 2: Agent Misunderstands Requirements

**Symptoms:**
- Agent suggests "free tier"
- Agent says "Docker optional"
- Agent says "Phase 0 works standalone"

**Solutions:**
```bash
"STOP. Please re-read CORRECTED_LAUNCH_STRATEGY_V2.md.

Key facts:
1. Both API key AND Docker required (not optional)
2. Setup: 17 minutes one-time per repo
3. Cost: $0.03-0.05/check (BYOK)
4. Phase 0 is pre-filter only, NOT standalone

Now retry your task with this understanding."
```

---

### Issue 3: Agent Reads Wrong Version of Docs

**Symptoms:**
- Agent references old incorrect information
- Agent cites configuration_management.md but wrong section

**Solutions:**
```bash
"Please re-read these files in order:
1. CORRECTED_LAUNCH_STRATEGY_V2.md (newest, most accurate)
2. dev_docs/03-implementation/configuration_management.md (recently corrected)

Ignore any older files or previous conversation context that contradicts these."
```

---

### Issue 4: Sessions Create Conflicts

**Symptoms:**
- Two sessions modify same file
- Git merge conflicts

**Solutions:**
```bash
# Prevention: Verify file separation
Backend session: .goreleaser.yml, .github/workflows/, Homebrew formula
Frontend session: /tmp/coderisk-frontend/app/*.tsx
Config session: internal/config/, cmd/crisk/configure.go

# If conflict occurs:
1. git status
2. Pick which change to keep (usually both are needed)
3. Manually merge
4. git add <file>
5. git commit
```

---

## Testing After Implementation

### After Backend Session

```bash
# 1. Test GoReleaser locally
goreleaser release --snapshot --clean

# 2. Test Homebrew formula (if created)
brew install --build-from-source ./Formula/crisk.rb

# 3. Test version injection
./dist/crisk_darwin_arm64/crisk --version
# Should show: CodeRisk v1.0.0 (or current version)

# 4. Test Docker image
docker build -t coderisk/crisk:test .
docker run --rm coderisk/crisk:test --version
```

---

### After Frontend Session

```bash
cd /tmp/coderisk-frontend

# 1. Build and run locally
npm run dev

# 2. Visit pages
open http://localhost:3000
open http://localhost:3000/pricing
open http://localhost:3000/open-source

# 3. Verify messaging
# - "From $0.04/check (BYOK)"
# - "17-minute setup"
# - No "free tier" language
# - Both API key AND Docker required
```

---

### After Config Session

```bash
# 1. Test crisk configure (interactive wizard)
crisk configure
# ✅ Should walk through 4-step setup:
#    - OpenAI API key input/validation
#    - Model selection (gpt-4o-mini vs gpt-4o)
#    - Budget limits (optional)
#    - Save confirmation
# ✅ Should validate API key format (starts with sk-)
# ✅ Should save to ~/.coderisk/config.yaml

# 2. Test crisk config commands
crisk config show
# ✅ Should display all config values (with masked API key)

crisk config get api.openai_key
# ✅ Should show full API key

crisk config set api.openai_key sk-test123
# ✅ Should update config file and confirm

crisk config set budget.monthly_limit 50.00
# ✅ Should update budget limit

# 3. Test config file location and format
cat ~/.coderisk/config.yaml
# ✅ Should have proper YAML format
# ✅ Should contain:
#    api:
#      openai_key: "sk-..."
#      openai_model: "gpt-4o-mini"
#    budget:
#      daily_limit: 2.00
#      monthly_limit: 60.00

# 4. Test OS keychain storage (PROFESSIONAL TIER)
crisk config set api.openai_key sk-test123 --use-keychain
# ✅ Should store in OS keychain (not plaintext)
# ✅ macOS: Keychain Access.app should show "CodeRisk" entry
# ✅ Windows: Credential Manager should show "CodeRisk" entry
# ✅ Linux: Secret Service should store credential

crisk config get api.openai_key --show-source
# ✅ Should show: "Source: keychain (secure)"

cat ~/.coderisk/config.yaml
# ✅ Should NOT contain plaintext API key
# ✅ Should have keychain flag: use_keychain: true

# 5. Test that crisk check uses keychain
unset OPENAI_API_KEY  # Clear environment variable
crisk check
# ✅ Should still work (reads from OS keychain)
# ✅ Should be completely secure (no plaintext storage)

# 6. Test precedence order (env var > keychain > config file > .env)
export OPENAI_API_KEY="sk-override"
crisk check
# ✅ Should use env var (highest precedence for CI/CD)

unset OPENAI_API_KEY
crisk check
# ✅ Should use keychain (second precedence for local dev)

# 7. Test migration command
echo "export OPENAI_API_KEY='sk-old'" >> ~/.zshrc
source ~/.zshrc
crisk migrate-to-keychain
# ✅ Should detect existing env var
# ✅ Should prompt to migrate to keychain
# ✅ Should offer to clean up shell config

# 8. Test .env file support (repo-specific config, plaintext for CI/CD)
cd /path/to/repo
echo "OPENAI_API_KEY=sk-repo-specific" > .env
crisk check
# ✅ Should read from .env file (plaintext OK for CI/CD)
# ✅ Should show warning: "Using plaintext .env file (consider keychain for local dev)"

# 9. Test keychain security
ps aux | grep crisk
# ✅ API key should NOT appear in process list
# ✅ API key should NOT be in environment variables

crisk config show
# ✅ Should mask API key: "sk-...abc123" (first 3, last 6)
# ✅ Should indicate source: "(stored in keychain)"

# 7. Test error messages
docker compose down
crisk check
# ✅ Should show clear error: "Cannot connect to graph database"
# ✅ Should show instructions: "Run: docker compose up -d"
```

---

### After Testing Session

```bash
# 1. Read the report
cat TESTING_REPORT_OMNARA.md

# 2. Verify critical metrics
# - False positive rate: <3%
# - All 10 modification types: PASS
# - Performance: 2-5 seconds per check
# - Edge cases: All handled

# 3. If issues found
# - Create GitHub issues
# - Prioritize fixes
# - Re-test after fixes
```

---

## Decision Matrix: Which Workflow to Use?

| Situation | Recommended Workflow | Duration | Risk |
|-----------|---------------------|----------|------|
| First time packaging CodeRisk | Workflow A (Test → Implement) | 5-6 hours | Low |
| Confident current functionality works | Workflow B (Parallel) | 2.5-3.5 hours | Medium |
| Just before public launch | Workflow A (Test first) | 5-6 hours | Low |
| After major Phase 1/2 changes | Workflow C (Testing only) | 2-3 hours | Low |
| Tight deadline, need speed | Workflow B (Parallel) | 2.5-3.5 hours | Medium |
| Want to validate <3% FP claim | Workflow C (Testing only) | 2-3 hours | Low |
| First time with multiple agents | Workflow A (Sequential) | 5-6 hours | Low |
| Experienced with Claude Code | Workflow B (Parallel) | 2.5-3.5 hours | Medium |

---

## Next Steps After All Sessions Complete

### 1. Integration Testing (1 hour)

```bash
# Full end-to-end test
brew install rohankatakam/coderisk/crisk  # From new Homebrew formula
crisk configure                            # From config session
docker compose up -d
crisk init-local
crisk check
crisk check --explain
```

### 2. Update Documentation (30 minutes)

```bash
# Update README.md with:
- New installation instructions
- Link to Homebrew formula
- Updated setup time (17 minutes)
- Cost information ($3-5/month)

# Update CHANGELOG.md with:
- New packaging support
- New configuration commands
- Website updates
```

### 3. Create Release (15 minutes)

```bash
# Tag release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions triggers:
- GoReleaser builds
- Homebrew formula updates
- Docker image pushes
```

### 4. Deploy Website (15 minutes)

```bash
cd /tmp/coderisk-frontend

# Deploy to Vercel
vercel --prod

# Or push to GitHub (if auto-deploy enabled)
git push origin main
```

### 5. Announce Launch (Variable)

```bash
# Update coderisk.dev
# Post on social media
# Submit to communities (HN, Reddit, etc.)
# Email announcement (if list exists)
```

---

## Summary: Quick Reference

**3 Main Prompts:**
1. **Testing:** [TESTING_PROMPT_OMNARA.md](TESTING_PROMPT_OMNARA.md) - Validate functionality
2. **Backend:** [BACKEND_PACKAGING_PROMPT.md](dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md) - Packaging & distribution
3. **Frontend:** [FRONTEND_WEBSITE_PROMPT.md](dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md) - Website updates
4. **Config:** [CONFIG_MANAGEMENT_PROMPT.md](dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md) - Configuration management

**3 Main Workflows:**
1. **Sequential (Safest):** Test → Backend → Frontend → Config (5-6 hours)
2. **Parallel (Fastest):** All 3 simultaneously (2.5-3.5 hours)
3. **Testing Only:** Just validation (2-3 hours)

**Key Documents:**
- [CORRECTED_LAUNCH_STRATEGY_V2.md](CORRECTED_LAUNCH_STRATEGY_V2.md) - Source of truth
- [packaging_and_distribution.md](dev_docs/03-implementation/packaging_and_distribution.md) - Strategy
- [website_messaging.md](dev_docs/03-implementation/website_messaging.md) - Content
- [configuration_management.md](dev_docs/03-implementation/configuration_management.md) - Config strategy

**Critical Facts (Repeat in every session):**
- Both API key AND Docker required (NOT optional)
- Setup: 17 minutes one-time per repo
- Cost: $0.03-0.05/check (~$3-5/month for 100 checks)
- Phase 0: Pre-filter only, NOT standalone

---

**Ready to start?** Choose a workflow and launch your session(s)!
