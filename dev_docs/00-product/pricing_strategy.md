# Pricing Strategy

**Last Updated:** October 2, 2025
**Owner:** Product Team
**Status:** Active

> **📘 Cross-reference:** See [spec.md](../spec.md) Sections 4.2, 5.6, and 6.2 for technical cost model details

---

## Executive Summary

CodeRisk's pricing is built on a **Bring Your Own Key (BYOK) model**—users provide their LLM API keys (OpenAI/Anthropic), and we charge only for infrastructure ($2.30/user/month at 1K users). This creates **70-85% cost savings** vs competitors who bundle LLM costs with markup.

**Core Pricing Principles:**
1. **Transparent LLM Costs**: User pays $0.03-0.05/check directly to OpenAI/Anthropic (no markup)
2. **Infrastructure Only**: We charge for Neptune, Postgres, Redis, compute
3. **Network Effects**: Shared public cache reduces costs for everyone
4. **Usage-Based Safety**: Budget caps prevent runaway costs

**Result:** Total cost of $13-15/month (100 checks) vs competitors' $30-100/month

---

## Pricing Tiers

### Overview Table

| Tier | Price | Target User | Repos | Checks/Month | Key Features |
|------|-------|-------------|-------|--------------|--------------|
| **Free** | $0 | OSS contributors | Public only | 100 | Public cache access, basic metrics |
| **Starter** | $10/user/month | Individual developers | Unlimited | 1,000 | Private repos, all metrics, API access |
| **Pro** | $25/user/month | Small teams (5-20) | Unlimited | 5,000 | Team sharing, webhooks, priority support |
| **Enterprise** | $50/user/month | Large orgs (100+) | Unlimited | Unlimited | Self-hosted VPC, SLA, audit logs |

**Note:** All tiers require user to provide their own LLM API key (OpenAI or Anthropic)

---

## Tier Details

### 1. Free Tier (OSS Contributors)

**Price:** $0/month

**Target User:**
- Open source contributors
- Developers evaluating CodeRisk
- Students and hobbyists

**Limitations:**
- Public repositories only (React, Kubernetes, Next.js, etc.)
- 100 checks/month
- Community support only
- Shared public cache (instant access if repo pre-built)

**Value Proposition:**
> "Try CodeRisk risk-free on any public repository. If the repo is popular (React, Next.js), you get instant access via shared cache."

**Conversion Strategy:**
- First 100 checks show value (habit formation)
- "Upgrade to Starter for private repos" CTA after 80 checks
- Friction-free: No credit card required

**Cost to CodeRisk:**
- $0.05-0.30/popular repo/month (amortized across all users)
- Example: React cache accessed by 1,000 users = $0.30/month total ($0.0003/user)

---

### 2. Starter Tier (Individual Developers)

**Price:** $10/user/month (billed annually: $100/year)

**Target User:**
- Individual developers at startups
- Freelancers working on private client repos
- Solo founders

