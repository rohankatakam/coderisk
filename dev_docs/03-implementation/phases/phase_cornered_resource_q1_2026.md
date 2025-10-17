# Phase: Cornered Resource (Q1 2026)

**Phase Name:** Build the Incident Knowledge Graph
**Timeline:** Q1 2026 (January - March)
**Status:** Planned
**Owner:** Engineering Team
**Last Updated:** October 4, 2025

> **📘 Product Context:** See [strategic_moats.md](../../00-product/strategic_moats.md) - Cornered Resource strategy
> **📘 Architecture:** See [incident_knowledge_graph.md](../../01-architecture/incident_knowledge_graph.md) for complete technical design

---

## Executive Summary

**Objective:** Establish CodeRisk's irreplaceable data asset—the Incident Knowledge Graph—creating a 5-10 year competitive advantage through cross-industry architectural incident data.

**Strategic Goal:** Launch "CVE for Architecture" (ARC database) and achieve 10,000 incidents attributed to commits by end of Q1 2026.

**Business Impact:**
- **Cornered Resource:** Unique dataset competitors cannot replicate
- **Network Effect:** More incidents = better predictions
- **Revenue:** $500K potential (ARC API, consulting, private incident linking)

---

## Success Criteria

### Must-Have (Launch Blockers)

- ✅ 100 ARC entries published (ARC-2025-001 to ARC-2025-100)
- ✅ 1,000 incidents automatically attributed to commits
- ✅ 10 companies contributing incident data
- ✅ 80% attribution accuracy (verified against manual reviews)
- ✅ Public ARC API live with 99% uptime

### Should-Have (Value Multipliers)

- ⚠️ 10,000 incidents in knowledge graph
- ⚠️ 85% attribution confidence score
- ⚠️ 50 companies opted into pattern sharing
- ⚠️ Federated learning prototype operational

### Nice-to-Have (Future Enhancements)

- 💡 Blockchain integration for immutable audit trail
- 💡 Real-time incident detection (<60s latency)
- 💡 Cross-ARC pattern correlation
- 💡 Predictive incident warnings (7-day forecast)

---

## Week-by-Week Roadmap

### Week 1-2: Foundation (Database & Initial Data)

**Objectives:**
- Set up PostgreSQL schema for ARC database
- Provision Neptune graph database for causal graphs
- Manually curate first 20 ARC entries

**Tasks:**
```
[ ] Deploy PostgreSQL RDS (t3.medium, 100GB)
[ ] Deploy Neptune cluster (t3.medium, 2 instances)
[ ] Create ARC database schema (arc_entries table)
[ ] Create incidents table schema
[ ] Create pattern_signatures table
[ ] Seed 20 initial ARC entries (manually curated)
[ ] Write migration scripts for schema versioning
[ ] Set up database backups (daily snapshots)
```

**Deliverables:**
- PostgreSQL + Neptune running in production
- First 20 ARC entries (ARC-2025-001 to ARC-2025-020)
- Database migration scripts

**Success Metrics:**
- Database uptime: 100%
- Migration script test coverage: 80%
- ARC entries validated by 2 architects

---

### Week 3-4: Monitoring Integrations

**Objectives:**
- Integrate Datadog for incident detection
- Integrate Sentry for error tracking
- Integrate PagerDuty for incident management

**Tasks:**
```
[ ] Set up Datadog webhook endpoint
[ ] Implement incident parsing (Datadog alerts → incidents table)
[ ] Set up Sentry webhook endpoint
[ ] Implement error tracking (Sentry errors → incidents table)
[ ] Set up PagerDuty webhook endpoint
[ ] Implement incident correlation (PagerDuty → incidents table)
[ ] Create unified incident model (supports all 3 sources)
[ ] Add authentication for webhook endpoints (HMAC signatures)
[ ] Write integration tests for each webhook
```

**Deliverables:**
- 3 monitoring integrations live (Datadog, Sentry, PagerDuty)
- Unified incident ingestion pipeline
- Webhook authentication system

**Success Metrics:**
- Webhooks handle 1,000 req/min (load tested)
- 100% of valid webhooks processed successfully
- Zero unauthorized webhook calls accepted

---

### Week 5-6: Attribution Pipeline

