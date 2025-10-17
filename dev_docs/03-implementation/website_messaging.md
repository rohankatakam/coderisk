# Website Messaging and Content Strategy

**Last Updated:** 2025-10-13
**Owner:** Product & Marketing Team
**Status:** Active - Ready for Implementation
**Phase:** Pre-Launch Website Update

> **📘 Cross-reference:** See [open_core_strategy.md](../00-product/open_core_strategy.md) for positioning, [vision_and_mission.md](../00-product/vision_and_mission.md) for product vision, and [packaging_and_distribution.md](packaging_and_distribution.md) for installation methods.

---

## Executive Summary

This document defines the website messaging, content structure, and user experience for coderisk.dev after adopting the open core model. The website must communicate three key messages:

1. **Open Source:** CLI and local mode are MIT licensed
2. **Easy to Install:** One-command setup via Homebrew
3. **Cloud Upgrade:** Optional paid platform for teams

**Goals:**
- Convert visitors to installers (30% conversion)
- Communicate open source positioning clearly
- Drive qualified leads to cloud platform
- Build developer trust through transparency

**Target Audience:**
- Individual developers (primary)
- Engineering managers (secondary)
- CTOs evaluating tools (tertiary)

---

## Core Messaging Framework

### Headline (Hero Section)

**Primary Headline:**
```
Open Source AI Code Risk Assessment
```

**Subheadline:**
```
LLM-powered agentic graph search - <3% false positives.
15-minute setup (one-time per repo). Self-hosted or cloud.
Transparent LLM costs: $0.03-0.05/check (BYOK).
```

**Why this works:**
- ✅ "Open Source" = trust signal
- ✅ "<3% false positives" = clear value prop
- ✅ "15-minute setup" = honest expectation setting
- ✅ "BYOK" (Bring Your Own Key) = transparent costs
- ✅ "Self-hosted or cloud" = deployment options

### Value Propositions (Feature Grid)

**1. Fast (Speed)**
```
2-5 second checks
Intelligent pre-commit analysis that doesn't slow you down
```

**2. Accurate (Quality)**
```
<3% false positives
Agentic graph search delivers accurate risk assessment
```

**3. Transparent Costs**
```
$3-5/month (100 checks)
BYOK model - you control LLM spend, no markup
```

**4. Open Source (Trust)**
```
MIT Licensed
Self-hosted option - full privacy, you control infrastructure
```

### Positioning Statements

**For Developers:**
> "CodeRisk is the open source tool for assessing code risk before you commit. 15-minute one-time setup per repo. Self-hosted: full privacy, you control infrastructure. LLM costs: $0.03-0.05/check (BYOK)."

**For Engineering Managers:**
> "Help your team catch architectural risks early. Self-hosted mode for individual developers ($3-5/month LLM costs), cloud platform for teams ($10-50/user/month)."

**For CTOs:**
> "Trust infrastructure for AI-generated code. Open core model ensures no lock-in. Self-host for privacy or use cloud for zero DevOps."

---

## Website Structure

### Page Hierarchy

**Primary Pages:**
1. **Homepage** (`/`) - Hero, features, installation, CTA
2. **Docs** (`/docs`) - Installation, usage, API reference
3. **Pricing** (`/pricing`) - Free vs cloud tiers
4. **Open Source** (`/open-source`) - Why open core, contribution guide
5. **About** (`/about`) - Team, mission, contact

**Secondary Pages:**
6. **Blog** (`/blog`) - Technical articles, announcements
7. **Changelog** (`/changelog`) - Release notes
8. **GitHub** (external link) - Repository

### Homepage Sections

**1. Navigation Bar**
- Logo: "CodeRisk"
- Links: Docs, Pricing, Open Source, GitHub
- CTA Button: "Get Started" (scrolls to install section)

**2. Hero Section**
- Headline + Subheadline
- Two CTAs: "Install CLI" (primary), "View Docs" (secondary)
- Quick install command: `brew install rohankatakam/coderisk/crisk`
- Open source badge: MIT License

**3. Quick Demo / Code Example**
```bash
# Before committing, run:
git add .
crisk check

# Output (example):
✅ LOW risk - Safe to commit
  - Coupling: 3/10 (acceptable)
  - Test coverage: 78% (good)
  - No incident patterns detected
```

**4. Key Features Grid**
- Fast (2-5 seconds)
- Accurate (<3% FP)
- Easy (brew install)
- Open Source (MIT)

**5. How It Works (3 Steps)**
- Step 1: Make changes (with Claude Code, Cursor, or manually)
- Step 2: Run `crisk check` before committing
- Step 3: Get instant feedback on architectural risks

