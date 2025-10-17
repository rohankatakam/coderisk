# CodeRisk: Professional Security Tier Implementation

**Date:** 2025-10-13
**Status:** Documentation Complete - Ready for Implementation
**Approach:** Tier 3 - OS Keychain Integration
**Estimated Time:** 4-5 hours implementation + testing

---

## Executive Summary

Following successful validation testing on the omnara repository (**7/7 tests passed**), a critical UX issue was identified: **OPENAI_API_KEY setup is too difficult for developers**.

**Decision:** Implement **Professional Security Tier (Tier 3)** using OS keychain integration for maximum security and best-in-class developer experience.

**User Requirement:** *"Package everything up securely and professionally and the right way with no shortcuts and best developer experience in mind."*

---

## Testing Success ✅

### Validation Results (omnara repository)

**Tests Passed:** 7/7 (100%)
**False Positive Rate:** 0%
**Performance:** <0.2s for low-risk changes

| Test | Result | Duration | Key Finding |
|------|--------|----------|-------------|
| Documentation-only changes | ✅ PASSED | 0.106s | Skip analysis=true, LOW risk |
| Configuration changes | ✅ PASSED | 16.8s | Force escalate=true, HIGH→MINIMAL |
| Structural refactoring | ✅ PASSED | 0.129s | LOW risk, no logic change |
| Empty diff | ✅ PASSED | 0.041s | Graceful handling |
| Missing API key | ✅ PASSED | N/A | Clear error with instructions |
| Neo4j not running | ✅ PASSED | N/A | Clear error message |

**Key Validation:**
- ✅ Phase 0 pre-analysis working
- ✅ Modification type detection operational (Documentation, Configuration, Structural)
- ✅ Force escalation for config changes working
- ✅ Skip analysis for docs-only changes working
- ✅ Co-change pattern detection working
- ✅ Error handling graceful and descriptive

**Reference:** See TESTING_REPORT_OMNARA.md for complete details

---

## Problem Identified ❌

### UX Issue: Difficult API Key Setup

**Root Cause Analysis:**

1. **CodeRisk has sophisticated config system** at `internal/config/config.go`
   - Uses Viper v1.18.2 for configuration management
   - Supports .env files, YAML config, environment variables
   - Has godotenv v1.5.1 for .env support

2. **But LLM client bypasses this system entirely**
   - Located at `internal/llm/client.go:51`
   - Reads directly from `os.Getenv("OPENAI_API_KEY")`
   - Does not use the sophisticated config infrastructure

3. **Current user experience:**
   - Manual editing of shell config files (.zshrc, .bashrc)
   - API key stored in plaintext
   - No validation or error checking
   - No secure storage option
   - No interactive setup

**Quote from user:** *"It seems like this OPENAI_KEY setting is difficult to set up right now"*

---

## Solution: Three-Tier Evaluation

### Options Considered

| Tier | Approach | Time | Security | DX | Decision |
|------|----------|------|----------|----|----------|
| **Tier 1** | Use existing config system | 30 min | Medium | Good | ❌ Rejected |
| **Tier 2** | Add interactive wizard | 2-3 hrs | Medium | Great | ❌ Rejected |
| **Tier 3** | OS Keychain Integration | 4-5 hrs | **Excellent** | **Best** | ✅ **SELECTED** |

### Why Tier 3? 🔒

**Professional Security Benefits:**
- 🔒 **Maximum Security**: API keys stored in OS keychain (never plaintext by default)
- 🎯 **Best Developer Experience**: One-command setup, automatic retrieval
- 🌍 **Cross-Platform**: macOS Keychain, Windows Credential Manager, Linux Secret Service
- ⚙️ **Flexible**: Multi-level precedence (env var > keychain > config > .env)
- 🚀 **Production Ready**: Industry best practice for credential management
- 🔧 **CI/CD Compatible**: Environment variable precedence (no keychain required)