**Objectives:**
- Implement automatic incident → commit attribution
- Build deployment correlation logic
- Construct causal graphs in Neptune

**Tasks:**
```
[ ] Implement deployment tracking (git SHA → deploy timestamp)
[ ] Build deployment correlation algorithm (incident time → recent deploys)
[ ] Implement commit analysis (run CodeRisk on historical commits)
[ ] Build pattern matching (graph signature → ARC database)
[ ] Implement causal link creation (commit → deploy → incident edges in Neptune)
[ ] Add confidence scoring for attributions
[ ] Create attribution review dashboard (for manual verification)
[ ] Write integration tests (end-to-end attribution)
```

**Deliverables:**
- Automatic incident attribution system (80% accuracy)
- Neptune causal graphs
- Attribution confidence scoring

**Success Metrics:**
- Attribution accuracy: 80% (vs manual review)
- Avg attribution time: <30 seconds per incident
- False attribution rate: <10%

---

### Week 7-8: Public ARC API

**Objectives:**
- Launch public ARC API (REST)
- Implement pattern search endpoint
- Enable incident submission (authenticated)

**Tasks:**
```
[ ] Design REST API spec (OpenAPI 3.0)
[ ] Implement GET /v1/arc/{arc_id} (retrieve ARC entry)
[ ] Implement POST /v1/arc/search (pattern similarity search)
[ ] Implement POST /v1/incidents/submit (authenticated)
[ ] Add rate limiting (1,000 req/min per IP)
[ ] Add API key authentication (for submissions)
[ ] Write API documentation (Swagger UI)
[ ] Deploy API to production (autoscaling, load balanced)
[ ] Set up monitoring (API latency, error rates)
```

**Deliverables:**
- Public ARC API live at api.coderisk.com/v1/
- API documentation at docs.coderisk.com/api
- Rate limiting + authentication

**Success Metrics:**
- API uptime: 99.9%
- p95 latency: <200ms
- 1,000 API requests/day by week 8

---

### Week 9-10: CLI Integration

**Objectives:**
- Update `crisk check` to show ARC matches
- Add `crisk incident link` for manual attribution
- Update pre-commit hook with ARC warnings

**Tasks:**
```
[ ] Add ARC matching to investigation logic
[ ] Display ARC warnings in `crisk check` output
[ ] Implement `crisk incident link` command
[ ] Update pre-commit hook template (show ARC matches)
[ ] Add --arc flag to show detailed ARC info
[ ] Write CLI tests for ARC integration
[ ] Update CLI documentation
```

**Example Output:**
```bash
$ crisk check

⚠️  HIGH risk detected:

Pattern matches known architectural risks:
  - ARC-2025-001: Auth + User Service Coupling (47 incidents)
  - ARC-2025-034: Payment + Database Coupling (23 incidents)

Your change is 91% similar to ARC-2025-001
Historical outcome: 89% incident rate within 7 days

Recommended actions:
  1. Add integration tests (see ARC-2025-001 mitigation)
  2. Review coupling with user_service.py
```

**Deliverables:**
- `crisk check` shows ARC matches
- `crisk incident link` command
- Updated CLI docs

**Success Metrics:**
- ARC matches shown in 20% of checks
- 10 manual incident links submitted per week
- CLI performance: <100ms overhead for ARC lookup

---

### Week 11-12: Data Collection & Partnership

**Objectives:**
- Partner with 10 friendly companies
- Collect first 1,000 incidents
- Curate first 100 ARC entries

**Tasks:**
```
[ ] Reach out to 20 potential partner companies
[ ] Onboard 10 companies (install webhooks, configure)
[ ] Collect incident data (target: 1,000 incidents)
[ ] Review incidents, identify patterns
[ ] Create 80 new ARC entries (total 100)
[ ] Validate ARC entries with partners
[ ] Set up privacy controls (anonymization)
[ ] Create partner dashboard (view their contributions)
```

**Deliverables:**
- 10 partner companies onboarded
- 1,000 incidents collected
- 100 ARC entries total

**Success Metrics:**
- Partner retention: 90% (9/10 stay active)
- Incident quality: 95% valid (not spam/test data)
- ARC coverage: 80% of incidents match at least one ARC

