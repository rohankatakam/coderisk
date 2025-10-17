# Customer Acquisition Strategy

**Last Updated:** October 2, 2025
**Owner:** Product Team
**Status:** Stub (Needs Development)

> **📘 Note:** This document is a stub. Content needs to be developed based on growth experiments and channel testing.

---

## Purpose

This document will define CodeRisk's customer acquisition strategy, including:

- **Acquisition Channels**: Organic, paid, referral, partnerships
- **Conversion Funnels**: Free → Starter → Pro → Enterprise
- **Cost per Acquisition (CAC)**: By channel, by segment
- **Lifetime Value (LTV)**: By tier, by cohort
- **LTV:CAC Ratio**: Target 3:1 or better
- **Payback Period**: Target <12 months

---

## Placeholder Sections

### 1. Acquisition Channels

**To Be Defined:**

**Organic Channels:**
- **SEO/Content**: Blog posts, technical guides, comparisons ("CodeRisk vs SonarQube")
- **Community**: Hacker News, Reddit (/r/programming), Dev.to, Twitter
- **Open Source**: GitHub presence, contributor engagement, OSS repo analysis
- **Word of Mouth**: Developer referrals, team invites (virality)

**Paid Channels:**
- **Google Ads**: Search keywords ("code risk analysis", "pre-commit checks")
- **Sponsored Content**: The New Stack, InfoQ, dev newsletters
- **Conference Sponsorships**: DevOps Days, KubeCon, GitHub Universe
- **Influencer Partnerships**: YouTube tech channels, Twitter developers

**Partnership Channels:**
- **IDE Integrations**: VS Code Marketplace, JetBrains plugins
- **CI/CD Marketplaces**: GitHub Actions, GitLab CI, CircleCI
- **Reseller Partnerships**: Dev tool consultancies, system integrators

---

### 2. Conversion Funnels

**To Be Defined:**

**Free to Starter Funnel:**
```
Landing page (10,000 visitors/month)
  ↓ 10%
Sign up (1,000 users)
  ↓ 80%
Install CLI (800 users)
  ↓ 70%
First check (560 users)
  ↓ 50%
10+ checks (280 users) ← Habit formation
  ↓ 15%
Upgrade to Starter (42 paid users) ← 0.42% overall conversion

Target: 1% overall conversion (100 paid users from 10K visitors)
```

**Starter to Pro Funnel:**
```
Starter users (1,000 users)
  ↓ 30%
Invite teammate (300 users)
  ↓ 50%
2+ teammates active (150 users)
  ↓ 60%
Upgrade to Pro (90 users) ← 9% overall conversion

Target: 20% Starter → Pro conversion (200 upgrades)
```

**Pro to Enterprise Funnel:**
```
Pro teams (100 teams)
  ↓ 40%
20+ users (40 teams)
  ↓ 30%
Enterprise inquiry (12 teams)
  ↓ 50%
Enterprise contract (6 teams) ← 6% overall conversion

Target: 10% Pro → Enterprise conversion (10 contracts)
```

---

### 3. Cost per Acquisition (CAC)

**To Be Defined:**

**By Channel (Estimates):**
| Channel | CAC (Starter) | CAC (Pro) | CAC (Enterprise) | Notes |
|---------|---------------|-----------|------------------|-------|
| **Organic (SEO)** | $5 | $20 | $500 | Content creation + distribution |
| **Community (HN, Reddit)** | $2 | $10 | $200 | Time investment only |
| **Paid Ads (Google)** | $50 | $200 | $2,000 | CPC $2, 2% conversion |
| **Conferences** | $100 | $400 | $5,000 | Booth + travel + time |
| **Referral (Word of Mouth)** | $1 | $5 | $100 | Incentive program |

**Target Blended CAC:**
- Starter: $10 (10% organic, 20% community, 50% paid, 10% referral, 10% other)
- Pro: $50 (upgrades cheaper than new acquisition)
- Enterprise: $1,000 (sales-assisted, higher touch)

---

### 4. Lifetime Value (LTV)

**To Be Defined:**

**By Tier:**
| Tier | ARPU | Avg Lifespan | LTV | Notes |
|------|------|--------------|-----|-------|
| **Starter** | $10/month | 18 months | $180 | Higher churn (solo users) |
| **Pro** | $25/month | 30 months | $750 | Team stickiness |
| **Enterprise** | $50/month | 48 months | $2,400 | Contracts + switching costs |

**LTV Calculation:**
```
LTV = ARPU × Avg Lifespan × Gross Margin

Starter: $10 × 18 × 0.77 = $139
Pro: $25 × 30 × 0.89 = $667
Enterprise: $50 × 48 × 0.85 = $2,040
```

---

### 5. LTV:CAC Ratio

**To Be Defined:**

**Target Ratios:**
- **Starter**: LTV $139 / CAC $10 = **13.9:1** ✅ (excellent)
- **Pro**: LTV $667 / CAC $50 = **13.3:1** ✅ (excellent)
- **Enterprise**: LTV $2,040 / CAC $1,000 = **2:1** ⚠️ (acceptable, target 3:1)