**6. Open Source Section**
- "Free Forever" messaging
- What's open source (CLI, local mode, Phase 1 metrics)
- Link to GitHub repository
- "Star us on GitHub" CTA

**7. Cloud Platform Teaser**
- "Need more? Try our cloud platform"
- Benefits: Zero setup, team collaboration, ARC database
- Pricing: $10-50/user/month
- CTA: "Start Free Trial" or "Compare Plans"

**8. Installation Instructions**
- Homebrew (macOS/Linux)
- Install script (curl one-liner)
- Docker (containerized)
- Direct download (all platforms)

**9. Social Proof**
- GitHub stars count (auto-updated)
- "Trusted by developers at..." (placeholder for logos)
- Testimonials (future: real user quotes)

**10. Call to Action (Final)**
- "Start using CodeRisk today"
- Install CLI button
- Browse docs button

**11. Footer**
- Product links (Features, Pricing, Docs)
- Company links (About, Blog, Careers)
- Connect links (Twitter, GitHub, Discord)
- Legal (Privacy, Terms)

---

## Detailed Page Content

### Homepage (`/`)

**Complete content structure:**

```markdown
# Navigation
- CodeRisk (logo)
- Docs | Pricing | Open Source | GitHub
- [Get Started] button

# Hero Section
## Open Source AI Code Risk Assessment

LLM-powered agentic graph search - <3% false positives.

15-minute setup (one-time per repo). Self-hosted or cloud.
Transparent LLM costs: $0.03-0.05/check (BYOK).

[Get Started] [View Docs]

MIT Licensed • 1.2K GitHub stars • Updated daily

# Installation (17 minutes one-time per repo)

```bash
# 1. Install CLI (30 seconds)
brew install rohankatakam/coderisk/crisk

# 2. Configure API key (30 seconds)
crisk configure

# 3. Start infrastructure (2 minutes)
docker compose up -d

# 4. Initialize repository (10-15 minutes)
cd your-repo
crisk init-local

# 5. Check for risks (2-5 seconds)
crisk check
```

# Demo Output
[Terminal screenshot showing crisk check output]

# Why CodeRisk?

⚡ **Fast**
2-5 second checks
Intelligent pre-commit analysis that doesn't slow you down

✅ **Accurate**
<3% false positives
Agentic graph search delivers accurate risk assessment

💰 **Transparent Costs**
$3-5/month (100 checks)
BYOK model - you control LLM spend, no markup

🔓 **Open Source**
MIT Licensed
Self-hosted - full privacy, you control infrastructure

# How It Works

1. **Make Your Changes**
   Code normally with Claude Code, Cursor, or manual coding

2. **Run the Check**
   ```bash
   git add .
   crisk check
   ```

3. **Get Instant Feedback**
   Clear, actionable insights about risks and dependencies

# Open Source

CodeRisk CLI is **open source** (MIT License). Self-host for full privacy.

**What's included:**
- ✅ CLI tool (full source code)
- ✅ Self-hosted mode (Docker + Neo4j)
- ✅ Core metrics (coupling, co-change, test coverage)
- ✅ Pre-commit hooks
- ✅ Graph database stack

**Requirements:**
- OpenAI API key ($0.03-0.05/check)
- Docker Desktop (free)
- 17-minute one-time setup per repo

**Monthly cost:** ~$3-5 for 100 checks (just OpenAI API)

[View on GitHub] [Read License]

# Need More? Try Cloud

For teams wanting zero DevOps and advanced features:

- ⚡ Zero setup (30 seconds, no Docker or init-local)
- 👥 Team collaboration (shared graphs)
- 🎯 Pre-built public cache (React, Next.js instant access)
- 🔄 Webhooks (auto-update on push)

**Pricing:**
- Self-hosted: $3-5/month (100 checks, BYOK)
- Cloud: $10-50/user/month (includes LLM costs)

[Compare Plans] [Contact Sales]

# Installation Options

## Homebrew (Recommended)
```bash
brew tap rohankatakam/coderisk
brew install crisk
```

## Install Script
```bash
curl -fsSL https://coderisk.dev/install.sh | bash
```

## Docker
```bash
docker pull coderisk/crisk:latest
```

## Direct Download
[Download for macOS] [Linux] [Windows]

[Full Installation Guide →]

# Trusted by Developers

[GitHub Stars: 1,234] [Weekly Downloads: 5,678]

"Game changer for pre-commit checks" - Developer at Startup X
"Finally, risk assessment that doesn't cry wolf" - Dev at Tech Inc

# Ready to Get Started?

Install CodeRisk and catch risky changes before they ship.

[Install CLI] [Browse Docs]

# Footer
[Product] Features, Pricing, Docs, Changelog
[Open Source] GitHub, Contributing, License
[Company] About, Blog, Careers
[Connect] Twitter, GitHub, Discord
[Legal] Privacy Policy, Terms of Service

© 2025 CodeRisk. Open Source (MIT License).
```

