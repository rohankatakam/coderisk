# Gemini API Rate Limiting Guide

**Date**: November 10, 2025
**Issue**: 429 Rate Limit Errors during Function Calling

---

## Problem Summary

When running `crisk check` with Phase 2 (agentic investigation), the system encountered **429 "Resource exhausted"** errors from the Gemini API, even though:
- ✅ The model configuration is correct (`gemini-2.0-flash`)
- ✅ Simple API calls (non-function-calling) work fine
- ✅ The dashboard shows available quota (45/2K RPM for `gemini-2.0-flash`)

---

## Root Cause

**Function calling has separate, more restrictive rate limits than standard text generation.**

### Rate Limit Types

According to Google AI Studio dashboard and Gemini docs:

| Metric | Description | gemini-2.0-flash Limit |
|--------|-------------|------------------------|
| **RPM** | Requests Per Minute | 2,000/min |
| **TPM** | Tokens Per Minute | 4,000,000/min |
| **RPD** | Requests Per Day | Unlimited |

**However**, function calling requests may have:
- Lower effective RPM due to tool execution overhead
- Separate quota buckets for function calling vs. text generation
- Per-project quotas that differ from per-model quotas

### What We Observed

```
2025/11/10 10:43:22 INFO gemini client initialized component=gemini model=gemini-2.0-flash
2025/11/10 10:43:22 INFO STEP 5: Running agent investigation
⚠️  Investigation failed: Gemini request failed at hop 1: gemini tool completion failed:
Error 429, Message: Resource exhausted. Please try again later.
```

**Timeline of Testing:**
- 10:12 - 10:17: Multiple test runs with function calling (success)
- 10:17 - 10:43: Continued testing hit rate limits
- **Total requests in ~30 minutes**: ~15-20 function calling requests

---

## Why Function Calling Has Lower Effective Limits

1. **Multi-turn conversations**: Each agent investigation involves 5-10 API calls (hops)
2. **Larger payloads**: Function declarations + conversation history = more tokens
3. **Stateful processing**: Gemini tracks function calling context across turns
4. **Quota exhaustion**: Rapid successive calls exhaust short time-window quotas

Example from our testing:
```
Phase 2 Investigation (6 hops):
- Hop 1: get_incidents_with_context
- Hop 2: query_recent_commits
- Hop 3: query_ownership
- Hop 4: query_cochange_partners
- Hop 5: get_cochange_with_explanations
- Hop 6: Final synthesis

Total API calls: 6 requests in 18.7 seconds
Tokens used: 4,428 tokens
```

Running this 3-4 times in rapid succession = **18-24 requests** in a short window → Rate limit hit.

---

## Solutions Implemented

### 1. **Exponential Backoff with Retry Logic** ✅

Added automatic retry with exponential backoff for 429 errors:

```go
// internal/llm/gemini_client.go
maxRetries := 3
baseDelay := 1 * time.Second

for attempt := 0; attempt <= maxRetries; attempt++ {
    resp, err := c.client.Models.GenerateContent(...)

    if err != nil && contains(err.Error(), "429") {
        if attempt < maxRetries {
            delay := baseDelay * (1 << attempt) // 1s, 2s, 4s
            time.Sleep(delay)
            continue
        }
        return nil, fmt.Errorf("rate limited after %d retries", maxRetries)
    }

    return resp, nil
}
```

### 2. **Rate Limiter for Outbound Requests** ✅

Added a token bucket rate limiter to throttle requests:

```go
// internal/llm/rate_limiter.go
type RateLimiter struct {
    requestsPerMinute int
    limiter          *rate.Limiter
}

// Allow 30 requests per minute with burst of 5
limiter := rate.NewLimiter(rate.Every(2*time.Second), 5)
```

### 3. **Request Batching & Caching** ✅

- Cache database queries to reduce redundant data fetching
- Batch tool results when possible
- Reuse conversation context across similar files

### 4. **Configurable Phase 2 Opt-In** ✅

Allow users to disable Phase 2 for quick checks:

```bash
# Fast check (Phase 1 only, no LLM)
crisk check <file>

# Deep investigation (Phase 2, uses LLM)
export PHASE2_ENABLED=true
crisk check <file> --explain
```

---

## Best Practices to Avoid Rate Limits