**Technology:**
- Library: `github.com/zalando/go-keyring v1.2.4`
- Cross-platform credential storage
- Graceful fallback to config file if keychain unavailable

---

## Documentation Created/Updated

### 1. KEYCHAIN_INTEGRATION_GUIDE.md ✨ NEW

**Purpose:** Complete implementation guide for OS keychain integration
**Lines:** 700+

**Structure:**
- **Step 0:** Overview and architecture
- **Step 1:** Add go-keyring dependency
- **Step 2:** Implement `internal/config/keyring.go`
  - KeyringManager interface
  - SaveAPIKey, GetAPIKey, DeleteAPIKey methods
  - IsAvailable check
  - GetAPIKeySource for debugging
- **Step 3:** Update config loading with precedence
- **Step 4:** Implement `crisk configure` with keychain wizard
- **Step 5:** Implement `crisk config` commands with --use-keychain flag
- **Step 6:** Implement `crisk migrate-to-keychain` command
- **Step 7:** Update LLM client to use config system
- **Step 8:** Update install.sh with interactive wizard

**Key Code Interfaces:**
```go
type KeyringManager struct {
    service string  // "CodeRisk"
    user    string  // "default"
}

// Core methods
func (km *KeyringManager) SaveAPIKey(apiKey string) error
func (km *KeyringManager) GetAPIKey() (string, error)
func (km *KeyringManager) DeleteAPIKey() error
func (km *KeyringManager) IsAvailable() bool
func (km *KeyringManager) GetAPIKeySource(cfg *Config) KeySourceInfo

// Utility
func MaskAPIKey(apiKey string) string  // sk-...abc → sk-...***abc
```

**Precedence Order:**
1. Environment variable (`OPENAI_API_KEY`)
2. **OS Keychain** ← New primary source
3. Config file (`~/.coderisk/config.yaml`)
4. .env files (`.env`, `.env.local`)

---

### 2. AGENT_ORCHESTRATION_GUIDE.md 📝 UPDATED

**Changes:**
- **Line 120-161:** Updated Session 2 → "PROFESSIONAL SECURITY TIER"
  - Duration: 4-5 hours (includes keychain)
  - Added security features list
- **Line 263-290:** Updated Session 3 prompt with keychain requirements
- **Line 656-708:** Added comprehensive keychain testing section

**New Testing Requirements:**
```bash
# Test keychain storage
crisk config set api.openai_key sk-test123 --use-keychain
# macOS: Verify in Keychain Access.app

# Test precedence
export OPENAI_API_KEY=sk-env-override
crisk config show --debug  # Should show: source=env_var

# Test migration
crisk migrate-to-keychain
```

---

### 3. CONFIG_MANAGEMENT_PROMPT.md 📝 UPDATED

**Changes:**
- **Line 1-11:** Title → "PROFESSIONAL SECURITY TIER (4-5 hours)"
- **Line 39-60:** Updated objectives with keychain features
- **Line 66-148:** Added Task 0 (dependencies) and Task 1 (extend config)
- **Line 150-371:** Updated all tasks with keychain integration

**Critical Architectural Note:**
> `internal/config/config.go` **ALREADY EXISTS** with sophisticated Viper setup.
> DO NOT rewrite the entire config system.
> ADD: `internal/config/keyring.go` (new file)
> UPDATE: `applyEnvOverrides()` function in existing config.go

**Key Implementation Update:**
```go
// Add to internal/config/config.go
func (cfg *Config) applyEnvOverrides() {
    // Check environment variable first (highest precedence)
    if key := os.Getenv("OPENAI_API_KEY"); key != "" {
        cfg.API.OpenAIKey = key
        return
    }

    // NEW: Check keychain if env var not set
    km := NewKeyringManager()
    if km.IsAvailable() {
        if keychainKey, err := km.GetAPIKey(); err == nil && keychainKey != "" {
            cfg.API.OpenAIKey = keychainKey
            return
        }
    }

    // Fall back to config file / .env (existing logic)
}
```