**Industry Benchmarks:**
- SaaS standard: 3:1 (healthy), 5:1 (excellent)
- PLG companies: 10:1+ (low CAC from self-serve)
- Enterprise sales: 3:1 (higher CAC from sales team)

**Improvement Strategies:**
- Reduce Starter CAC (invest in SEO, community)
- Increase Pro LTV (reduce churn, encourage annual billing)
- Improve Enterprise CAC efficiency (refine sales process)

---

### 6. Payback Period

**To Be Defined:**

**Target Payback:**
| Tier | CAC | Monthly Profit | Payback Period | Target |
|------|-----|----------------|----------------|--------|
| **Starter** | $10 | $7.70 (77% GM) | 1.3 months | <6 months ✅ |
| **Pro** | $50 | $22.25 (89% GM) | 2.2 months | <6 months ✅ |
| **Enterprise** | $1,000 | $42.50 (85% GM) | 23.5 months | <12 months ❌ |

**Why Enterprise Payback is Long:**
- High sales touch (SDR, AE, SE, CSM)
- Long sales cycle (6-12 months)
- Multi-year contracts (amortize over 3 years)

**Mitigation:**
- Annual prepay discounts (improve cash flow)
- Upsell/cross-sell (increase ARPU over time)
- Reduce sales cycle (better qualification, faster POCs)

---

### 7. Growth Loops

**To Be Defined:**

**Loop 1: Public Cache Virality**
```
User analyzes React (public repo)
  ↓
Shares results on Twitter ("CodeRisk found X issues in React")
  ↓
Other devs try CodeRisk on React (instant access via cache)
  ↓
Some convert to paid (private repos)
  ↓
More users → More shared cache → More instant value
```

**Loop 2: Team Expansion**
```
Developer signs up (Starter)
  ↓
Invites teammate (referral)
  ↓
Team adopts (upgrades to Pro)
  ↓
More teammates join (network effects)
  ↓
Team becomes evangelists (word of mouth)
```

**Loop 3: Content-Driven Growth**
```
User finds CodeRisk via blog post ("How to prevent incidents")
  ↓
Signs up, uses product
  ↓
Writes their own blog post ("How we reduced incidents with CodeRisk")
  ↓
More users discover CodeRisk via content
  ↓
More content → More SEO → More discovery
```

---

### 8. Activation Metrics

**To Be Defined:**

**Activation Definition:**
- User runs 10+ checks within first 7 days (habit formation)

**Activation Funnel:**
```
Sign up (1,000 users)
  ↓ 80%
Install CLI (800 users)
  ↓ 70%
Run first check (560 users)
  ↓ 50%
Run 10+ checks in 7 days (280 users) ← 28% activation rate

Target: 50% activation rate (500 users)
```

**Activation Drivers:**
- Onboarding emails (Day 1, 3, 7 - encourage checks)
- CLI tips (show value: "Found 3 risks, prevented potential incident")
- Team invites (social proof: "5 teammates are using CodeRisk")

---

### 9. Retention Strategies

**To Be Defined:**

**Cohort Retention Targets:**
| Cohort Month | Retention Target | Actual (TBD) | Notes |
|--------------|------------------|--------------|-------|
| Month 1 | 90% | - | Onboarding critical |
| Month 3 | 80% | - | Habit formation |
| Month 6 | 70% | - | Long-term value |
| Month 12 | 60% | - | Mature users |

**Retention Tactics:**
- **Week 1**: Onboarding emails, CLI tips, quick wins
- **Month 1**: Usage reports ("You prevented 2 potential incidents this month")
- **Month 3**: Feature announcements (webhooks, branch deltas)
- **Month 6**: Team expansion offers (invite 5 teammates, get Pro discount)
- **Churn Risk**: Re-engagement campaigns (email, in-app prompts)

---

### 10. Expansion Revenue

**To Be Defined:**

**Expansion Opportunities:**
- **Seat Expansion**: Starter → Pro (invite teammates)
- **Tier Upgrades**: Pro → Enterprise (self-hosted VPC, SSO)
- **Usage Expansion**: More repos, more checks (usage-based pricing in future)
- **Feature Upsells**: CI/CD integration, API access, custom metrics

**Net Revenue Retention (NRR) Target:**
- 100%+ (expansion revenue exceeds churn)
- Industry benchmark: 120% (SaaS leaders)

---

## Next Steps

1. **Channel Testing**: Run small experiments (Google Ads, HN launch, conference booth)
2. **Cohort Analysis**: Track first 100 users (activation, retention, referrals)
3. **CAC/LTV Model**: Build detailed spreadsheet with assumptions
4. **Growth Playbook**: Document what works, double down on best channels
5. **Referral Program**: Design incentives (give $10, get $10?)

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - Strategic positioning and network effects
- [user_personas.md](user_personas.md) - Target users and adoption triggers
- [competitive_analysis.md](competitive_analysis.md) - Win/loss analysis
- [pricing_strategy.md](pricing_strategy.md) - Pricing tiers and conversion funnels
- [success_metrics.md](success_metrics.md) - Growth OKRs and targets
- [go_to_market.md](go_to_market.md) - Launch strategy and distribution channels

---

**Last Updated:** October 2, 2025
**Next Review:** TBD (after channel testing phase)
