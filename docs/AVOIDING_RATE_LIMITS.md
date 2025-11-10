# How to Avoid Rate Limits with crisk check

Quick reference guide for developers and users.

---

## TL;DR

**Problem**: Running `crisk check` multiple times quickly hits Gemini API rate limits (429 errors)

**Solution**: Built-in retry logic now handles this automatically, but here's how to avoid them entirely:

```bash
# ✅ Good: Space out checks by 30-60 seconds
crisk check file1.ts
sleep 60
crisk check file2.ts

# ❌ Bad: Rapid-fire checks
for file in *.ts; do crisk check $file --explain; done  # Will hit rate limits!
```

---

## Quick Fixes

### 1. **Use Phase 1 Only (No LLM, No Rate Limits)**

```bash
# Fast metrics only, skips Phase 2 investigation
crisk check <file>
```

**Use when**:
- Iterating on code changes
- Running in CI/CD pipelines
- Checking multiple files quickly

### 2. **Enable Phase 2 Only When Needed**

```bash
# Deep investigation with LLM (slower, uses quota)
export PHASE2_ENABLED=true
crisk check <file> --explain
```

**Use when**:
- Reviewing critical pull requests
- Investigating high-risk files
- Need detailed explanations

### 3. **Batch Check with Delays**

```bash
# Check multiple files with automatic spacing
for file in src/**/*.ts; do
  crisk check "$file"
  sleep 10  # 10 second pause between checks
done
```

---

## Rate Limit Tiers

### Free Tier (Google AI Studio)
- **RPM**: 15 requests/minute
- **TPM**: 1M tokens/minute
- **RPD**: 1,500 requests/day

**Recommendation**: Max 1 check every 4-5 seconds

### Paid Tier (Gemini API)
- **RPM**: 2,000 requests/minute
- **TPM**: 4M tokens/minute
- **RPD**: Unlimited

**Recommendation**: Max 30 checks per minute (2 per second)

---

## Best Practices by Use Case

### During Development

```bash
# Option 1: Phase 1 only for rapid iteration
crisk check src/components/Button.tsx

# Option 2: Phase 2 for final review
export PHASE2_ENABLED=true
crisk check src/components/Button.tsx --explain
```

**Typical workflow**:
1. Make code changes
2. Run Phase 1 check (instant feedback)
3. Iterate
4. Final check with Phase 2 before commit

### In CI/CD Pipelines

```yaml
# GitHub Actions example
- name: Risk check changed files
  run: |
    # Get changed files
    CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD | grep -E '\.(ts|tsx|js|jsx)$')

    # Check each file with Phase 1 only (fast)
    for file in $CHANGED_FILES; do
      crisk check "$file" --quiet || true
    done
```

**Tips**:
- Use `--quiet` mode for concise output
- Don't enable Phase 2 (too slow for CI)
- Set timeout to 2-3 minutes max

### For Pre-commit Hooks

```bash
# .git/hooks/pre-commit
#!/bin/bash

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(ts|tsx)$')

# Check only modified files
for file in $STAGED_FILES; do
  echo "Checking $file..."
  crisk check "$file" --pre-commit || exit 1
done
```

**Key points**:
- Only check staged files (not entire repo)
- Use `--pre-commit` flag for faster checks
- Exit on first high-risk file

### For Production Monitoring

```bash
# Daily risk assessment
#!/bin/bash

# Check high-traffic files only
HIGH_TRAFFIC_FILES=(
  "src/api/users.ts"
  "src/api/payments.ts"
  "src/middleware/auth.ts"
)

for file in "${HIGH_TRAFFIC_FILES[@]}"; do
  echo "Checking $file..."
  PHASE2_ENABLED=true crisk check "$file" --explain > "/tmp/risk_reports/$(basename $file).txt"
  sleep 30  # 30 second spacing
done
```

---

## Understanding Retry Behavior

### What Happens When You Hit a Rate Limit

1. **First attempt**: Request sent → 429 error
2. **Wait 2 seconds**: Exponential backoff delay
3. **Second attempt**: Request sent → Success or another 429
4. **Wait 4 seconds**: If still rate-limited
5. **Third attempt**: Request sent → Success or another 429
6. **Wait 8 seconds**: If still rate-limited
7. **Fourth attempt**: Final try → Success or give up

**Total retry time**: Up to 14 seconds

### Log Messages

```bash
# When retrying
WARN: rate limit encountered, retrying attempt=1 delay_seconds=2

# When successful after retry
INFO: request succeeded after retry attempt=2

# When all retries exhausted
ERROR: rate limited after 3 retries (waited 14s total)
```