---

### 4. TESTING_REPORT_OMNARA.md 📊 CREATED

**Purpose:** Complete validation report from testing phase
**Lines:** 346
**Status:** All tests passed

**Key Metrics:**
- 7/7 tests passed (100%)
- 0% false positive rate
- <0.2s for non-risky changes
- ~17s for HIGH risk with LLM investigation

**Tested Modification Types:**
- ✅ Documentation (skip analysis)
- ✅ Configuration (force escalate)
- ✅ Structural/Refactoring (low risk)

**Edge Cases Validated:**
- ✅ Empty diff (graceful handling)
- ✅ Missing API key (clear instructions)
- ✅ Neo4j not running (clear error)

---

### 5. IMPROVED_API_KEY_SETUP.md 📄 CREATED

**Purpose:** Original problem analysis and three-tier solution design
**Content:**
- Problem statement and root cause
- Three-tier solution comparison
- Full code examples for each tier
- Migration strategy

---

## Architecture Changes

### Before (Current State) ❌

```
User → install.sh → Manually edit .zshrc/.bashrc
                  → export OPENAI_API_KEY=sk-...
                  → Plaintext in shell config

crisk check → llm.NewClient()
           → os.Getenv("OPENAI_API_KEY")
           → ❌ Bypasses config system
```

**Problems:**
- Manual setup required
- Plaintext storage
- No validation
- Bypasses existing config infrastructure

---

### After (Professional Security Tier) ✅

```
User → install.sh → Interactive wizard
                  → crisk configure
                  → API key saved to OS keychain
                  → ✅ Encrypted by OS

crisk check → config.Load()
           → applyEnvOverrides()
           → Precedence check:
              1. os.Getenv("OPENAI_API_KEY")
              2. ✅ Keyring.Get() ← New
              3. config.yaml
              4. .env files
           → llm.NewClient(ctx, cfg)
           → ✅ Uses cfg.API.OpenAIKey
```

**Benefits:**
- One-command setup
- OS-level encryption
- Automatic validation
- Uses existing config system
- CI/CD compatible (env var precedence)

---

## Implementation Checklist

### Phase 1: Dependencies and Core Interface (1 hour)
- [ ] Add go-keyring to go.mod: `go get github.com/zalando/go-keyring@v1.2.4`
- [ ] Create `internal/config/keyring.go` with KeyringManager
  - [ ] SaveAPIKey method
  - [ ] GetAPIKey method
  - [ ] DeleteAPIKey method
  - [ ] IsAvailable method
  - [ ] GetAPIKeySource method
- [ ] Add unit tests for keyring operations

### Phase 2: Config System Integration (1 hour)
- [ ] Update `internal/config/config.go`
  - [ ] Add `UseKeychain bool` to APIConfig struct
  - [ ] Update `applyEnvOverrides()` to check keychain
  - [ ] Add `GetAPIKeySource()` debug method
- [ ] Update `internal/llm/client.go`
  - [ ] Change signature: `NewClient(ctx context.Context, cfg *config.Config)`
  - [ ] Read from `cfg.API.OpenAIKey` instead of `os.Getenv()`
- [ ] Update all callers of `llm.NewClient()` to pass config

### Phase 3: CLI Commands (1.5 hours)
- [ ] Implement `cmd/crisk/configure.go` with keychain wizard
  - [ ] Interactive prompts for API key
  - [ ] Automatic keychain storage
  - [ ] Validation of API key format
- [ ] Implement `cmd/crisk/config.go` commands
  - [ ] `crisk config get api.openai_key`
  - [ ] `crisk config set api.openai_key <value> --use-keychain`
  - [ ] `crisk config show --debug` (shows source)
  - [ ] `crisk config delete api.openai_key --from-keychain`
- [ ] Implement `cmd/crisk/migrate.go`
  - [ ] `crisk migrate-to-keychain`
  - [ ] Finds API key in .env or config file
  - [ ] Moves to keychain securely
  - [ ] Removes from plaintext

