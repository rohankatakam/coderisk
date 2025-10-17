# Open Core Strategy

**Last Updated:** 2025-10-12
**Owner:** Product & Strategy Team
**Status:** Active - Hybrid Open Source Model Approved

> **📘 Cross-reference:** See [strategic_moats.md](strategic_moats.md) for 7 Powers alignment, [competitive_analysis.md](competitive_analysis.md) for market positioning, and [vision_and_mission.md](vision_and_mission.md) for product vision.

---

## Executive Summary

CodeRisk adopts a **hybrid open core model**: open source CLI and local mode under MIT License, with proprietary cloud infrastructure and trust platform features. This strategy maximizes adoption while protecting competitive moats.

**Key Decisions:**
1. ✅ **Open Source:** CLI, local mode, Phase 1 metrics (MIT License)
2. ❌ **Closed Source:** Cloud platform, ARC database, Phase 2 LLM, trust infrastructure
3. 🎯 **Goal:** Drive adoption through open source, monetize through cloud SaaS

**Strategic Rationale:**
- **Adoption:** Open source removes adoption barriers (trust, audit, lock-in fears)
- **Distribution:** Enables Homebrew, package managers, viral growth
- **Moat Protection:** Keeps proprietary moats (ARC database, insurance, cross-org learning) closed
- **Revenue:** Free-to-paid funnel via cloud upgrade path

**Precedents:** Sentry ($100M ARR), GitLab ($15B valuation), Elastic ($1B+ valuation), HashiCorp ($5B+ valuation) - all successful open core companies.

---

## License Structure

### MIT License (Open Source)

**Repository:** `github.com/rohankatakam/coderisk-go` (public)

**Components:**
- ✅ CLI tool (`crisk` binary and all source in `cmd/crisk/`)
- ✅ Core graph engine (`internal/graph/`, `internal/ingestion/`)
- ✅ Phase 1 baseline metrics (`internal/metrics/tier1/`)
- ✅ Tree-sitter parsers (`internal/treesitter/`)
- ✅ Pre-commit hook system (`internal/hooks/`)
- ✅ Local Docker Compose stack (`docker-compose.yml`)
- ✅ Database schemas and migrations (local mode)

**Use Cases:**
- Individual developers and hobbyists
- Small teams (1-5 people)
- Privacy-sensitive environments (air-gapped, on-premise)
- Educational and research purposes
- Community contributions

**Distribution:**
- Homebrew formula: `brew install rohankatakam/coderisk/crisk`
- Debian/Ubuntu packages
- Docker Hub official images
- Direct download from GitHub Releases

### Proprietary License (Closed Source)

**Repository:** `github.com/coderisk-private/cloud` (private)

**Components:**
- ❌ Multi-tenant cloud infrastructure (AWS Neptune, ECS, RDS)
- ❌ Public repository caching (reference counting, garbage collection)
- ❌ Branch delta optimization (98% storage reduction)
- ❌ ARC (Architectural Risk Catalog) database contents (100+ patterns, 10K+ incidents)
- ❌ Phase 2 LLM investigation engine (agentic graph navigation)
- ❌ Cross-organization pattern learning (federated learning)
- ❌ Trust infrastructure (certificates, insurance, provenance)
- ❌ Enterprise features (SSO, SAML, audit logs, self-hosted)
- ❌ Settings portal (Next.js web app)
- ❌ Team management and billing

**Revenue Model:**
- Cloud SaaS: $10-50/user/month
- Trust certificates: $0.05/certificate
- Certification program: $299-2,999/seat
- ARC consulting: $500K/year potential
- AI code insurance: $0.10/insured check

---

## Strategic Rationale

### Why Open Source (CLI + Local Mode)

**Problem Solved:** Adoption barriers for security-sensitive developer tools

**Benefits:**

**1. Trust & Transparency**
```
Developer thinking: "Should I adopt CodeRisk?"
→ Open source: "I can audit the code" ✅
→ Closed source: "What if there's a backdoor?" ❌
```