**Features:**
- ✅ Unlimited private repositories
- ✅ 1,000 checks/month (~50/day)
- ✅ All Tier 1 + Tier 2 metrics (coupling, co-change, test ratio, ownership, incidents)
- ✅ CLI + API access
- ✅ Email support (48-hour response)
- ✅ Budget caps: $0.05/check (user's LLM API)

**Value Proposition:**
> "Pre-flight check for your private repos. $10/month + your LLM costs (~$3-5/month) = $13-15/month total."

**Comparison to Competitors:**
- **vs Greptile ($30-70/user/month)**: 50-85% cheaper
- **vs Codescene ($50-100/user/month)**: 80-90% cheaper
- **vs SonarQube Pro ($10/user/month)**: Similar price, but complementary (architecture vs security)

**Cost to CodeRisk:**
- Infrastructure: $2.30/user/month (Neptune, Postgres, Redis, compute)
- Gross margin: 77% (($10 - $2.30) / $10)

**Upgrade Path:**
- "Invite team members" CTA (upgrade to Pro for team sharing)
- "Need webhooks?" (upgrade to Pro for auto-updates)

---

### 3. Pro Tier (Small Teams)

**Price:** $25/user/month (billed annually: $250/user/year)

**Target User:**
- Small engineering teams (5-20 people)
- Startups with multiple repos
- Teams wanting auto-graph updates

**Features:**
- ✅ Everything in Starter
- ✅ 5,000 checks/month per user (~250/day)
- ✅ **Team sharing**: One graph shared across team (90% cost reduction)
- ✅ **Webhooks**: Auto-update graph on push to main (incremental, <30s)
- ✅ **Branch deltas**: On-demand branch analysis (98% smaller graphs)
- ✅ Priority email support (12-hour response)
- ✅ Team admin portal (usage analytics, budget management)
- ✅ Metric validation dashboard (FP rate tracking)

**Value Proposition:**
> "Your whole team shares one graph. Build once, everyone benefits. Webhooks keep it fresh automatically."

**Comparison to Competitors:**
- **vs Greptile Team ($70/user/month)**: 64% cheaper ($25 vs $70)
- **vs Codescene ($100/user/month)**: 75% cheaper
- **Team savings**: 10-person team = $250/month (not $1000/month like competitors)

**Cost to CodeRisk:**
- Infrastructure: $2.30/user/month (same as Starter, but amortized graph costs)
- Team graph: $5/team/month (one-time build + incremental updates)
- Gross margin: 86% (($25 - $2.30 - $0.50) / $25)

**Upgrade Path:**
- "Need self-hosted VPC?" (upgrade to Enterprise)
- "Want SLA guarantees?" (upgrade to Enterprise)

---

### 4. Enterprise Tier (Large Organizations)

**Price:** $50/user/month (custom contracts for 100+ users)

**Target User:**
- Large enterprises (100+ developers)
- Companies with compliance requirements (HIPAA, SOC2, GDPR)
- Organizations wanting self-hosted option

**Features:**
- ✅ Everything in Pro
- ✅ Unlimited checks/month
- ✅ **Self-hosted VPC deployment**: Neptune in customer's AWS account (100% data residency)
- ✅ **SLA**: 99.9% uptime guarantee
- ✅ **SSO integration**: SAML, Okta, Azure AD
- ✅ **Audit logs**: Full activity tracking for compliance
- ✅ **Dedicated support**: Slack channel, 1-hour response SLA
- ✅ **Custom hop limits**: Configure agent investigation depth (3-10 hops)
- ✅ **API rate limits**: Custom quotas for CI/CD integration
- ✅ **Training & onboarding**: Dedicated customer success manager

**Value Proposition:**
> "Enterprise-grade deployment with 100% data residency. Your code never leaves your VPC."

**Comparison to Competitors:**
- **vs Codescene Enterprise ($150/user/month)**: 67% cheaper
- **vs SonarQube Enterprise ($150/user/month)**: 67% cheaper
- **100-user org savings**: $5K/month vs $15K/month (Codescene)

**Cost to CodeRisk:**
- Infrastructure: $0 (customer manages their own VPC)
- Support: $5K-10K/month (dedicated CSM, Slack support)
- Gross margin: 80%+ (minimal infrastructure costs)

**Custom Pricing:**
- Volume discounts for 500+ users
- Multi-year contracts (10-20% discount)
- Academic/non-profit discounts (50% off)

---

## Cost Breakdown (User Perspective)

### Example: Typical Startup Developer (100 checks/month)

**CodeRisk Costs:**
- Starter plan: $10/month
- LLM costs (user's API key): 100 checks × $0.04/check = $4/month
- **Total: $14/month**

**Competitor Comparison:**
- Greptile: $30-70/month (LLM costs bundled with markup)
- Codescene: $50-100/month (no LLM)
- SonarQube Pro: $10/month (no LLM, rules-based)

**Savings: 50-80% vs competitors**

---

### Example: Mid-Size Team (10 developers, 500 checks/month each)

**CodeRisk Costs:**
- Pro plan: $25/user × 10 = $250/month
- LLM costs (user's API keys): 10 × 500 × $0.04 = $200/month
- **Total: $450/month ($45/user/month all-in)**

**Competitor Comparison:**
- Greptile Team: $70/user × 10 = $700/month
- Codescene: $100/user × 10 = $1,000/month

**Savings: 35-55% vs competitors**

---

### Example: Enterprise (100 developers, unlimited checks)

**CodeRisk Costs:**
- Enterprise plan: $50/user × 100 = $5,000/month
- Self-hosted VPC: Customer manages ($500-1K/month AWS infra)
- LLM costs: Variable (user's API keys), assume $10/user/month = $1,000/month
- **Total: $6,000-6,500/month ($60-65/user/month all-in)**

**Competitor Comparison:**
- Codescene Enterprise: $150/user × 100 = $15,000/month
- SonarQube Enterprise: $150/user × 100 = $15,000/month

**Savings: 57-60% vs competitors**

---

## BYOK (Bring Your Own Key) Model

### How It Works

1. **User Signs Up**: Create account at app.coderisk.dev
2. **Add API Key**: Paste OpenAI or Anthropic API key in settings portal (AES-256-GCM encrypted)
3. **Run Check**: `crisk check` uses user's key for LLM calls
4. **Billing**: User sees LLM costs on OpenAI/Anthropic dashboard, infrastructure costs on CodeRisk invoice

**Security:**
- API keys encrypted at rest (AES-256-GCM)
- Never logged or exposed
- Stored in Postgres with column-level encryption
- Key rotation UI (regenerate without data loss)

### Why BYOK?

**Transparency:**
- User sees exactly what they pay for LLM calls ($0.03-0.05/check)
- No markup (competitors charge 2-3x)
- User controls LLM spend (can pause, set limits)

**Flexibility:**
- User can switch LLM providers (OpenAI GPT-4o, Anthropic Claude 3.5 Sonnet)
- User can use org-negotiated LLM pricing (enterprise discounts)
- User can optimize costs (use cheaper models for Phase 1)

**Economics:**
- We avoid LLM cost risk (no unexpected usage spikes)
- Sustainable unit economics (70-85% gross margin)
- Competitive pricing (can undercut bundled competitors)

### Budget Caps (User Protection)

**Problem:** Runaway LLM costs if agent gets stuck in loop

**Solution:** Configurable budget caps per check/day/month

**Default Limits:**
- **Per-check cap**: $0.05 (agent stops at hop 5 if exceeded)
- **Daily cap**: $2 (40 checks max)
- **Monthly cap**: $60 (1,200 checks max)

**User Configuration:**
```yaml
# .crisk/config.yml
budget:
  per_check: 0.05  # Stop agent after $0.05 LLM spend
  daily: 2.00      # Max $2/day on LLM calls
  monthly: 60.00   # Max $60/month on LLM calls
```

**Alerts:**
- Email when reaching 80% of any cap
- Agent stops gracefully when cap hit (returns partial results)
- User can adjust caps in settings portal

---

## Infrastructure Cost Model (CodeRisk Perspective)

### Cost Per User (at 1K users)

| Component | Cost/Month | % of Total |
|-----------|------------|------------|
| **Neptune Serverless** | $545 | 48% |
| **PostgreSQL (RDS)** | $350 | 31% |
| **Redis (ElastiCache)** | $200 | 18% |
| **Compute (EKS)** | $30 | 3% |
| **Total** | **$1,125** | 100% |

**Per-User Cost:** $1,125 / 1,000 users = **$1.13/user/month**

**Wait, why did spec.md say $2.30/user?**
- The $1.13 is infrastructure only
- Add: Support ($0.50/user), monitoring ($0.30/user), overhead ($0.37/user)
- **Total operational cost: $2.30/user/month**

### Gross Margin by Tier

| Tier | Price | Cost | Gross Margin |
|------|-------|------|--------------|
| **Free** | $0 | $0.30 (amortized) | N/A (loss leader) |
| **Starter** | $10 | $2.30 | **77%** |
| **Pro** | $25 | $2.80 | **89%** |
| **Enterprise** | $50 | $5-10 (support) | **80%** |

**Target Gross Margin: 75-85%** (SaaS industry standard is 70-80%)

### Scaling Economics

| Users | Infrastructure Cost | Revenue (Starter avg) | Gross Margin |
|-------|---------------------|----------------------|--------------|
| 100 | $230/month | $1,000/month | 77% |
| 1,000 | $2,300/month | $10,000/month | 77% |
| 10,000 | $23,000/month | $100,000/month | 77% |

**Key Insight:** Gross margin stays consistent as we scale (infrastructure scales linearly)

---

## Public Cache Economics

### Problem: Graph Build Costs

**Without shared cache:**
- 1,000 users analyzing React = 1,000 × 5-10 min builds = 83-167 hours compute
- 1,000 × 155MB graphs = 155GB storage
- **Cost: $150-300/month**

**With shared cache:**
- First user triggers build (5-10 min)
- Subsequent 999 users get instant access (0-2s)
- **Cost: $0.30/month** (99% reduction)

### Reference Counting & Garbage Collection

**Problem:** Unused repos waste storage

**Solution:** Track reference count (active users per repo)

**Lifecycle:**
1. **Active** (ref_count > 0): Keep in Neptune hot tier
2. **Warm** (ref_count = 0, <30 days): Move to Neptune scalable tier
3. **Archived** (ref_count = 0, 30-90 days): Move to S3 ($0.01/GB/month)
4. **Deleted** (ref_count = 0, >90 days): Delete from S3

**Cost Savings:**
- Neptune hot: $3.50/GB/month
- Neptune scalable: $0.10/GB/month
- S3 archive: $0.01/GB/month
- **70-99% storage cost reduction for inactive repos**

### Popular Repo Economics

**Assumption:** Top 100 OSS repos (React, Next.js, Kubernetes, etc.) cover 80% of usage

**Cost:**
- 100 repos × 155MB avg × $3.50/GB/month = $54/month
- Accessed by 5,000+ users = **$0.01/user/month**

**Business Model:**
- Free tier users access public cache at no cost
- We subsidize popular repos ($54/month) as marketing expense
- Conversion to paid tiers pays for public cache 100x over

---

## Pricing Psychology

### Anchoring Strategy

**Problem:** $10/month feels expensive without context

**Solution:** Anchor against competitors and total cost

**Messaging:**
```
CodeRisk Starter: $10/month + LLM ($4/month) = $14/month
vs
Greptile: $30/month (all-in, LLM markup)
Codescene: $50/month (no LLM)

Save 50-70% with CodeRisk
```

### Decoy Pricing (Pro Tier)

**Strategy:** Make Pro ($25) look attractive by comparison

| Tier | Price | Checks/Month | Value/Check |
|------|-------|--------------|-------------|
| Starter | $10 | 1,000 | $0.01 |
| **Pro** | **$25** | **5,000** | **$0.005** (50% better) |
| Enterprise | $50 | Unlimited | - |

**Psychology:** Pro offers 5× checks for 2.5× price → "better value"

### Loss Aversion (Free to Starter)

**Problem:** Users hesitate to start paying

**Solution:** Show what they'll lose

**Messaging:**
```
You've used 95 of 100 free checks this month.

Without upgrading:
❌ Can't check private repos
❌ No team sharing
❌ Limited to 100 checks/month

Upgrade to Starter ($10/month):
✅ Unlimited private repos
✅ 1,000 checks/month (10×)
✅ Priority support
```

---

## Competitive Pricing Comparison

### Feature-Normalized Pricing

| Product | Pre-Commit | Agentic Search | Graph DB | LLM Cost | Price/Month |
|---------|------------|----------------|----------|----------|-------------|
| **CodeRisk Starter** | ✅ | ✅ | ✅ | User BYOK | **$10 + $4 = $14** |
| **Greptile** | ❌ | ⚠️ RAG | ⚠️ Vector | Bundled | **$30-70** |
| **Codescene** | ❌ | ❌ | ❌ | N/A | **$50-100** |
| **SonarQube Pro** | ⚠️ IDE | ❌ | ❌ | N/A | **$10** |

**Value Proposition:**
- CodeRisk = SonarQube price + Greptile intelligence + Codescene insights
- **50-85% cheaper than alternatives**

### Price Elasticity

**Hypothesis:** Developers are price-sensitive, enterprises are not

**Strategy:**
- **Starter ($10)**: Price-sensitive individuals (maximize volume)
- **Pro ($25)**: Small teams (balance volume + margin)
- **Enterprise ($50)**: Large orgs (maximize margin, low price sensitivity)

**Evidence:**
- SonarQube has 7M users on free tier, 50K on paid (0.7% conversion)
- GitHub Copilot has 1M+ users at $10/month (high volume, low price)
- Datadog Enterprise charges $100-500/user (low volume, high margin)

**CodeRisk Target:**
- 1M users on free tier (OSS)
- 10K users on Starter ($100K/month revenue)
- 1K users on Pro ($25K/month revenue)
- 100 users on Enterprise ($5K/month revenue)
- **Total: $130K/month at scale**

---

## Discounts & Promotions

### Launch Discount (First 1,000 Users)

**Offer:** 50% off first year (Starter $5/month, Pro $12.50/month)

**Goal:** Rapid adoption, habit formation

**Cost:** $60K/year revenue deferred (worth it for early traction)

### Annual Billing Discount

**Offer:** 2 months free (17% discount)

**Pricing:**
- Starter: $100/year (vs $120 monthly)
- Pro: $250/year (vs $300 monthly)
- Enterprise: Custom (10-20% off)

**Goal:** Cash flow predictability, reduce churn

### Team Discounts (Volume Pricing)

| Team Size | Discount | Effective Price (Pro) |
|-----------|----------|----------------------|
| 5-10 users | 0% | $25/user |
| 11-25 users | 10% | $22.50/user |
| 26-50 users | 20% | $20/user |
| 51-100 users | 30% | $17.50/user |
| 100+ users | Custom | Contact sales |

**Goal:** Incentivize team adoption, land-and-expand

### Academic & Non-Profit

**Offer:** 50% off all tiers

**Eligibility:**
- .edu email domain
- Verified non-profit (501(c)(3), charity registration)

**Goal:** Goodwill, future talent pipeline

---

## Pricing Evolution Roadmap

### Phase 1: MVP (Current)

**Tiers:** Free, Starter ($10), Pro ($25), Enterprise ($50)

**Focus:** Validate BYOK model, optimize infrastructure costs

**Success Metric:** 1,000 paid users, 75%+ gross margin

### Phase 2: Usage-Based (2026)

**Model:** Base fee + usage overage

**Example:**
- Starter: $10/month (includes 1,000 checks) + $0.01/check over
- Pro: $25/month (includes 5,000 checks) + $0.005/check over

**Rationale:** Heavy users pay more, light users pay less (fairer)

### Phase 3: Platform Pricing (2027)

**New Tier:** API-only access for tool builders

**Example:**
- API Platform: $100/month (10K API calls) + $0.01/call over
- Target: IDE extensions (VS Code, JetBrains), CI/CD integrations

**Goal:** Become infrastructure for other dev tools

---

## Pricing FAQs

### Why BYOK instead of bundling LLM costs?

**Answer:**
- **Transparency**: You see exactly what you pay for LLM calls (no markup)
- **Control**: You set budget caps, choose models, pause anytime
- **Savings**: Competitors mark up LLM costs 2-3× (we don't)

### What if I don't have an OpenAI/Anthropic account?

**Answer:**
- We'll guide you through signup (takes 2 minutes)
- OpenAI charges $0.03-0.05/check (affordable)
- Free tier works without API key (public repos, heuristic-based)

### Can I use my organization's enterprise LLM pricing?

**Answer:**
- Yes! BYOK means you can use org-negotiated rates
- Example: Your company has OpenAI enterprise contract → use that API key
- Massive savings for large orgs

### What happens if I hit my budget cap?

**Answer:**
- Agent stops gracefully, returns partial results
- You get email alert at 80% of cap
- You can adjust caps in settings portal (increase or pause checks)

### Do I pay for failed checks?

**Answer:**
- No! We only charge if agent completes investigation
- LLM costs are only for successful checks (you pay OpenAI/Anthropic, not us)
- Infrastructure costs are flat (no per-check fee)

### Can I downgrade tiers mid-month?

**Answer:**
- Yes, downgrade anytime (prorated refund)
- Team data stays intact (graph preserved)
- Can re-upgrade without losing history

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - Product vision and BYOK rationale
- [competitive_analysis.md](competitive_analysis.md) - Pricing vs competitors
- [user_personas.md](user_personas.md) - Willingness to pay by persona

**Technical:**
- [spec.md](../spec.md) - Cost model details (Sections 4.2, 5.6, 6.2)
- [01-architecture/cloud_deployment.md](../01-architecture/cloud_deployment.md) - Infrastructure costs

---

**Last Updated:** October 2, 2025
**Next Review:** January 2026 (quarterly pricing review)