### Phase 4: Install Script Enhancement (0.5 hours)
- [ ] Update `install.sh`
  - [ ] Add interactive wizard option (default)
  - [ ] Add quick setup option (config file)
  - [ ] Add skip option (configure later)
  - [ ] Show keychain availability status
  - [ ] Clear instructions for each option

### Phase 5: Testing and Validation (1 hour)
- [ ] Unit tests
  - [ ] KeyringManager methods
  - [ ] Config loading precedence
  - [ ] Error handling
- [ ] Integration tests
  - [ ] Config loading from keychain
  - [ ] Precedence order validation
  - [ ] Migration tool
- [ ] E2E tests
  - [ ] `crisk configure` → keychain → `crisk check` works
  - [ ] Environment variable override
  - [ ] Migration from .env to keychain
- [ ] Cross-platform validation
  - [ ] macOS: Verify Keychain Access.app entry
  - [ ] Linux: Test with/without libsecret
  - [ ] Windows: Manual verification (optional)

---

## Files to Create/Modify

### New Files
1. `internal/config/keyring.go` - KeyringManager implementation
2. `cmd/crisk/configure.go` - Interactive setup wizard (or update existing)
3. `cmd/crisk/config.go` - Config management commands
4. `cmd/crisk/migrate.go` - Migration tool
5. Unit tests for all new files

### Files to Modify
1. `go.mod` - Add go-keyring dependency
2. `internal/config/config.go` - Update applyEnvOverrides()
3. `internal/llm/client.go` - Update NewClient signature
4. All callers of `llm.NewClient()` - Pass config parameter
5. `install.sh` - Add interactive wizard
6. `README.md` - Update setup instructions (after implementation)

---

## Security Considerations

### What Gets Stored in Keychain
✅ **OPENAI_API_KEY** (sensitive, never plaintext by default)
✅ Future: Other API keys (Anthropic, Azure, etc.)

### What Stays in Config File
✅ Model names (gpt-4o-mini, etc.)
✅ Neo4j connection details
✅ PostgreSQL connection
✅ Non-sensitive settings (log level, timeouts, etc.)

### Precedence Reasoning
1. **Environment Variable** (highest) - CI/CD override, emergency use
2. **OS Keychain** - Default for local development (secure)
3. **Config File** - Fallback (with warning about plaintext)
4. **.env Files** - Legacy support (with migration prompt)

### CI/CD Compatibility
- Environment variable support maintained (highest precedence)
- No keychain dependency in automated environments
- Clear documentation for CI/CD setup
- No breaking changes to existing workflows

---

## Cross-Platform Support

### macOS ✅
**Storage:** Keychain Services API
**Library:** go-keyring native support
**Verification:** Keychain Access.app → "CodeRisk" entry
**Status:** Full support

### Linux ✅
**Storage:** freedesktop.org Secret Service API
**Requirement:** libsecret-1-dev package
**Library:** go-keyring native support
**Fallback:** Config file if libsecret not installed
**Status:** Full support with graceful degradation

### Windows ⚠️
**Storage:** Windows Credential Manager
**Library:** go-keyring native support
**Verification:** Manual (Credential Manager UI)
**Status:** Supported but optional for testing

---

## Testing Strategy

### Unit Tests
```go
func TestKeyringManager_SaveAPIKey(t *testing.T)
func TestKeyringManager_GetAPIKey(t *testing.T)
func TestKeyringManager_DeleteAPIKey(t *testing.T)
func TestKeyringManager_IsAvailable(t *testing.T)
func TestConfig_ApplyEnvOverrides_Precedence(t *testing.T)
```

### Integration Tests
```bash
# Test keychain storage
./crisk config set api.openai_key sk-test123 --use-keychain
./crisk config get api.openai_key  # Should return: sk-...***123

# Test precedence
export OPENAI_API_KEY=sk-env-override
./crisk config show --debug
# Expected: source=env_var, value=sk-...***ide

# Test migration
echo "OPENAI_API_KEY=sk-dotenv123" > .env
./crisk migrate-to-keychain
# Should move key to keychain, remove from .env
```