**Evidence:**
- Sentry: Open SDK → developers trusted → 100K+ organizations
- GitLab: Open core → auditable → enterprise adoption
- Terraform: Open CLI → viral adoption → $5B+ valuation

**2. Distribution & Viral Growth**

**Homebrew/Package Managers (Open Source Required):**
```bash
# Enabled by open source
brew install coderisk
apt-get install coderisk
docker pull coderisk/crisk

# Not possible with closed source
# Developers must clone + build = friction
```

**Network Effects:**
- Developer A uses CodeRisk → tells Developer B
- Developer B installs via Homebrew (30 seconds) → tells Developer C
- Viral loop: Low friction adoption = exponential growth

**3. Community Contributions**

**Open Source Enables:**
- New metrics contributed by community
- Language parsers for niche languages (Elixir, Haskell, etc.)
- Bug fixes from users who hit edge cases
- Documentation improvements

**Example:**
- React: Community contributions → 2,000+ contributors → dominance
- VS Code: Open source → extensions ecosystem → market leader

**4. No Lock-In Fears**

**Enterprise Decision-Makers:**
```
CTO thinking: "What if CodeRisk goes out of business?"
→ Open source: "We can run local mode forever" ✅
→ Closed source: "We'd be stuck" ❌
```

**Sales Advantage:**
- Enterprise customers require "no lock-in" assurance
- Open core = "You own the code" = lower perceived risk
- Easier procurement approval

---

### Why Closed Source (Cloud + Trust Platform)

**Problem Solved:** Protecting competitive moats and revenue

**Strategic Moats Requiring Closed Source:**

#### 1. Cornered Resource: ARC Database

**Asset:** 100+ architectural risk patterns from 10,000 GitHub incidents

**Timeline to Build:**
- With GitHub mining: 8 weeks (our approach)
- Organic growth: 3-5 years
- Competitor copying open source: Day 1

**Decision:** ❌ Keep closed to protect 3-5 year competitive lead

**Revenue Impact:**
- ARC API access: $500K/year potential
- Consulting on ARC remediation: $200K/year potential
- "First ARC Database" brand: Priceless

#### 2. Counter-Positioning: Insurance Business Model

**Asset:** Actuarial models for AI code risk underwriting

**Risk if Open:**
- Competitors copy underwriting algorithm
- Gaming risk (users optimize to get insured, then write bad code)
- Reinsurance partners require proprietary models

**Decision:** ❌ Keep closed to protect insurance moat ($400K/year potential revenue)

#### 3. Network Effects: Cross-Org Learning

**Asset:** Federated learning infrastructure (privacy-preserving)

**Implementation:** 23 companies → 47 incidents learned → 91% prediction accuracy

**Risk if Open:**
- Competitors replicate network → fragment market
- Loss of "largest pattern database" advantage

**Decision:** ❌ Keep closed to maintain network effect moat

#### 4. Cloud Infrastructure: Multi-Tenant Platform

**Asset:** AWS Neptune, public caching, branch deltas (99% storage reduction)

**Cost to Build:** 8 weeks engineering + $2.30/user/month operating cost

**Risk if Open:**
- AWS/Google fork → offer managed CodeRisk
- Undercut on pricing (they have infrastructure at cost)
- Loss of $10-50/user/month revenue stream

**Decision:** ❌ Keep closed to protect SaaS revenue ($500K-2M ARR potential)

---

### Competitive Comparison

| Company | Model | Open Source Components | Proprietary Components | Result |
|---------|-------|----------------------|------------------------|---------|
| **Sentry** | Open Core | SDK, error tracking | Cloud platform, analytics | $100M+ ARR |
| **GitLab** | Open Core | Core platform, CI/CD | Enterprise features, Geo | $15B valuation |
| **Elastic** | Open Core (→ SSPL) | Search engine | X-Pack features | $1B+ valuation |
| **HashiCorp** | Open Core | Terraform CLI, Vault core | Cloud platform, Sentinel | $5B+ valuation |
| **SonarQube** | Open Core | Community Edition | Enterprise Edition | $200M+ ARR |
| **Greptile** | Fully Closed | None | Everything | Slower adoption |
| **CodeRisk** | Open Core | CLI, local mode | Cloud, ARC, trust platform | **Our Model** |