### Pricing Page (`/pricing`)

**Structure:**

```markdown
# Pricing

## Self-Hosted (Local Mode)
**From $0.04/check (BYOK)**

Perfect for individual developers, small teams, and privacy-sensitive use cases.

**What's included:**
- ✅ CLI tool (MIT licensed, unlimited use)
- ✅ Self-hosted mode (Docker + Neo4j)
- ✅ Full agentic graph navigation
- ✅ <3% false positive rate
- ✅ Complete source code access
- ✅ Community support

**Requirements:**
- OpenAI API key ($0.03-0.05/check)
- Docker Desktop (free)
- 17-minute one-time setup per repo

**Monthly cost examples:**
- 100 checks: $3-5/month (just OpenAI API)
- 500 checks: $15-25/month
- 1,000 checks: $30-50/month

[Installation Guide]

---

## Cloud Platform

For teams wanting zero DevOps and advanced features.

### Starter
**$10/user/month**

- Everything in Self-Hosted
- Zero setup (30 seconds)
- No Docker or database management
- Private repositories
- 1,000 checks/month included
- Email support

[Start Free Trial]

### Pro
**$25/user/month**

- Everything in Starter
- Team collaboration (shared graphs)
- 5,000 checks/month included
- Pre-built public cache (instant React/Next.js)
- Webhooks (auto-update on push)
- Priority support

[Start Free Trial]

### Enterprise
**$50/user/month**

- Everything in Pro
- Unlimited checks
- On-premise deployment option
- SSO/SAML
- Dedicated support
- SLA guarantees
- Custom integrations

[Contact Sales]

---

## FAQs

**Is the CLI really free?**
The CLI is open source (MIT License), but requires an OpenAI API key ($0.03-0.05/check). Total cost: ~$3-5/month for 100 checks.

**What's the difference between self-hosted and cloud?**
Self-hosted runs on your machine (17-min setup, $3-5/month LLM costs). Cloud is hosted (30-sec setup, $10-50/user/month all-in).

**Can I upgrade/downgrade anytime?**
Yes, no long-term contracts. Move between self-hosted and cloud freely.

**Do you offer discounts?**
Startups (YC, <2 years): 25% off cloud plans. Self-hosted always pay-per-use.

**Can I avoid LLM costs?**
No - LLM is required for agentic graph navigation and <3% FP rate. Without LLM, you'd get 10-20% false positives (industry standard).

[View Full Pricing Details]
```

### Open Source Page (`/open-source`)

**Structure:**

```markdown
# Open Source

CodeRisk is **open core**: free CLI, optional cloud platform.

## Philosophy

We believe developer tools should be:
- ✅ **Open:** Audit the code, no black boxes
- ✅ **Transparent:** Honest about costs and requirements
- ✅ **Sustainable:** Commercial cloud funds development

## What's Open Source

**MIT Licensed (Self-Hosted Mode):**

- CLI tool (`crisk` binary + source)
- Self-hosted mode (Docker + Neo4j stack)
- Core agentic graph engine
- Full metrics (coupling, co-change, temporal patterns)
- Pre-commit hook system
- Complete documentation

**Requirements for self-hosted:**
- OpenAI API key ($0.03-0.05/check, BYOK)
- Docker Desktop (free)
- 17-minute one-time setup per repo

**Cost:** ~$3-5/month for 100 checks (just OpenAI API)

[View on GitHub →]

## What's Proprietary

**Commercial Cloud Platform:**

- Multi-tenant infrastructure
- ARC database (100+ patterns, 10K incidents)
- Phase 2 LLM investigation
- Trust infrastructure (certificates, insurance)
- Enterprise features (SSO, audit logs)

**Why?** Sustains development and supports open source maintenance.

## Contributing

We welcome contributions!

**How to contribute:**
- 🐛 Report bugs
- 💡 Suggest features
- 🔧 Submit PRs (metrics, parsers, docs)
- ⭐ Star on GitHub

[Contribution Guide →]

**Areas accepting contributions:**
- New metrics
- Language parsers
- Documentation
- Bug fixes
- Testing

**Areas closed:**
- Cloud infrastructure
- ARC database
- Phase 2 LLM
- Enterprise features

## License

CodeRisk CLI: MIT License
Cloud Platform: Proprietary

[Read Full License →]

## Community

- **GitHub:** Report issues, discussions
- **Discord:** Chat with developers
- **Twitter:** Follow updates

[Join Community →]
```

