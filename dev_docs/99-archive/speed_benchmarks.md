# CodeRisk Speed Benchmarks & Performance Targets

## Performance Philosophy

CodeRisk prioritizes **"fast enough for development flow"** over unrealistic sub-second targets. Our goal is to be significantly faster than PR-level analysis tools while providing meaningful risk assessment.

## Target Response Times

### Primary Targets

| Codebase Size | Target Time | Maximum Acceptable | Use Case |
|---------------|-------------|-------------------|----------|
| **Small** (<1K files) | 2-3 seconds | 5 seconds | Microservices, libraries |
| **Medium** (1K-10K files) | 3-5 seconds | 10 seconds | Standard applications |
| **Large** (10K-50K files) | 5-8 seconds | 15 seconds | Monoliths, enterprise apps |
| **Enterprise** (50K+ files) | 8-12 seconds | 20 seconds | Large-scale systems |

### Change Size Impact

| Change Scope | Time Modifier | Example |
|--------------|---------------|---------|
| **Single file** | 1.0x base | Bug fix, small feature |
| **3-5 files** | 1.2x base | Feature implementation |
| **10+ files** | 1.5x base | Refactoring |
| **Cross-module** | 2.0x base | Architecture changes |

## Speed vs. Analysis Depth Trade-offs

### Level 1: Quick Check (Target: <5 seconds)
```
✓ Local risk sketches only
✓ Pre-computed graph queries
✓ Cached temporal patterns
✗ No API calls
✗ No deep semantic analysis
```

### Level 2: Standard Check (Target: <10 seconds)
```
✓ Local risk sketches
✓ Limited API calls (≤3)
✓ Pattern matching
✓ Basic semantic analysis
✗ Full graph traversal
```

### Level 3: Deep Check (Target: <20 seconds)
```
✓ Full graph analysis
✓ Multiple API calls
✓ Cross-repository patterns
✓ Complete semantic analysis
✓ Historical correlation
```

## Competitive Context

| Tool | Operation Type | Typical Speed | Depth |
|------|---------------|---------------|-------|
| **Git status** | Local file check | <0.5s | Minimal |
| **ESLint** | Static analysis | 2-10s | Syntax/style |
| **CodeRisk** | Risk assessment | **<10s (target <5s)** | **Risk patterns** |
| **SonarQube** | Full analysis | 30s-5min | Comprehensive |
| **Greptile** | PR analysis | 30s-2min | Full context |
| **Codescene** | Health check | 1-5min | Historical |

## Implementation Strategy

### 1. Pre-computation (During `crisk init`)
- Build risk sketches
- Calculate graph metrics
- Cache temporal patterns
- Store in local SQLite

### 2. Incremental Updates
- Track file changes since last init
- Update only affected risk sketches
- Maintain cache freshness

### 3. Smart Degradation
```python
if time_elapsed > 5_seconds and not critical_risk_found:
    return partial_results_with_warning()
```

### 4. Parallel Processing
- Concurrent risk detector execution
- Parallel graph queries
- Async API calls when needed

## Performance Monitoring

### Key Metrics

1. **P50 Response Time**: Target <5 seconds
2. **P95 Response Time**: Target <10 seconds
3. **P99 Response Time**: Target <15 seconds
4. **Cache Hit Rate**: Target >90%

### Benchmark Test Suite

```bash
# Small project test
crisk benchmark --size small   # Should complete <3s

# Medium project test
crisk benchmark --size medium  # Should complete <5s

# Large project test
crisk benchmark --size large   # Should complete <10s

# Real-world test
crisk benchmark --repo ./path  # Actual performance
```

## User Experience Guidelines

### Speed Perception

| Actual Time | User Perception | UI Response |
|-------------|----------------|-------------|
| <2s | Instant | Simple spinner |
| 2-5s | Fast | Progress indicator |
| 5-10s | Acceptable | Detailed progress |
| >10s | Slow | Show partial results |

### Progressive Results

```
0-2s:  Show critical risks (if any)
2-5s:  Add high-priority warnings
5-10s: Complete analysis with suggestions
>10s:  Timeout with partial results
```

## Optimization Priorities

### Phase 1: Foundation (Current)
- [ ] Implement local risk sketch caching
- [ ] Remove serialization bottlenecks
- [ ] Optimize SQLite queries

### Phase 2: Enhancement
- [ ] Add incremental analysis
- [ ] Implement smart cache invalidation
- [ ] Parallel detector execution

### Phase 3: Scale
- [ ] Distributed caching for teams
- [ ] Cloud-assisted analysis (optional)
- [ ] Background pre-computation

## Realistic Expectations

### What We Promise
✅ **<10 second** checks for most operations
✅ **<5 second** target for small changes
✅ **Consistent performance** via caching
✅ **Faster than PR-level tools** by 5-10x

### What We Don't Promise
❌ Sub-second response for large codebases
❌ Real-time analysis during typing
❌ Instant first-run (init still required)
❌ Same speed as simple linters

## Marketing Messaging

### Primary: "Risk Assessment in Seconds"
- "Know your risk in under 10 seconds"
- "5x faster than waiting for PR feedback"
- "Quick enough for your development flow"

### Secondary: "Speed Without Sacrifice"
- "Meaningful analysis, not just syntax"
- "Cached intelligence for instant results"
- "Deep insights at development speed"

## Conclusion

CodeRisk targets **<10 second response times** with a goal of **<5 seconds** for typical operations. This is:
- **10x faster** than PR-level analysis tools
- **Fast enough** for iterative development
- **Realistic** for meaningful risk assessment
- **Achievable** with proper architecture

The focus is on being "fast enough to not interrupt flow" rather than achieving unrealistic sub-second speeds that would compromise analysis quality.