### E2E Tests
```bash
# Clean slate
./crisk config delete api.openai_key --from-keychain
unset OPENAI_API_KEY

# Interactive configure
./crisk configure
# Wizard should guide through setup

# Verify crisk check works
./crisk check
# Phase 2 should work (API key retrieved from keychain)
```

### Cross-Platform Tests
```bash
# macOS
security find-generic-password -s CodeRisk -a default
# Should show encrypted entry

# Linux (with libsecret)
secret-tool lookup service CodeRisk username default
# Should return API key

# Linux (without libsecret)
./crisk config set api.openai_key sk-test123 --use-keychain
# Should gracefully fall back to config file with warning
```

---

## Success Criteria

### Functional Requirements ✅
- [ ] `crisk configure` sets up API key in one command
- [ ] API key stored in OS keychain (never plaintext by default)
- [ ] `crisk check` retrieves key automatically from keychain
- [ ] `crisk config show --debug` displays source (env/keychain/config/.env)
- [ ] `crisk migrate-to-keychain` moves existing keys securely
- [ ] Precedence order working: env > keychain > config > .env
- [ ] CI/CD environments work with env var (no keychain required)
- [ ] Graceful fallback if keychain unavailable

### Developer Experience ✅
- [ ] One-command setup: `./install.sh` → interactive wizard
- [ ] No manual .zshrc/.bashrc editing required
- [ ] Clear error messages if keychain unavailable
- [ ] Easy migration path from existing .env setup
- [ ] Works on macOS, Linux (with libsecret), Windows
- [ ] `crisk doctor` shows clear status of all components

### Security ✅
- [ ] API keys never stored in plaintext by default
- [ ] OS-level encryption (Keychain/Credential Manager/Secret Service)
- [ ] Optional plaintext for CI/CD (explicit opt-in with warning)
- [ ] Clear warnings when using plaintext storage
- [ ] Migration tool removes plaintext after moving to keychain

---

## Timeline Estimate

| Phase | Tasks | Time | Cumulative |
|-------|-------|------|------------|
| Phase 1 | Dependencies + KeyringManager | 1 hour | 1 hour |
| Phase 2 | Config integration + LLM client | 1 hour | 2 hours |
| Phase 3 | CLI commands | 1.5 hours | 3.5 hours |
| Phase 4 | Install script | 0.5 hours | 4 hours |
| Phase 5 | Testing | 1 hour | **5 hours** |

**Total:** 4-5 hours (with buffer)

---

## Risk Assessment

### Implementation Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| go-keyring platform issues | Low | Medium | Fallback to config file, clear error messages |
| Linux libsecret dependency | Medium | Low | Graceful degradation, document requirement |
| Breaking existing .env users | Low | Medium | Auto-migration tool, backward compatibility |
| CI/CD environments break | Very Low | High | Env var precedence maintained (highest) |
| Keychain permission issues | Low | Medium | Clear error messages, fallback option |

### Timeline Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Implementation >5 hours | Low | Low | Well-documented, clear scope |
| Testing uncovers edge cases | Medium | Medium | Buffer time built into estimate |
| Cross-platform issues | Medium | Medium | Focus macOS first, Linux second |

**Overall Risk:** **LOW** - Well-defined scope, clear requirements, existing config system to build on.

---

## Next Steps

### Immediate (After Approval)
1. **Begin Phase 1:** Add go-keyring dependency
2. **Implement KeyringManager:** Create `internal/config/keyring.go`
3. **Write unit tests:** Validate keyring operations

### Short-term (Within 5 hours)
4. **Update config system:** Modify `applyEnvOverrides()` in config.go
5. **Update LLM client:** Change NewClient signature
6. **Implement CLI commands:** configure, config, migrate
7. **Update install.sh:** Add interactive wizard