**Key Insight:** Open core wins for developer tools. 100% closed → slow adoption. 100% open → no moat. Hybrid = best of both.

---

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2) ✅ COMPLETE

**Actions Taken:**
- ✅ Added [LICENSE](../../LICENSE) (MIT License with scope clarification)
- ✅ Updated [README.md](../../README.md) with open source badges and commercial platform section
- ✅ Created [CONTRIBUTING.md](../../CONTRIBUTING.md) with contribution guidelines
- ✅ Documented this open core strategy

**Current State:**
- Repository: `github.com/rohankatakam/coderisk-go` (public)
- License: MIT License
- Components: CLI, local mode, Phase 1 metrics

### Phase 2: Community Launch (Weeks 3-4)

**Goals:**
- 100 GitHub stars in 4 weeks
- 10 external contributors
- 5 OSS projects using CodeRisk

**Tactics:**

**1. Packaging & Distribution**
```bash
# Homebrew tap
brew tap rohankatakam/coderisk
brew install coderisk

# Debian/Ubuntu
curl -fsSL https://coderisk.dev/install.sh | sudo bash

# Docker
docker pull coderisk/crisk:latest
```

**2. Launch Posts**
- **Hacker News:** "Show HN: CodeRisk - Open source AI code risk assessment"
- **Reddit:** r/golang, r/programming, r/devtools
- **Twitter/X:** Thread with demo GIF
- **Product Hunt:** "Open source AI code trust infrastructure"
- **Dev.to:** "How We Built an Open Source Code Risk Tool"

**3. Community Onboarding**
- "Good first issue" labels (10 issues)
- Contributor documentation (CONTRIBUTING.md)
- Community Discord or GitHub Discussions
- Monthly contributor recognition

**4. OSS Partnerships**
- Reach out to 20 popular OSS projects
- Offer to add CodeRisk pre-commit hooks
- Showcase results in case studies

### Phase 3: Freemium Funnel (Months 2-3)

**Goal:** Convert 5% of OSS users to paid cloud

**Funnel:**
```
100 OSS users (free, local mode)
→ 20 try cloud (faster, webhooks, team sharing)
→ 5 convert to paid ($10-25/mo)
→ 1 becomes enterprise ($50/mo)

Conversion: 5% → Pays for all 100 free users
```

**Conversion Triggers:**

**1. Usage Limits (Soft Paywall)**
- Local mode: Unlimited
- Cloud free tier: 100 checks/month
- Upgrade CTA after 80 checks: "Running low on checks. Upgrade to Pro for 1,000/month."

**2. Feature Gating**
- Local mode: Phase 1 metrics only
- Cloud Pro: Phase 2 LLM investigation
- Cloud Pro: ARC database access (100+ patterns)
- Cloud Enterprise: Trust certificates, insurance

**3. Performance Advantage**
- Local mode: 5-10 min graph build
- Cloud mode: 0-2s (pre-built cache)
- Messaging: "Your repo is already cached! Switch to cloud for instant access."

**4. Team Collaboration**
- Local mode: Single user
- Cloud Pro: Team sharing (10 users, one graph)
- Messaging: "Add teammates for $25/user/month. Everyone shares the same graph."

### Phase 4: Enterprise Sales (Months 4-6)

**Goal:** 5 enterprise customers @ $50/user/month

**Enterprise Value Props:**

**1. Self-Hosted (On-Premise)**
- Deploy in customer VPC (AWS, Azure, GCP)
- Open source core + proprietary control plane
- Data never leaves customer network
- Pricing: $50/user/month minimum ($5K/month for 100 users)