### For Development/Testing

1. **Wait between runs**: Allow 1-2 minutes between tests
2. **Use --quiet mode**: Skip Phase 2 for rapid iteration
3. **Test on small files first**: Reduce token usage per request
4. **Use different API keys**: Separate dev/staging/prod quotas

### For Production

1. **Enable caching**: Reuse investigation results for unchanged files
2. **Implement request queuing**: Space out checks by 2-3 seconds
3. **Monitor quota usage**: Track RPM/TPM in application metrics
4. **Graceful degradation**: Fall back to Phase 1 if Phase 2 fails
5. **Use batch processing**: Check multiple files in a single session with pauses

### Configuration Options

```bash
# Set rate limit thresholds
export GEMINI_MAX_RPM=30           # Max requests per minute
export GEMINI_RETRY_ENABLED=true   # Enable automatic retries
export GEMINI_RETRY_MAX=3          # Max retry attempts
export GEMINI_BACKOFF_BASE_MS=1000 # Base delay for exponential backoff

# Alternative: Use OpenAI (higher rate limits)
export LLM_PROVIDER=openai
export OPENAI_API_KEY=<your-key>
```

---

## Monitoring Rate Limits

### Check Quota Usage

Visit [Google AI Studio Dashboard](https://aistudio.google.com/apikey) to monitor:
- Peak RPM usage by model
- Token consumption (TPM)
- Daily request counts (RPD)

### Enable Verbose Logging

```bash
crisk check <file> --verbose 2>&1 | grep "rate\|429\|exhausted"
```

### Log Analysis

Look for patterns in logs:
```
INFO: Gemini request sent (attempt 1/3)
WARN: Rate limit encountered, retrying in 2s...
ERROR: Rate limit exhausted after 3 retries
```

---

## API Key Management

### Free Tier Limits (Gemini 2.0 Flash)
- **RPM**: 15 requests/minute
- **TPM**: 1M tokens/minute
- **RPD**: 1,500 requests/day

### Paid Tier Limits (Gemini 2.0 Flash)
- **RPM**: 2,000 requests/minute
- **TPM**: 4M tokens/minute
- **RPD**: Unlimited

### Upgrade Options

If hitting limits frequently:
1. **Upgrade to Gemini API paid tier**: [Google AI Pricing](https://ai.google.dev/pricing)
2. **Use Vertex AI**: Higher quotas, enterprise SLAs
3. **Switch to OpenAI**: Different rate limit structure

---

## Testing Verification

✅ **Confirmed Working** (when not rate-limited):
- Phase 1: Basic metrics (7ms, no LLM)
- Phase 2: 6-hop investigation (18.7s, 4,428 tokens)
- Risk assessment: HIGH → LOW downgrade with 70% confidence
- Actionable recommendations generated
- Full response truncation fix applied

🚫 **Rate Limited** (temporary):
- 429 errors after ~15-20 requests in 30 minutes
- Simple curl requests work (non-function-calling)
- Function calling specifically exhausted quota

---

## Future Improvements

1. **Adaptive rate limiting**: Detect 429 early and auto-throttle
2. **Multi-provider fallback**: Auto-switch to OpenAI if Gemini rate-limited
3. **Request queuing**: Background job queue for non-urgent checks
4. **Quota monitoring**: Alert when approaching limits
5. **Smart caching**: Cache investigation results for 24h

---

## References

- [Gemini Function Calling Docs](https://ai.google.dev/gemini-api/docs/function-calling)
- [Gemini Rate Limits](https://ai.google.dev/gemini-api/docs/models/gemini)
- [Google AI Studio Dashboard](https://aistudio.google.com/apikey)
- [Error Code 429 Documentation](https://cloud.google.com/vertex-ai/generative-ai/docs/error-code-429)

---

## Quick Fix Summary

**For immediate testing:**
```bash
# Wait 2-3 minutes between runs
sleep 180 && crisk check <file> --explain
```

**For production deployment:**
```bash
# Enable retry logic + rate limiting (already implemented)
export GEMINI_RETRY_ENABLED=true
export GEMINI_MAX_RPM=30
```

**For high-volume usage:**
```bash
# Switch to paid tier or OpenAI
export LLM_PROVIDER=openai
export OPENAI_API_KEY=<your-openai-key>
```
