# Success Metrics & OKRs

**Last Updated:** October 2, 2025
**Owner:** Product Team
**Status:** Active

> **📘 Cross-reference:** See [spec.md](../spec.md) Section 2.4 for technical performance targets

---

## Executive Summary

CodeRisk success is measured across three dimensions:
1. **Performance**: Latency, accuracy, reliability (developer experience)
2. **Quality**: False positive rate, incident prevention, user satisfaction
3. **Business**: Revenue, adoption, retention (sustainable growth)

**North Star Metric:** Daily checks per user (>10 = habit formation)

**Critical Success Factors:**
- <3% false positive rate (trust)
- <5 second latency p50 (usability)
- >85% incident prediction accuracy (value)
- >1,000 teams within 12 months (traction)

---

## Performance Metrics ⚡

### Latency Targets

| Metric | Target (p50) | Target (p95) | Blocker Threshold | Current Status |
|--------|--------------|--------------|-------------------|----------------|
| **Total Check Latency** | 2-5s | 8s | >10s (unusable) | 🚧 TBD (MVP) |
| **Phase 1 (Baseline)** | 200ms | 500ms | >1s | 🚧 TBD |
| **Phase 2 (LLM Investigation)** | 3-5s | 8s | >15s | 🚧 TBD |
| **Neptune Cold Start** | 100ms | 500ms | >1s | ✅ Spec validated |
| **Per-Hop Latency** | 1-2s | 3s | >5s | 🚧 TBD |

**Why This Matters:**
- <5s → Developer stays in flow state (no context switch)
- >10s → Developer abandons tool (checks email, loses focus)
- Industry benchmark: `git status` ~50ms, `git diff` ~200ms, linters ~1-3s

**Measurement Strategy:**
- Instrument every phase (Phase 1, Phase 2, Neptune queries, LLM calls)
- Track p50, p95, p99 (p50 = typical experience, p95 = worst 5%)
- Alert if p95 > 8s for 5 consecutive checks

---

### Throughput & Scalability

| Metric | Target | Blocker Threshold | Current Status |
|--------|--------|-------------------|----------------|
| **Checks per Second (System-Wide)** | 100 checks/s | <10 checks/s | 🚧 Load test pending |
| **Concurrent Users (per Neptune DB)** | 1,000 users | <100 users | 🚧 TBD |
| **Cache Hit Rate (Redis)** | >90% | <70% | 🚧 TBD |
| **Public Cache Hit Rate** | >80% | <50% | 🚧 TBD |
| **Graph Build Time (5K files)** | 5-10 min | >15 min | 🚧 TBD |
| **Incremental Update Time (push to main)** | 10-30s | >60s | 🚧 TBD |

**Why This Matters:**
- 100 checks/s = 8.6M checks/day (supports 10K users at 10 checks/day each)
- Cache hit rate directly impacts latency (cache miss = 5-10× slower)
- Graph build time impacts onboarding (15 min wait = user abandons)

**Measurement Strategy:**
- Weekly load tests (simulate 1K, 5K, 10K concurrent users)
- Monitor Neptune auto-scaling (NCUs, query latency)
- Track cache hit rate by category (investigation cache, metric cache, materialized views)

---

### Cost Efficiency