---

## Technical Architecture

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                 Incident Knowledge Graph                │
│                                                         │
│  ┌──────────┐   ┌──────────────┐   ┌───────────────┐  │
│  │   ARC    │   │   Causal     │   │   Pattern     │  │
│  │ Database │   │    Graph     │   │   Matcher     │  │
│  │          │   │              │   │               │  │
│  │ Postgres │   │   Neptune    │   │  Redis Cache  │  │
│  └──────────┘   └──────────────┘   └───────────────┘  │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Monitoring Integrations                    │
│                                                         │
│  Datadog  ────►  Webhook  ────►  Incident Parser       │
│  Sentry   ────►  Handler  ────►  Attribution Engine    │
│  PagerDuty ───►           ────►  Causal Graph Builder  │
└─────────────────────────────────────────────────────────┘
```

### Data Flow

**Incident Detection → Attribution:**
```
1. Incident occurs in production
2. Datadog sends webhook to CodeRisk
3. Incident parser extracts metadata
4. Find recent deploys (last 24h)
5. For each deploy, get commit SHAs
6. Run CodeRisk analysis on each commit
7. Match patterns to ARC database
8. Calculate attribution confidence
9. Create causal graph edges (commit → deploy → incident)
10. Increment ARC incident count if strong match
```

### Database Schema

**ARC Entries (PostgreSQL):**
```sql
CREATE TABLE arc_entries (
    arc_id VARCHAR(20) PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    pattern_signature VARCHAR(64) NOT NULL,
    severity VARCHAR(10),
    incident_count INTEGER DEFAULT 0,
    company_count INTEGER DEFAULT 0,
    first_reported TIMESTAMPTZ NOT NULL,
    last_updated TIMESTAMPTZ NOT NULL,
    mitigation_steps JSONB,
    affected_patterns JSONB
);
```

**Causal Graph (Neptune):**
```gremlin
// Nodes
commit (sha, timestamp, files_changed, risk_score)
deploy (id, timestamp, service, environment)
incident (id, timestamp, severity, downtime_minutes)
arc (arc_id, title, pattern_signature)