---

## Advanced: Rate Limit Avoidance Strategies

### 1. **Caching Investigation Results**

```bash
# Check if file changed since last investigation
FILE_HASH=$(git hash-object "$FILE")
CACHE_KEY="crisk_${FILE_HASH}"

if [ -f "/tmp/${CACHE_KEY}" ]; then
  echo "Using cached result"
  cat "/tmp/${CACHE_KEY}"
else
  crisk check "$FILE" --explain | tee "/tmp/${CACHE_KEY}"
fi
```

### 2. **Priority-Based Checking**

```bash
# Check critical files first, with longest delays
CRITICAL_FILES=("auth.ts" "payment.ts")
NORMAL_FILES=("utils.ts" "helpers.ts")

# Critical files: 60s spacing
for file in "${CRITICAL_FILES[@]}"; do
  PHASE2_ENABLED=true crisk check "$file" --explain
  sleep 60
done

# Normal files: Phase 1 only, 5s spacing
for file in "${NORMAL_FILES[@]}"; do
  crisk check "$file"
  sleep 5
done
```

### 3. **Distributed Checking with Multiple Keys**

```bash
# Round-robin across multiple API keys
API_KEYS=(
  "key1"
  "key2"
  "key3"
)

index=0
for file in src/**/*.ts; do
  export GEMINI_API_KEY="${API_KEYS[$index]}"
  crisk check "$file"

  # Rotate to next key
  index=$(( (index + 1) % ${#API_KEYS[@]} ))
  sleep 2
done
```

### 4. **Background Job Queue**

```python
# Python example with celery
from celery import Celery
import subprocess
import time

app = Celery('crisk_queue')

@app.task(rate_limit='30/m')  # Max 30 checks per minute
def check_file(file_path):
    result = subprocess.run(
        ['crisk', 'check', file_path],
        capture_output=True,
        text=True
    )
    return result.stdout

# Queue files for checking
for file in files_to_check:
    check_file.delay(file)
```

---

## Troubleshooting

### "Still Getting 429 Errors After Retries"

**Causes**:
1. Too many checks in short time window
2. Hitting daily quota (RPD)
3. Project-level rate limits

**Solutions**:
- Wait 5-10 minutes before trying again
- Check [Google AI Studio Dashboard](https://aistudio.google.com/apikey) for quota usage
- Upgrade to paid tier
- Use multiple API keys

### "Retries Taking Too Long"

**If retries add 10-14 seconds per file**:
- You're consistently hitting rate limits
- Need to space out checks more
- Consider using Phase 1 only

**Solution**:
```bash
# Increase spacing between checks
for file in *.ts; do
  crisk check "$file"
  sleep 60  # Increase from 10s to 60s
done
```

### "How Do I Know If I'm Close to Rate Limit?"

**Monitor these patterns in logs**:
```bash
# Count retry attempts in last 100 checks
grep "rate limit encountered" logs.txt | wc -l

# If > 20% of checks need retries, you're too fast
# Increase delays between checks
```

---

## Summary

| Use Case | Recommended Approach | Spacing |
|----------|---------------------|---------|
| **Development** | Phase 1 only | None needed |
| **Pre-commit** | Phase 1 with `--pre-commit` | None (small # of files) |
| **CI/CD** | Phase 1 only, `--quiet` | 5-10s between files |
| **Production Monitoring** | Phase 2 with `--explain` | 30-60s between files |
| **Bulk Analysis** | Phase 1 with caching | 10-30s between files |

**Golden Rule**: If you need Phase 2, space checks by **at least 30 seconds**.

---

## Quick Reference Commands

```bash
# Fast check (no rate limits)
crisk check <file>

# Deep investigation (may hit rate limits)
PHASE2_ENABLED=true crisk check <file> --explain

# Batch check with safety spacing
for f in *.ts; do crisk check "$f"; sleep 30; done

# Check rate limit status
curl -H "x-goog-api-key: $GEMINI_API_KEY" \
  https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash

# Monitor retry attempts
crisk check <file> 2>&1 | grep -E "retry|rate limit"
```

---

For more details, see:
- [RATE_LIMITING_GUIDE.md](../RATE_LIMITING_GUIDE.md) - Full explanation
- [RATE_LIMITING_SOLUTION_SUMMARY.md](../RATE_LIMITING_SOLUTION_SUMMARY.md) - Implementation details