| Metric | Target | Blocker Threshold | Current Status |
|--------|--------|-------------------|----------------|
| **Infrastructure Cost per User** | $2.30/month | >$5/month | ✅ Modeled in spec.md |
| **LLM Cost per Check (User's API Key)** | $0.03-0.05 | >$0.10 | 🚧 TBD (depends on hop count) |
| **Graph Hops per Check (Average)** | 2-3 hops | >5 hops | 🚧 TBD |
| **Public Cache Storage Cost** | $54/month (top 100 repos) | >$200/month | 🚧 TBD |
| **Neptune Cost (1K users)** | $545/month | >$1,000/month | ✅ Modeled |

**Why This Matters:**
- Unit economics: Must maintain >70% gross margin
- User LLM costs: $0.05/check × 100 checks/month = $5/month (acceptable)
- If LLM costs >$0.10/check → $10/month → user churn

**Measurement Strategy:**
- Track hop efficiency (% of checks completing in ≤3 hops)
- Monitor Neptune NCU usage (alert if consistently >50 NCUs)
- Weekly cost review (infrastructure spend vs active users)

---

## Quality Metrics 🎯

### False Positive Rate (CRITICAL)

| Metric | Target | Acceptable | Blocker | Current Status |
|--------|--------|------------|---------|----------------|
| **Overall FP Rate** | <3% | <5% | >10% | 🚧 Baseline in progress |
| **Tier 1 Metrics FP Rate** | <3% | <5% | >5% | 🚧 TBD |
| **Tier 2 Metrics FP Rate** | <8% | <12% | >15% | 🚧 TBD |
| **Phase 1 (Baseline) FP Rate** | <5% | <8% | >10% | 🚧 TBD |
| **Phase 2 (LLM Investigation) FP Rate** | <3% | <5% | >8% | 🚧 TBD |

**False Positive Definition:**
- User runs `crisk check` → HIGH risk warning
- User ships change → No incident within 30 days
- User marks as false positive via `crisk override --false-positive`

**Why This Matters:**
- 10% FP rate = Developer ignores 1 in 10 warnings (alert fatigue)
- 3% FP rate = Developer trusts warnings (takes action)
- Competitor benchmark: SonarQube ~10-20%, Codescene ~5-10%

**Measurement Strategy:**
1. **User Overrides**: Track `crisk override --false-positive` submissions
2. **Incident Correlation**: Did HIGH warning → no incident within 30 days?
3. **A/B Testing**: Test new metrics on 10% of users, measure FP delta
4. **Auto-Exclusion**: Disable any metric with >3% FP for 2+ weeks

**Tracking Schema (Postgres):**
```sql
CREATE TABLE metric_validation (
  metric_name TEXT PRIMARY KEY,
  total_predictions INT DEFAULT 0,
  false_positives INT DEFAULT 0,
  fp_rate FLOAT GENERATED ALWAYS AS (false_positives::FLOAT / NULLIF(total_predictions, 0)),
  last_updated TIMESTAMPTZ DEFAULT NOW(),
  status TEXT CHECK (status IN ('active', 'probation', 'disabled'))
);
```

---

### Incident Prediction Accuracy

| Metric | Target | Acceptable | Blocker | Current Status |
|--------|--------|------------|---------|----------------|
| **Incident Recall** | >85% | >70% | <50% | 🚧 Requires historical data |
| **Incident Precision** | >60% | >40% | <20% | 🚧 Requires historical data |
| **True Positive Rate** | >90% | >80% | <70% | 🚧 TBD |

**Definitions:**
- **Recall**: Of all production incidents, what % did CodeRisk predict?
  - Example: 10 incidents in Q1, CodeRisk warned on 9 → 90% recall
- **Precision**: Of all HIGH warnings, what % became incidents?
  - Example: 100 HIGH warnings, 15 became incidents → 15% precision
- **True Positive**: HIGH warning → Incident within 30 days

**Why This Matters:**
- High recall = Catch most incidents (safety)
- High precision = Low false positive rate (trust)
- Trade-off: Can't optimize both (choose recall for V1.0)

**Measurement Strategy:**
1. **Retrospective Analysis**: Run CodeRisk on historical commits before known incidents
2. **Incident Linking**: `crisk incident link INC-123 --commit abc456`
3. **Post-Mortem Integration**: Did we warn about the risky code?
4. **Quarterly Review**: Analyze all incidents, calculate recall/precision

**Example Calculation:**
```
Q1 2026:
- Total incidents: 12
- CodeRisk predicted: 10 (83% recall)
- Total HIGH warnings: 67
- True positives (became incidents): 10 (15% precision)
- False positives: 57 (85% noise - UNACCEPTABLE, need to improve)

Action: Increase warning threshold (reduce HIGH rate), improve metric selection
```

---

### User Satisfaction

| Metric | Target | Acceptable | Blocker | Measurement Method |
|--------|--------|------------|---------|-------------------|
| **Net Promoter Score (NPS)** | >4.5/5 | >4.0/5 | <3.5/5 | Post-check survey (10% sample) |
| **Daily Active Users (DAU)** | >70% WAU | >50% WAU | <30% WAU | CLI telemetry |
| **Checks per User per Day** | >10 | >5 | <3 | CLI telemetry |
| **Retention (30-day)** | >80% | >60% | <40% | Cohort analysis |
| **Time to First Value** | <2 min | <5 min | >10 min | Onboarding analytics |

**NPS Survey (After 20th Check):**
```
How likely are you to recommend CodeRisk to a colleague? (0-10)
[ 0 1 2 3 4 5 6 7 8 9 10 ]

Why? (Optional)
[ Free text ]
```

**Why This Matters:**
- NPS 4.5+ = Strong word-of-mouth growth
- 10 checks/day = Habit formation (muscle memory)
- 80% 30-day retention = Product-market fit

**Measurement Strategy:**
- CLI telemetry (anonymous, opt-in): Check count, latency, overrides
- In-app surveys (post-check, non-intrusive)
- User interviews (monthly, 5-10 users)

---

## Business Metrics 💰

### Revenue & Growth

| Metric | Q1 2026 Target | Q2 2026 Target | Q4 2026 Target | Current Status |
|--------|----------------|----------------|----------------|----------------|
| **Monthly Recurring Revenue (MRR)** | $10K | $30K | $100K | $0 (pre-launch) |
| **Annual Recurring Revenue (ARR)** | $120K | $360K | $1.2M | $0 |
| **Paid Users** | 1,000 | 3,000 | 10,000 | 0 |
| **Enterprise Contracts** | 2 | 5 | 15 | 0 |
| **Average Revenue per User (ARPU)** | $10 | $10 | $10 | N/A |

**Why This Matters:**
- $1M ARR = Series A fundable (18-24 months runway)
- 10K paid users = Critical mass for network effects
- Enterprise contracts = Revenue stability (reduces churn risk)

**Measurement Strategy:**
- Weekly revenue dashboard (Stripe + internal billing)
- Cohort analysis (which months have best retention?)
- Sales pipeline tracking (leads, demos, contracts)

---

### Adoption & Activation

| Metric | Target | Acceptable | Blocker | Measurement Method |
|--------|--------|------------|---------|-------------------|
| **Sign-ups per Month** | 1,000 | 500 | <100 | Marketing analytics |
| **Free → Starter Conversion** | 15% | 10% | <5% | Funnel analysis |
| **Starter → Pro Upgrade** | 20% | 10% | <5% | Upgrade tracking |
| **Time to First Check** | <5 min | <10 min | >20 min | Onboarding telemetry |
| **Team Invites (Pro users)** | 3 invites/user | 2 invites/user | <1 invite/user | Referral tracking |

**Activation Funnel:**
```
1. Sign up (1,000 users/month)
   ↓ 80%
2. Install CLI (800 users)
   ↓ 70%
3. Run first check (560 users)
   ↓ 50%
4. Run 10+ checks (280 users) ← Habit formation
   ↓ 30%
5. Upgrade to Starter (84 paid users) ← 8.4% overall conversion
```

**Leaky Bucket Analysis:**
- Where do users drop off? (identify friction points)
- A/B test onboarding flow (OAuth vs manual API key)
- Exit surveys for churned users

---

### Retention & Churn

| Metric | Target | Acceptable | Blocker | Measurement Method |
|--------|--------|------------|---------|-------------------|
| **Monthly Churn Rate** | <5% | <8% | >12% | Cohort analysis |
| **Annual Churn Rate** | <40% | <50% | >60% | Year-over-year retention |
| **Dollar Churn Rate** | <3% | <5% | >10% | Revenue retention |
| **Net Revenue Retention (NRR)** | >100% | >90% | <80% | Upgrades - downgrades |

**Churn Analysis:**
- Voluntary churn: User cancels (why? survey them)
- Involuntary churn: Payment fails (retry logic, dunning emails)
- Downgrades: Pro → Starter (budget cuts, team shrinks)

**Why This Matters:**
- 5% monthly churn = 60% annual churn (unsustainable)
- 3% monthly churn = 36% annual churn (acceptable for early stage)
- NRR >100% = Expansion revenue exceeds churn (growth mode)

**Cohort Retention Example:**
```
Jan 2026 Cohort (100 users):
- Month 0: 100 users (100%)
- Month 1: 92 users (92% retained, 8% churned)
- Month 3: 81 users (81% retained, 19% churned)
- Month 6: 70 users (70% retained, 30% churned)

Target: >80% 6-month retention
```

---

## Category Creation Metrics 📈

### "Pre-Flight Check" Brand Awareness

| Metric | Target | Acceptable | Measurement Method |
|--------|--------|------------|-------------------|
| **"Pre-flight check" Google searches** | 1,000/month | 500/month | Google Trends |
| **"crisk check" as verb usage** | 50+ mentions | 20+ mentions | Twitter/Reddit monitoring |
| **Analyst mentions (Gartner, Forrester)** | 3+ reports | 1 report | Media tracking |
| **Competitor repositioning** | 2+ competitors | 1 competitor | Competitive intel |

**Why This Matters:**
- Category creation = Own mindshare (developers think "CodeRisk = pre-flight check")
- Verb usage = Ultimate stickiness ("Let me crisk this before committing")
- Analyst recognition = Enterprise credibility

**Evidence of Success:**
- Blog posts: "How to do pre-flight checks with CodeRisk"
- Competitor messaging shifts: "Unlike pre-flight checks, we focus on..."
- Job postings: "Experience with pre-flight check tools (CodeRisk, etc.)"

---

## OKRs (Objectives & Key Results)

### Q1 2026: Achieve Product-Market Fit

**Objective 1: Validate Core Value Proposition**

**Key Results:**
1. ✅ Launch MVP to 100 beta users
2. ✅ Achieve <5% false positive rate on Tier 1 metrics
3. ✅ 80% of beta users report "would be disappointed without CodeRisk"
4. ✅ >10 checks/user/day average (habit formation)

**Objective 2: Prove Technical Feasibility**

**Key Results:**
1. ✅ P50 latency <5 seconds
2. ✅ Successfully onboard public repos (React, Next.js) to shared cache
3. ✅ Zero-config onboarding success rate >95%
4. ✅ Infrastructure costs <$3/user/month

**Objective 3: Generate Initial Revenue**

**Key Results:**
1. ✅ 1,000 sign-ups (free tier)
2. ✅ 100 paid users (Starter tier)
3. ✅ $10K MRR
4. ✅ 2 enterprise pilot contracts

---

### Q2 2026: Scale to 1,000 Teams

**Objective 1: Optimize Performance & Quality**

**Key Results:**
1. ✅ Reduce false positive rate to <3%
2. ✅ Achieve >85% incident prediction recall
3. ✅ P50 latency <3 seconds (20% improvement)
4. ✅ NPS >4.5/5

**Objective 2: Drive Adoption**

**Key Results:**
1. ✅ 1,000 teams using CodeRisk daily
2. ✅ 5,000 paid users
3. ✅ Free → Starter conversion rate >15%
4. ✅ Public cache covers top 100 OSS repos (>80% hit rate)

**Objective 3: Expand Revenue**

**Key Results:**
1. ✅ $50K MRR
2. ✅ 5 enterprise contracts (>$5K/month each)
3. ✅ <5% monthly churn rate
4. ✅ Launch Pro tier (team sharing, webhooks)

---

### Q3 2026: Build Competitive Moat

**Objective 1: Category Leadership**

**Key Results:**
1. ✅ "Pre-flight check" term appears in 3+ analyst reports
2. ✅ 2+ competitors reposition against CodeRisk
3. ✅ 500+ "crisk check" mentions on Twitter/Reddit
4. ✅ Featured in major tech media (TechCrunch, The New Stack, InfoQ)

**Objective 2: Technical Differentiation**

**Key Results:**
1. ✅ Metric validation system live (auto-disable metrics >3% FP)
2. ✅ Incident linking feedback loop (>100 incidents linked)
3. ✅ Cross-repo pattern learning (detect auth issues across team repos)
4. ✅ Branch delta efficiency >98% (vs full graph)

**Objective 3: Enterprise Traction**

**Key Results:**
1. ✅ 15 enterprise contracts
2. ✅ $100K MRR
3. ✅ Launch self-hosted VPC deployment (Enterprise tier)
4. ✅ SOC2 Type I compliance

---

### Q4 2026: Achieve $1M ARR

**Objective 1: Scale Infrastructure**

**Key Results:**
1. ✅ Support 10,000 paid users
2. ✅ 100 checks/second system throughput
3. ✅ <2% infrastructure downtime
4. ✅ Public cache covers 500+ OSS repos

**Objective 2: Revenue Growth**

**Key Results:**
1. ✅ $100K MRR ($1.2M ARR run rate)
2. ✅ 10,000 paid users
3. ✅ 50 enterprise contracts
4. ✅ <3% monthly churn rate

**Objective 3: Product Maturity**

**Key Results:**
1. ✅ CI/CD integration (GitHub Actions, GitLab CI)
2. ✅ API for third-party tools (IDE extensions)
3. ✅ Real-time webhook graph updates (<30s)
4. ✅ Team analytics dashboard (usage, costs, FP rates)

---

## Metric Dashboard (Weekly Review)

### Performance Dashboard

```
┌─────────────────────────────────────────────┐
│ Performance Metrics (Week of Oct 2, 2025)   │
├─────────────────────────────────────────────┤
│ Latency (p50):        3.2s  ✅ (target: <5s) │
│ Latency (p95):        7.8s  ✅ (target: <8s) │
│ Cache Hit Rate:       87%   ⚠️  (target: >90%)│
│ Graph Hops (avg):     2.8   ✅ (target: 2-3) │
│ LLM Cost/Check:    $0.041   ✅ (target: <$0.05)│
│ Infrastructure Cost: $2.45  ✅ (target: <$3)  │
└─────────────────────────────────────────────┘
```

### Quality Dashboard

```
┌─────────────────────────────────────────────┐
│ Quality Metrics (Week of Oct 2, 2025)       │
├─────────────────────────────────────────────┤
│ False Positive Rate:  2.8%  ✅ (target: <3%) │
│ Incident Recall:      82%   ⚠️  (target: >85%)│
│ User NPS:            4.6/5  ✅ (target: >4.5) │
│ Checks/User/Day:      11.2  ✅ (target: >10)  │
│ 30-Day Retention:     78%   ⚠️  (target: >80%)│
└─────────────────────────────────────────────┘
```

### Business Dashboard

```
┌─────────────────────────────────────────────┐
│ Business Metrics (Week of Oct 2, 2025)      │
├─────────────────────────────────────────────┤
│ MRR:                $12.3K  ✅ (target: $10K) │
│ Paid Users:          1,234  ✅ (target: 1K)  │
│ Sign-ups (week):       287  ⚠️  (target: 300) │
│ Free→Starter Conv:    14%   ⚠️  (target: 15%) │
│ Monthly Churn:       4.2%   ✅ (target: <5%)  │
│ Enterprise Deals:       3   ✅ (target: 2)    │
└─────────────────────────────────────────────┘
```

---

## Alert Thresholds (Automated Monitoring)

### P0 Alerts (Page Immediately)

| Alert | Threshold | Action |
|-------|-----------|--------|
| **P95 latency >15s** | 5 consecutive checks | Scale Neptune, investigate agent loops |
| **False positive rate >10%** | 50+ overrides in 24h | Disable suspect metric, rollback |
| **System downtime** | >5 min | Incident response, status page update |
| **LLM cost spike** | >$0.15/check | Kill runaway agents, audit hop limits |

### P1 Alerts (Investigate within 1 hour)

| Alert | Threshold | Action |
|-------|-----------|--------|
| **P95 latency >8s** | Sustained 1 hour | Optimize slow queries, check Neptune NCUs |
| **Cache hit rate <70%** | Sustained 1 hour | Investigate cache invalidation, increase TTL |
| **Monthly churn >8%** | Month-end | User surveys, retention analysis |
| **Infrastructure cost >$3/user** | Weekly review | Optimize Neptune queries, review NCU usage |

### P2 Alerts (Review Weekly)

| Alert | Threshold | Action |
|-------|-----------|--------|
| **NPS <4.0** | 20+ responses | User interviews, product improvements |
| **Free→Starter conversion <10%** | Monthly | A/B test pricing, improve onboarding |
| **Checks/user/day <5** | Cohort analysis | Re-engagement campaigns, feature tutorials |

---

## Reporting Cadence

### Daily (Automated Slack Bot)

- Total checks (yesterday)
- Latency (p50, p95)
- False positive overrides
- New sign-ups, paid conversions
- Infrastructure costs

### Weekly (Team Meeting)

- Performance dashboard review
- Quality dashboard review
- Business dashboard review
- OKR progress tracking
- Incidents & postmortems

### Monthly (Board/Investors)

- MRR, ARR, paid users
- Churn rate, retention cohorts
- OKR scorecard (on track / at risk)
- Competitive intel updates
- Next month priorities

### Quarterly (Strategic Review)

- OKR retrospective (what worked, what didn't)
- Product roadmap adjustments
- Pricing strategy review
- Competitive landscape analysis
- Next quarter OKRs

---

## Success Criteria by Phase

### MVP (Month 1-3)

**Must Have:**
- ✅ 100 beta users
- ✅ <5% false positive rate
- ✅ <5s latency p50
- ✅ Zero-config onboarding >90%

**Nice to Have:**
- ⚠️ $5K MRR
- ⚠️ 10 checks/user/day
- ⚠️ 1 enterprise pilot

**Blocker to Next Phase:**
- ❌ False positive rate >10% (fix before scaling)
- ❌ Latency p95 >15s (unusable)
- ❌ Infrastructure costs >$5/user (unsustainable)

---

### Growth (Month 4-9)

**Must Have:**
- ✅ 1,000 paid users
- ✅ $30K MRR
- ✅ <3% false positive rate
- ✅ 5 enterprise contracts

**Nice to Have:**
- ⚠️ $50K MRR
- ⚠️ 80% public cache hit rate
- ⚠️ NPS >4.5

**Blocker to Next Phase:**
- ❌ Churn rate >10% (leaky bucket)
- ❌ Infrastructure costs exceed revenue (unprofitable)

---

### Scale (Month 10-18)

**Must Have:**
- ✅ $100K MRR ($1.2M ARR)
- ✅ 10,000 paid users
- ✅ 50 enterprise contracts
- ✅ Category recognition (analyst mentions)

**Nice to Have:**
- ⚠️ Verb usage ("crisk check")
- ⚠️ Competitor repositioning
- ⚠️ SOC2 Type II

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - Strategic goals and category creation
- [user_personas.md](user_personas.md) - User behavior and adoption triggers
- [competitive_analysis.md](competitive_analysis.md) - Market positioning and benchmarks
- [pricing_strategy.md](pricing_strategy.md) - Revenue and conversion targets

**Technical:**
- [spec.md](../spec.md) - Technical performance targets (Section 2.4)
- [01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Metric validation system

---

**Last Updated:** October 2, 2025
**Next Review:** January 2026 (quarterly OKR planning)