// Edges
commit --[deployed_in]--> deploy
deploy --[caused]--> incident
incident --[matches]--> arc
```

---

## Implementation Priorities

### P0 (Launch Blockers)

**Must complete by March 31, 2026:**

1. **ARC Database** - 100 entries published
2. **Attribution Pipeline** - 80% accuracy
3. **Public API** - 99.9% uptime
4. **CLI Integration** - `crisk check` shows ARC matches
5. **Partner Onboarding** - 10 companies contributing

### P1 (High Value)

**Complete if time permits:**

6. **Federated Learning** - Privacy-preserving pattern matching
7. **Attribution Confidence** - 85%+ confidence scores
8. **Real-time Detection** - <60s incident detection
9. **Cross-ARC Correlation** - Find related ARC patterns

### P2 (Future Phases)

**Defer to Q2 2026:**

10. **Blockchain Integration** - Immutable audit trail
11. **Predictive Warnings** - 7-day incident forecast
12. **Enterprise Dashboard** - Self-service ARC analytics

---

## Risk Mitigation

### Risk 1: Insufficient Incident Data

**Risk:** Only collect 200 incidents instead of 1,000
**Impact:** Cannot accurately identify patterns
**Probability:** Medium (30%)

**Mitigation:**
- Partner with incident-prone industries (fintech, e-commerce)
- Offer free CodeRisk subscription in exchange for data
- Integrate with more monitoring tools (New Relic, Splunk)

**Contingency:**
- Lower success threshold to 500 incidents for Q1
- Extend data collection into Q2
- Focus on quality over quantity (manually curate high-value incidents)

---

### Risk 2: Attribution Accuracy <80%

**Risk:** Automated attribution is too unreliable
**Impact:** Requires manual review (doesn't scale)
**Probability:** Medium (40%)

**Mitigation:**
- Implement confidence scoring (only auto-attribute >85% confidence)
- Add human review queue for medium confidence (70-85%)
- Improve pattern matching algorithm

**Contingency:**
- Hire 2 incident reviewers to manually verify attributions
- Build better deployment correlation (integrate with CD tools)
- Use LLM to improve pattern matching accuracy

---

### Risk 3: Partner Churn

**Risk:** Partners stop contributing data after initial interest
**Impact:** Slow growth of incident database
**Probability:** High (50%)

**Mitigation:**
- Show partners ROI (how their data improved predictions)
- Build partner dashboard (visualize their contributions)
- Gamification (leaderboard of top contributors)

**Contingency:**
- Offer premium features to active contributors
- Share aggregated insights with partners (industry benchmarks)
- Sign data-sharing agreements with committed partners

---

## Success Metrics & KPIs

### Q1 2026 Targets

| Metric | Target | Stretch Goal |
|--------|--------|--------------|
| **ARC Entries** | 100 | 150 |
| **Incidents Collected** | 1,000 | 5,000 |
| **Attribution Accuracy** | 80% | 85% |
| **Partner Companies** | 10 | 20 |
| **API Requests/Day** | 1,000 | 5,000 |
| **Incident→ARC Match Rate** | 70% | 80% |

### Weekly Tracking

**Week 1-2:**
- Database uptime: 100%
- ARC entries: 20

**Week 3-4:**
- Webhook uptime: 99.9%
- Incidents ingested: 100

**Week 5-6:**
- Attributions created: 200
- Attribution accuracy: 75%

**Week 7-8:**
- API uptime: 99.9%
- API requests/day: 100

**Week 9-10:**
- CLI checks showing ARC: 10%
- Manual incident links: 20

**Week 11-12:**
- Partner companies: 10
- Total incidents: 1,000
- Total ARC entries: 100

---

## Budget & Resources

### Infrastructure Costs (Q1 2026)

| Component | Monthly Cost | Q1 Total |
|-----------|-------------|----------|
| PostgreSQL (RDS t3.medium) | $150 | $450 |
| Neptune (t3.medium × 2) | $400 | $1,200 |
| Redis (ElastiCache t3.micro) | $50 | $150 |
| API Service (ECS, ALB) | $200 | $600 |
| Monitoring (Datadog) | $100 | $300 |
| **Total Infrastructure** | **$900/month** | **$2,700** |

### Personnel (Q1 2026)

| Role | Allocation | Cost |
|------|-----------|------|
| Backend Engineer | 100% (3 months) | $60K |
| Data Engineer | 50% (1.5 months) | $22.5K |
| DevOps Engineer | 25% (0.75 months) | $11.25K |
| **Total Personnel** | | **$93.75K** |

### Total Q1 Budget: $96.45K

---

## Handoff to Q2 2026

### Deliverables for Q2 Team

**Data Assets:**
- 100 ARC entries (ARC-2025-001 to ARC-2025-100)
- 1,000+ incidents with commit attribution
- 10 partner companies actively contributing

**Infrastructure:**
- Production-ready ARC API (99.9% uptime)
- Incident attribution pipeline (80% accuracy)
- Monitoring integrations (Datadog, Sentry, PagerDuty)

**Documentation:**
- ARC API docs (Swagger)
- Attribution algorithm documentation
- Partner onboarding guide

### Q2 Focus Areas

**Recommended priorities for Q2 2026:**
1. **Federated Learning** - Privacy-preserving pattern sharing (100 companies)
2. **Improve Attribution** - 85%+ accuracy, real-time detection
3. **Scale Partners** - 50 companies contributing data
4. **Public ARC Website** - coderisk.com/arc browsable catalog
5. **Enterprise Features** - Private ARC entries, custom patterns

---

## Related Documents

**Product Strategy:**
- [strategic_moats.md](../../00-product/strategic_moats.md) - Cornered Resource strategy
- [vision_and_mission.md](../../00-product/vision_and_mission.md) - Trust infrastructure vision

**Architecture:**
- [incident_knowledge_graph.md](../../01-architecture/incident_knowledge_graph.md) - Complete technical design
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Graph schema

**Implementation:**
- [status.md](../status.md) - Current project status
- [phase_trust_layer_q2_q3_2026.md](phase_trust_layer_q2_q3_2026.md) - Next phase

---

**Last Updated:** October 4, 2025
**Next Review:** December 2025 (pre-Q1 kickoff)