### Validation (1 hour)
8. **Run comprehensive tests:** Unit, integration, E2E
9. **Validate precedence:** Test env > keychain > config > .env
10. **Test migration:** Verify .env → keychain works
11. **Cross-platform check:** macOS primary, Linux secondary

### Documentation (After Implementation)
12. **Update README.md:** New setup instructions
13. **Update CONTRIBUTING.md:** Config system architecture
14. **Create troubleshooting guide:** Keychain issues

---

## Workstream Coordination

### Backend (coderisk-go) ✅
**Status:** Ready for implementation
**Session:** CONFIG_MANAGEMENT_PROMPT.md (Professional Security Tier)
**Time:** 4-5 hours
**Files:** keyring.go (new), config.go (update), llm/client.go (update), CLI commands (new)

### Frontend (/tmp/coderisk-frontend) ℹ️
**Status:** No changes needed for keychain implementation
**Note:** API key setup is backend-only concern
**Future:** May update website to mention secure keychain storage

### Config Management ✅
**Status:** Architecture complete, ready for implementation
**Approach:** Professional security tier with OS keychain
**Benefits:** Maximum security + best developer experience

---

## References

### Internal Documentation
- **TESTING_REPORT_OMNARA.md** - Validation testing results (7/7 passed)
- **KEYCHAIN_INTEGRATION_GUIDE.md** - Complete implementation guide (700+ lines)
- **AGENT_ORCHESTRATION_GUIDE.md** - Updated with keychain testing
- **CONFIG_MANAGEMENT_PROMPT.md** - Professional security tier prompt
- **IMPROVED_API_KEY_SETUP.md** - Problem analysis and solution design

### External Resources
- **go-keyring:** https://github.com/zalando/go-keyring
- **Viper:** https://github.com/spf13/viper (already using v1.18.2)
- **godotenv:** https://github.com/joho/godotenv (already using v1.5.1)

### Key Files in Codebase
- `internal/config/config.go` - **EXISTS** (sophisticated Viper setup)
- `internal/llm/client.go:51` - **NEEDS UPDATE** (remove os.Getenv)
- `install.sh` - **NEEDS UPDATE** (add interactive wizard)

---

## Conclusion

CodeRisk has been successfully validated with **100% test success rate** on the omnara repository. The professional security tier approach with OS keychain integration will provide:

1. ✅ **Maximum Security** - API keys encrypted in OS keychain
2. ✅ **Best Developer Experience** - One-command setup, automatic retrieval
3. ✅ **Production Ready** - Industry best practice, CI/CD compatible
4. ✅ **Cross-Platform** - macOS, Linux, Windows support
5. ✅ **Graceful Degradation** - Fallback to config file if keychain unavailable

**Implementation Status:** Documentation complete, ready to begin
**Estimated Time:** 4-5 hours
**Next Step:** Begin Phase 1 (add go-keyring dependency)

---

**Generated:** 2025-10-13 by Claude Code
**User Decision:** Tier 3 - Professional Security Tier
**Quote:** *"Package everything up securely and professionally and the right way with no shortcuts and best developer experience in mind."*

---

## Quick Start (For Implementation)

```bash
# Step 1: Add dependency
go get github.com/zalando/go-keyring@v1.2.4

# Step 2: Create keyring manager
touch internal/config/keyring.go
# Implement KeyringManager (see KEYCHAIN_INTEGRATION_GUIDE.md)

# Step 3: Update config system
# Edit internal/config/config.go → applyEnvOverrides()

# Step 4: Update LLM client
# Edit internal/llm/client.go → NewClient(ctx, cfg)

# Step 5: Implement CLI commands
touch cmd/crisk/config.go
touch cmd/crisk/migrate.go
# Update cmd/crisk/configure.go

# Step 6: Test
go test ./internal/config/...
./crisk configure
./crisk check

# Success!
```

**Ready to implement. All documentation in place. Let's build it!** 🚀
