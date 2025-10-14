# CodeRisk Launch Setup - 3 Parallel Sessions

**Created:** 2025-10-13
**Status:** Ready to Execute
**Estimated Total Time:** 2-3 hours per session (can run in parallel)

---

## Overview

Based on the successful `crisk check` test run showing Phase 0 working without configuration, we've created a comprehensive launch setup with three parallel Claude Code sessions.

**Key Insight from Test:** Phase 0 (static analysis) successfully detected security risks without any API key or Docker setup! This is our biggest value prop - **it works immediately**.

---

## Three Parallel Sessions

### Session 1: Backend Packaging 📦
**Directory:** `coderisk-go`
**Prompt:** [dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md](dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md)
**Documentation:** [dev_docs/03-implementation/packaging_and_distribution.md](dev_docs/03-implementation/packaging_and_distribution.md)

**What it does:**
- Set up GoReleaser for multi-platform builds
- Create Homebrew formula (`brew install crisk`)
- Configure GitHub Actions for automated releases
- Build Docker image
- Create install script for universal setup
- Add `--version` flag with build info

**Deliverables:**
- `.goreleaser.yml`
- `.github/workflows/release.yml`
- `homebrew-coderisk` repository
- `Dockerfile`
- `install.sh` (improved)
- `CHANGELOG.md`

**Success:** `brew install rohankatakam/coderisk/crisk` works

---

### Session 2: Frontend Website 🌐
**Directory:** `coderisk-frontend`
**Prompt:** [dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md](dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md)
**Documentation:** [dev_docs/03-implementation/website_messaging.md](dev_docs/03-implementation/website_messaging.md)

**What it does:**
- Update homepage to emphasize "Open Source"
- Add MIT License badge
- Show tiered setup (Phase 0 → Phase 2 → Full)
- Create pricing page ($0 → $10 → $25 → $50/user/month)
- Create open source page (explains open core model)
- Update installation instructions (4 methods)

**Deliverables:**
- Updated `app/page.tsx` (homepage)
- New `app/pricing/page.tsx`
- New `app/open-source/page.tsx`
- Updated navigation and footer
- SEO meta tags
- Badge components

**Success:** Website clearly shows "Free CLI, optional cloud"

---

### Session 3: Configuration Management ⚙️
**Directory:** `coderisk-go`
**Prompt:** [dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md](dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md)
**Documentation:** [dev_docs/03-implementation/configuration_management.md](dev_docs/03-implementation/configuration_management.md)

**What it does:**
- Implement `crisk configure` (interactive setup wizard)
- Implement `crisk config` (get/set/show/reset)
- Implement `crisk doctor` (diagnostic health check)
- Add config file support (`~/.config/crisk/config.toml`)
- Improve messaging to show tiered UX
- Emphasize Phase 0 works immediately

**Deliverables:**
- `internal/config/` package
- `cmd/crisk/configure.go`
- `cmd/crisk/config.go`
- `cmd/crisk/doctor.go`
- Updated `cmd/crisk/check.go` (better messaging)
- Updated `README.md` (tiered instructions)

**Success:** Users understand Phase 0 works without setup, Phase 2 is optional

---

## Why Three Sessions?

**Based on your test output showing:**
1. ✅ **Phase 0 works without API key** - This is huge! We should emphasize it.
2. 💡 **Phase 2 requires OPENAI_API_KEY** - Optional but provides deeper analysis
3. 🐳 **Docker optional** - Only needed for full graph analysis

**Sessions 1 & 3 can run simultaneously in `coderisk-go`** because they touch different files:
- Session 1: `.goreleaser.yml`, `.github/workflows/`, `Dockerfile`, version in `main.go`
- Session 3: `internal/config/`, `cmd/crisk/configure.go`, `cmd/crisk/config.go`, messaging in `check.go`

**Session 2 runs in separate repo** (`coderisk-frontend`)

---

## Execution Order

### Option A: All in Parallel (Fastest - 2-3 hours)
```bash
# Terminal 1: Backend packaging
cd coderisk-go
# Paste BACKEND_PACKAGING_PROMPT.md

# Terminal 2: Frontend website
cd coderisk-frontend
# Paste FRONTEND_WEBSITE_PROMPT.md

# Terminal 3: Configuration management
cd coderisk-go
# Paste CONFIG_MANAGEMENT_PROMPT.md
```

### Option B: Sequential (Safer - 6-9 hours)
1. Session 1 (Backend Packaging) - 2-3 hours
2. Session 2 (Frontend Website) - 2-3 hours
3. Session 3 (Configuration Management) - 2-3 hours

### Option C: Backend First, Then Parallel (Recommended - 4-6 hours)
1. Session 1 (Backend Packaging) - 2-3 hours
   - Creates `install.sh` needed by frontend
2. Sessions 2 & 3 in parallel - 2-3 hours
   - Session 2: Frontend (uses install.sh in `public/`)
   - Session 3: Config management (no conflicts)

---

## Key Messaging from Test Run

**Your test showed:**
```
✅ Phase 0: Static analysis (0.2s)
   - Detected [Security] modification type
   - Set force_escalate=true
   - Assigned CRITICAL risk level

⚠️  Phase 2 unavailable: OPENAI_API_KEY not set

💡 Tip: Set OPENAI_API_KEY for Phase 2 deep investigation
```

**This is perfect!** It shows:
1. ✅ Phase 0 works immediately (detected security risk)
2. 💡 Phase 2 is optional (helpful tip, not error)
3. Clear value prop (Phase 2 = "deep investigation")

**All three sessions will emphasize this tiered approach:**
- **Session 1:** README shows "brew install && works immediately"
- **Session 2:** Website shows "Free CLI, optional cloud"
- **Session 3:** Messaging shows "Phase 0 ✅, Phase 2 optional 💡"