**2. Trust Infrastructure**
- AI code provenance certificates
- Trust score leaderboards
- CodeRisk Verified badges
- Industry-first positioning

**3. Compliance & Security**
- SOC 2 Type II certification (target Q2 2026)
- GDPR data residency
- SAML/SSO integration
- Audit logs and retention

**Sales Motion:**
- Inbound from OSS usage → Enterprise inquiry
- Outbound to Fortune 500 with AI coding initiatives
- Partners: Anthropic (Claude Code), Cursor, GitHub

---

## Community Strategy

### Contributor Onboarding

**1. Contribution Areas (Open to Community)**

**High-Value Contributions:**
- New metrics (complexity, code smells, etc.)
- Language parsers (Rust, Elixir, Haskell, etc.)
- Documentation improvements
- Bug fixes and edge case handling
- Performance optimizations

**Community Engagement:**
- Monthly "Contributor Spotlight" on blog
- Swag for meaningful contributions
- Recognition in release notes
- Fast PR review turnaround (<48 hours)

**2. Closed Areas (Not Accepting PRs)**

**Politely Decline:**
- Cloud infrastructure code (proprietary)
- ARC database contents (competitive advantage)
- Phase 2 LLM investigation (IP protection)
- Trust infrastructure (insurance, certificates)
- Enterprise features (SSO, billing, etc.)

**Standard Response:**
> "Thank you for your interest in contributing to this area! This component is part of our commercial cloud platform and not open for external contributions. However, we'd love your input in a GitHub Discussion where we can incorporate community feedback into our roadmap."

### Community Health Metrics

**Track Monthly:**
- GitHub stars (target: 1,000 in 12 months)
- External contributors (target: 50 in 12 months)
- PRs merged from community (target: 20/month)
- GitHub Discussions activity (target: 100 posts/month)
- OSS projects using CodeRisk (target: 100 in 12 months)

**Red Flags:**
- Declining contributor count (engagement drop)
- High PR rejection rate (too restrictive)
- No external contributions (not welcoming enough)
- Negative sentiment on HN/Reddit (community backlash)

---

## Revenue Model

### Free Tier (Loss Leader)