---

## Visual Design Guidelines

### Color Palette

**Primary:**
- Black: `#000000` (text, CTAs)
- White: `#FFFFFF` (background)
- Gray: `#6B7280` (secondary text)

**Accent:**
- Green: `#10B981` (success, open source badge)
- Blue: `#3B82F6` (links)
- Red: `#EF4444` (warnings, errors)

**Why minimalist:**
- ✅ Developer-focused (not consumer marketing)
- ✅ Clean, professional
- ✅ High contrast (accessibility)

### Typography

**Headings:**
- Font: Inter or System UI
- Weight: Bold (700)
- Size: 48px (h1), 32px (h2), 24px (h3)

**Body:**
- Font: Inter or System UI
- Weight: Regular (400)
- Size: 16px (body), 14px (small)

**Code:**
- Font: Menlo, Monaco, Courier New
- Weight: Regular (400)
- Size: 14px

### Components

**Buttons:**
- Primary: Black background, white text
- Secondary: White background, black border
- Size: 14px text, 12px padding

**Code Blocks:**
- Background: Light gray (`#F3F4F6`)
- Border: 1px solid `#E5E7EB`
- Border radius: 8px
- Padding: 16px

**Badges:**
- MIT License: Green background, white text
- GitHub stars: Gray background, black text
- Format: Pill shape (fully rounded)

---

## Call-to-Action Strategy

### Primary CTAs

**Install CLI** (highest priority)
- Text: "Install CLI" or "Get Started"
- Action: Scroll to installation section
- Placement: Hero, navigation, footer

**View Docs** (secondary)
- Text: "View Docs" or "Browse Docs"
- Action: Navigate to `/docs`
- Placement: Hero, navigation

### Tertiary CTAs

**Star on GitHub**
- Text: "Star on GitHub" with count
- Action: Open GitHub repository
- Placement: Open source section

**Compare Plans**
- Text: "Compare Plans" or "See Pricing"
- Action: Navigate to `/pricing`
- Placement: Cloud platform teaser

**Contact Sales**
- Text: "Contact Sales" or "Talk to Us"
- Action: Open contact form or mailto
- Placement: Enterprise tier, footer

### CTA Hierarchy

1. **Top priority:** Install CLI (free, no friction)
2. **Medium priority:** View docs (education)
3. **Low priority:** Cloud signup (qualified leads only)

**Rationale:** Drive installations first, monetize later via upgrade path.

---

## Conversion Funnel

### Landing → Installation (Primary)

**User Journey:**
1. Land on homepage
2. Read headline ("Open Source")
3. See quick install command
4. Copy command
5. Run in terminal
6. Success!

**Conversion Goal:** 30% of visitors install within 5 minutes

**Optimizations:**
- One-click copy for install command
- Platform detection (show macOS/Linux/Windows command)
- Clear "no signup required" messaging

### Landing → Cloud Signup (Secondary)

**User Journey:**
1. Install CLI
2. Use for 1-2 weeks
3. Hit free tier limits or want team features
4. Visit pricing page
5. Start free trial
6. Convert to paid

**Conversion Goal:** 5% of CLI users upgrade to cloud within 30 days

**Optimizations:**
- "Upgrade" prompts in CLI (soft, non-intrusive)
- Free trial: 14 days, no credit card
- Testimonials from teams

---

## SEO Strategy

### Target Keywords

**Primary:**
- "code risk assessment"
- "pre-commit code check"
- "AI code review"
- "open source code analysis"

**Secondary:**
- "architectural risk detection"
- "coupling analysis tool"
- "temporal code coupling"
- "pre-flight check for code"

**Long-tail:**
- "how to check code risk before commit"
- "open source alternative to SonarQube"
- "AI code risk assessment tool"

### Meta Tags

**Title:** `CodeRisk - Open Source AI Code Risk Assessment`

**Description:** `Free CLI for pre-commit code risk checks. Detect architectural risks, coupling, and incidents before they ship. MIT licensed, 2-5 second analysis.`

**Keywords:** `code risk, pre-commit, AI code review, open source, CLI tool, architectural analysis`

### Structured Data

**Organization Schema:**
```json
{
  "@type": "Organization",
  "name": "CodeRisk",
  "url": "https://coderisk.dev",
  "logo": "https://coderisk.dev/logo.png",
  "sameAs": [
    "https://github.com/rohankatakam/coderisk-go",
    "https://twitter.com/coderiskdev"
  ]
}
```