---

## Expected Outcomes

### After Session 1 (Packaging):
```bash
# Users can install in 30 seconds
brew tap rohankatakam/coderisk
brew install crisk

# And use immediately (no setup!)
cd my-repo
crisk check
# ✅ Phase 0 analysis complete
```

### After Session 2 (Website):
```
Homepage headline: "Open Source AI Code Risk Assessment"
Subheadline: "Free CLI, optional cloud platform"

Prominent: MIT License badge, GitHub stars

Installation:
  1. brew install crisk (30 sec) - Works now!
  2. Add API key (1 min) - Optional, for Phase 2
  3. Docker setup (10 min) - Optional, for full mode
```

### After Session 3 (Config):
```bash
# First-time user
crisk check
# ✅ Phase 0 complete (0.2s)
# 💡 Want deeper analysis? Run: crisk configure

# Easy setup
crisk configure
# Interactive wizard walks through Phase 2 setup

# Advanced users
crisk config set api.openai_key sk-...
crisk doctor  # Check what's working

# Result: Clear, tiered UX
```

---

## Documentation Created

**Strategy Documents (dev_docs/):**
1. [packaging_and_distribution.md](dev_docs/03-implementation/packaging_and_distribution.md) - 46 pages
2. [website_messaging.md](dev_docs/03-implementation/website_messaging.md) - 52 pages
3. [configuration_management.md](dev_docs/03-implementation/configuration_management.md) - 48 pages

**Claude Code Prompts (dev_docs/03-implementation/):**
1. [BACKEND_PACKAGING_PROMPT.md](dev_docs/03-implementation/BACKEND_PACKAGING_PROMPT.md) - Concise, actionable
2. [FRONTEND_WEBSITE_PROMPT.md](dev_docs/03-implementation/FRONTEND_WEBSITE_PROMPT.md) - Concise, actionable
3. [CONFIG_MANAGEMENT_PROMPT.md](dev_docs/03-implementation/CONFIG_MANAGEMENT_PROMPT.md) - Concise, actionable

**Root Files:**
- [LICENSE](LICENSE) - MIT License (already created)
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines (already created)
- [README.md](README.md) - Will be updated by Sessions 1 & 3

---

## Success Metrics

**After all three sessions:**

- [ ] ✅ **Installation:** `brew install crisk` works in 30 seconds
- [ ] ✅ **Zero config:** `crisk check` works immediately (Phase 0)
- [ ] ✅ **Optional Phase 2:** `crisk configure` sets up API key easily
- [ ] ✅ **Website clarity:** Homepage says "Open Source" front and center
- [ ] ✅ **Tiered messaging:** Users understand what's free vs optional vs paid
- [ ] ✅ **Open source badge:** MIT License badge visible on website
- [ ] ✅ **Installation methods:** 4 methods documented (Homebrew, script, Docker, direct)
- [ ] ✅ **Automated releases:** Tag push → binaries built → Homebrew updated
- [ ] ✅ **Configuration file:** `~/.config/crisk/config.toml` created by `crisk configure`
- [ ] ✅ **Diagnostic:** `crisk doctor` shows what's working vs missing

---

## Common Questions

**Q: Can Sessions 1 and 3 really run in parallel?**
A: Yes! They touch different files. Session 1 focuses on release infrastructure, Session 3 focuses on user-facing commands and config.

**Q: Do I need to run all three?**
A: Session 1 (Packaging) is highest priority - enables one-command install. Sessions 2 and 3 enhance UX but aren't blocking.

**Q: What if there are conflicts?**
A: Both sessions will update `README.md`. Recommendation: Run Session 1 first, then Sessions 2 & 3 in parallel. Merge README changes manually if needed (should be different sections).

**Q: How do I know what to paste to Claude Code?**
A: Copy the entire content of each prompt file:
- Session 1: `BACKEND_PACKAGING_PROMPT.md`
- Session 2: `FRONTEND_WEBSITE_PROMPT.md`
- Session 3: `CONFIG_MANAGEMENT_PROMPT.md`

**Q: What version should I tag for first release?**
A: Recommend `v0.1.0` for beta or `v1.0.0` for stable. Test with beta tag first.

---

## Next Steps After Sessions Complete

1. **Test installation flow:**
   ```bash
   brew tap rohankatakam/coderisk
   brew install crisk
   cd test-repo
   crisk check
   ```

2. **Tag first release:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0: Open Source Launch"
   git push origin v1.0.0
   ```

3. **Monitor release:**
   - GitHub Actions workflow runs
   - Binaries created in GitHub Releases
   - Homebrew formula updated
   - Docker image pushed

4. **Deploy website:**
   - Push to Vercel
   - Verify all links work
   - Test install script: `curl -fsSL https://coderisk.dev/install.sh | bash`

5. **Launch announcements:**
   - Hacker News: "Show HN: CodeRisk - Open Source AI Code Risk Assessment"
   - Twitter/X: Thread with demo GIF
   - Reddit: r/golang, r/programming
   - Product Hunt: Launch page

---

## Questions Before Starting?

**Ask yourself:**
1. Do I have Docker Hub credentials? (Needed for Session 1)
2. Do I want `v0.1.0-beta` or `v1.0.0` for first release? (Session 1)
3. Should install script support Windows/WSL? (Session 1)
4. Do we need separate docs site or link to GitHub README? (Session 2)
5. Should config file be TOML or YAML? (Session 3 - recommend TOML)

---

**Ready to launch! Pick your execution option and start sessions.** 🚀

**Recommended:** Option C (Backend first, then parallel) for safest, fastest path.