**Costs:**
- Infrastructure: $0 (local mode, user's hardware)
- Support: Community forum (minimal cost)
- **Total:** $0

**Purpose:**
- Adoption driver
- Viral growth
- Community building
- Brand awareness

### Starter Tier ($10/user/month)

**Costs:**
- Infrastructure: $2.30/user/month (Neptune, Redis, ECS)
- Support: Email (48h response)
- **Total:** $2.30/user/month

**Margin:** 77% gross margin

**Revenue (1,000 users):** $10,000/month = $120K/year

### Pro Tier ($25/user/month)

**Costs:**
- Infrastructure: $2.30/user/month
- Team features: $0.50/user/month (amortized)
- Support: Priority email (12h response)
- **Total:** $2.80/user/month

**Margin:** 89% gross margin

**Revenue (1,000 users):** $25,000/month = $300K/year

### Enterprise Tier ($50/user/month)

**Costs:**
- Infrastructure: $2.30/user/month
- Self-hosted support: $1.00/user/month
- Dedicated support: $2.00/user/month
- **Total:** $5.30/user/month

**Margin:** 89% gross margin

**Revenue (1,000 users):** $50,000/month = $600K/year

### New Revenue Streams (Year 2)

**From Open Core Positioning:**
- **Certification:** $3.1M/year (11,000 practitioners)
- **ARC Consulting:** $500K/year (10 engagements)
- **Trust Certificates:** $400K/year (insurance product)
- **AI Tool Partnerships:** $200K/year (5 vendors @ $40K each)

**Total New Revenue:** $4.2M/year (on top of $600K-1M SaaS ARR)

---

## Risk Mitigation

### Risk 1: Competitors Fork Open Source

**Scenario:** AWS/Google fork CodeRisk → offer managed version

**Likelihood:** Low-Medium (requires significant engineering investment)

**Mitigation:**
1. **Network effects:** ARC database proprietary → our version has better predictions
2. **Brand:** "CodeRisk" owns developer mindshare
3. **Velocity:** Ship faster than forks can keep up
4. **License clarity:** MIT license allows forks, but clearly document proprietary boundaries

**Precedent:** Elastic vs AWS OpenSearch (AWS forked, Elastic still thriving)

### Risk 2: Community Rejects Hybrid Model

**Scenario:** "CodeRisk isn't truly open source" backlash

**Likelihood:** Low (common model, well-documented precedents)

**Mitigation:**
1. **Transparency:** Clearly document open vs closed in LICENSE and README
2. **Precedent:** Reference successful open core companies (GitLab, Sentry, HashiCorp)
3. **Value:** Free local mode is genuinely useful (not crippled)
4. **Messaging:** "Open source for individuals, commercial for teams/enterprises"

**Precedent:** GitLab, Sentry, HashiCorp all use this model successfully

### Risk 3: Slow Open Source Adoption

**Scenario:** Developers don't adopt local mode (prefer cloud)

**Likelihood:** Low (local mode solves real problem: privacy, cost, control)

**Mitigation:**
1. **Local mode quality:** Ensure local mode is production-ready (not second-class)
2. **Documentation:** Excellent local mode docs and tutorials
3. **Use cases:** Target privacy-sensitive orgs (finance, healthcare, gov)
4. **Marketing:** Emphasize "runs on your machine, no data leaves network"

### Risk 4: Proprietary Moat Insufficient

**Scenario:** Competitors catch up on ARC database, Phase 2 LLM

**Likelihood:** Medium (AI moves fast)

**Mitigation:**
1. **Bootstrap advantage:** GitHub mining gives 3-5 year head start
2. **Network effects:** Cross-org learning compounds advantage
3. **Velocity:** Ship ARC updates faster than competitors can replicate
4. **Open standard:** CodeRisk Trust Framework → own category definition

---

## Success Metrics (12 Months)

### Open Source Adoption

- ✅ 1,000 GitHub stars
- ✅ 50 external contributors
- ✅ 20 PRs merged/month from community
- ✅ 100 OSS projects using CodeRisk
- ✅ 10K CLI downloads/month

### Commercial Conversion

- ✅ 5% free → paid conversion rate
- ✅ 1,000 paid users ($10-50/mo)
- ✅ $600K-1M SaaS ARR
- ✅ 5 enterprise customers ($5K+/month each)

### Brand & Community

- ✅ "CodeRisk" = top result for "AI code risk assessment"
- ✅ 100+ blog posts/articles mentioning CodeRisk
- ✅ 500+ active community members (Discord/Discussions)
- ✅ 10,000+ monthly unique visitors to coderisk.dev

### Competitive Moat

- ✅ ARC database: 100+ patterns, 10K+ incidents
- ✅ Trust Framework v1.0 published (open standard)
- ✅ 1,000+ certified Trust Engineers
- ✅ 3 AI tool vendor partnerships

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - Product vision (updated with trust infrastructure)
- [strategic_moats.md](strategic_moats.md) - 7 Powers implementation plan
- [competitive_analysis.md](competitive_analysis.md) - Market positioning vs SonarQube, Greptile, etc.
- [pricing_strategy.md](pricing_strategy.md) - Free → Paid funnel

**Technical:**
- [../01-architecture/cloud_deployment.md](../01-architecture/cloud_deployment.md) - Cloud vs local architecture
- [../spec.md](../spec.md) - Complete requirements specification

**Community:**
- [../../CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines
- [../../LICENSE](../../LICENSE) - MIT License with scope clarification

---

**Last Updated:** 2025-10-12
**Next Review:** January 2026 (quarterly review after community launch)
**Recent Changes:**
- Adopted hybrid open core model (MIT License for CLI, proprietary for cloud)
- Created LICENSE, updated README.md, added CONTRIBUTING.md
- Documented open vs closed boundaries and rationale