**SoftwareApplication Schema:**
```json
{
  "@type": "SoftwareApplication",
  "name": "CodeRisk",
  "applicationCategory": "DeveloperApplication",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD"
  },
  "operatingSystem": "macOS, Linux, Windows"
}
```

---

## Analytics & Tracking

### Key Metrics

**Traffic:**
- Unique visitors/month
- Page views/session
- Bounce rate
- Avg session duration

**Engagement:**
- Scroll depth (hero → install section)
- Install command copy rate
- GitHub link clicks
- Docs page visits

**Conversion:**
- CLI installations (track via analytics)
- GitHub stars (track via API)
- Cloud signups (track via backend)
- Newsletter signups

**Retention:**
- Return visitors
- CLI usage (telemetry, opt-in)
- Cloud platform usage

### Tracking Implementation

**Tools:**
- **Google Analytics 4:** Page views, events
- **PostHog:** User behavior, funnels
- **GitHub API:** Stars, forks, downloads
- **CLI telemetry:** Version, OS, usage (opt-in)

**Events to Track:**
- Install command copied
- GitHub link clicked
- Docs link clicked
- Pricing page viewed
- Cloud signup started

---

## Content Updates Required

### Current Frontend (coderisk-frontend)

**Files to Update:**

**1. `app/page.tsx` (Homepage)**
- Update hero headline to "Open Source AI Code Risk Assessment"
- Add "Free CLI, optional cloud platform" subheadline
- Update quick start command to Homebrew install
- Add MIT License badge
- Add open source section (what's free)
- Add cloud platform teaser (what's paid)
- Update feature grid (add "Open Source" card)
- Update installation instructions (Homebrew, script, Docker)

**2. Navigation**
- Add "Open Source" link
- Add "Pricing" link
- Update GitHub link

**3. Footer**
- Add "Open Source" section with links
- Add "MIT License" notice

**4. New Pages to Create:**
- `app/pricing/page.tsx` - Pricing tiers
- `app/open-source/page.tsx` - Open core explanation
- `app/docs/page.tsx` - Documentation hub (or link to external docs)

**5. Public Assets:**
- `public/install.sh` - Install script
- Update `public/favicon.ico` if needed
- Add open source badges

### Messaging Changes

**Before:**
```
Trust infrastructure for AI-generated code
```

**After:**
```
Open Source AI Code Risk Assessment
The pre-flight check for developers. Free CLI, optional cloud.
```

**Rationale:** Lead with "open source" to build trust immediately.

---

## Implementation Checklist

### Phase 1: Content Updates (Week 1)

- [ ] Update homepage hero section
- [ ] Add open source section
- [ ] Add cloud platform teaser
- [ ] Update installation instructions
- [ ] Add MIT License badge
- [ ] Update navigation (add Open Source, Pricing)
- [ ] Update footer (add license notice)

### Phase 2: New Pages (Week 1-2)

- [ ] Create pricing page (`/pricing`)
- [ ] Create open source page (`/open-source`)
- [ ] Create docs hub (`/docs`) or link to external
- [ ] Add `install.sh` to public directory

### Phase 3: Visual Polish (Week 2)

- [ ] Add code block styling
- [ ] Add badge components (MIT License, GitHub stars)
- [ ] Optimize responsive design
- [ ] Add loading states for dynamic content (GitHub stars)

### Phase 4: Analytics (Week 2)

- [ ] Set up Google Analytics 4
- [ ] Add event tracking (install command copy, GitHub clicks)
- [ ] Set up conversion goals
- [ ] Test tracking on staging

### Phase 5: SEO (Week 2)

- [ ] Update meta tags (title, description)
- [ ] Add structured data (Organization, SoftwareApplication)
- [ ] Submit sitemap to Google Search Console
- [ ] Verify Open Graph tags (social sharing)

### Phase 6: Launch (Week 3)

- [ ] Deploy to production (Vercel)
- [ ] Verify all links work
- [ ] Test install script (`coderisk.dev/install.sh`)
- [ ] Monitor analytics
- [ ] Announce on social media

---

## Related Documents

**Product & Strategy:**
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Open source positioning
- [vision_and_mission.md](../00-product/vision_and_mission.md) - Product vision
- [pricing_strategy.md](../00-product/pricing_strategy.md) - Pricing tiers

**Implementation:**
- [packaging_and_distribution.md](packaging_and_distribution.md) - Installation methods
- [LICENSE](../../LICENSE) - MIT License text
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines

**Frontend Repository:**
- `rohankatakam/coderisk-frontend` - Next.js app

---

**Last Updated:** 2025-10-13
**Next Review:** After launch (feedback from users)
**Next Steps:** Implement Phase 1 (content updates)
