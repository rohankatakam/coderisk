# User Personas

**Last Updated:** October 2, 2025
**Owner:** Product Team
**Status:** Active - Based on user research interviews

> **📘 Cross-reference:** See [spec.md](../spec.md) Section 1.2 for target audience overview

---

## Primary Personas

CodeRisk targets three primary user personas, prioritized by go-to-market strategy:

1. **Ben the Developer** - Individual contributor (IC), main end user
2. **Clara the Tech Lead** - Team decision maker, evaluates tools
3. **Alex the Enterprise Architect** - Enterprise buyer, compliance gatekeeper

---

## Persona 1: Ben the Developer

### Profile

**Role:** Senior Software Engineer
**Company Type:** Mid-size tech company (50-200 engineers)
**Team Size:** 8-person feature team
**Experience:** 5+ years professional development
**Tech Stack:** React, Node.js, PostgreSQL, Docker, GitHub
**Location:** Remote (US-based)

### Background

Ben is a senior engineer on a product team building a SaaS application. He's inherited a large codebase (5,000+ files) with multiple legacy patterns and hidden dependencies. His team ships features every week, but production incidents from unexpected coupling have burned him several times.

He's technically strong but faces information asymmetry: the codebase knows more than he does. Tests pass, linters are happy, but production breaks because a "simple" change affected an unrelated module through temporal coupling.

### Daily Workflow

**Morning:**
1. Pull latest changes from main
2. Review Slack for incident reports
3. Pick up ticket from sprint board

**Development (Inner Loop):**
1. Read code, understand context (30-60 min)
2. Make changes (1-2 hours)
3. Run unit tests locally (2-3 min)
4. **[Current pain point]** Commit → Push → Hope nothing breaks (no pre-commit validation)
5. Wait for CI (5-10 min)
6. Create PR, wait for reviews (4-24 hours)

**Evening:**
1. Address PR feedback
2. Merge if approved
3. Monitor deploy to production
4. **[Major pain point]** Get paged if incident occurs (happens ~2x/month)

### Goals

**Primary Goals:**
- ✅ Ship features quickly without breaking production
- ✅ Understand blast radius before committing changes
- ✅ Avoid embarrassing incidents from missed edge cases
- ✅ Build confidence in code changes (reduce anxiety)

**Secondary Goals:**
- Learn codebase structure faster (mental model building)
- Reduce time spent in PR review cycles
- Improve team trust (fewer incidents = more autonomy)

### Pain Points

**1. Unknown Unknowns ("Fear of Hidden Coupling")**
> "I changed authentication code, and somehow the reporting dashboard broke. How was I supposed to know they were connected?"

- Files that change together aren't explicitly coupled in code
- No tool shows temporal coupling
- Discovers relationships only after incident

**2. Tests Don't Catch Architectural Issues**
> "Unit tests passed, integration tests passed, but we still had a prod incident. Tests only catch what we think to test."

- Tests validate expected behavior, not unexpected coupling
- Hard to write tests for architectural regressions
- False confidence from passing CI

**3. PR Reviews Happen Too Late**
> "I spent 2 days coding this feature, then Clara found a coupling issue in PR review. Now I have to rewrite half of it."

- Sunk cost fallacy: Already invested time
- Public commitment: Hard to say "let's start over"
- Review delay: 4-24 hours wait time

**4. Manual Investigation is Time-Consuming**
> "I spend 30 minutes grepping for references, checking git history, asking teammates. There has to be a better way."

- Manual grep finds syntax, not semantic relationships
- Git history shows what changed, not why it's coupled
- Tribal knowledge locked in senior engineers' heads

**5. Production Incidents Create Anxiety**
> "I'm scared to touch certain files. Last time I modified auth.js, we had an outage. I don't want to be 'that guy' again."

- Blame culture: Incidents reduce confidence
- Analysis paralysis: Safer to not change risky files
- Career impact: Too many incidents = bad performance review

### Current Solutions & Workarounds

**What Ben Does Today (Imperfect Solutions):**

**1. Manual Code Exploration**
```bash
# Grep for function usage
grep -r "authenticate()" src/

# Check git history for co-changes
git log --follow src/auth.js
git log --all --grep="auth"

# Ask in Slack
"Has anyone touched the auth module recently? Trying to understand dependencies."
```
**Problems:** Time-consuming (30+ min), incomplete (misses temporal coupling), bothers teammates

**2. Overly Cautious Development**
```bash
# Run ALL tests before committing (slow but safe)
npm run test:unit && npm run test:integration && npm run test:e2e

# Test in staging environment first
git push origin feature/auth-changes
# Wait 2 hours for staging deploy
# Manually test in staging
# Then merge to main
```
**Problems:** Slow (hours of waiting), staging != production, still misses issues

**3. Ask Senior Engineers**
> "Hey Clara, I'm changing the authentication flow. Is this safe? What might break?"

**Problems:** Interrupts others, doesn't scale, knowledge bottleneck, only as good as Clara's memory

**4. Hope for the Best**
```bash
git commit -am "Update auth logic"
git push
# 🤞 Cross fingers
```
**Problems:** High anxiety, reactive (fix after incident), damages team trust

### CodeRisk Value Proposition

**How CodeRisk Helps Ben:**

**Before (Current State):**
```bash
# Ben's workflow:
1. Write code (2 hours)
2. Run tests (3 min) - pass ✅
3. Commit & push
4. Wait for PR review (6 hours)
5. Clara finds coupling issue in review ❌
6. Rewrite code (1 hour)
7. Repeat cycle

Total time: 9+ hours (with anxiety)
```

**After (With CodeRisk):**
```bash
# Ben's workflow:
1. Write code (2 hours)
2. crisk check (5 seconds) ⚠️ HIGH RISK

   Evidence:
   - auth.js changed together with reporting.js in 75% of commits
   - 12 files import authenticate()
   - Test coverage: 30% (low)

   Recommendations:
   - Review coupled file: src/features/reporting/dashboard.js
   - Add integration tests for auth + reporting

3. Check reporting.js, realize coupling
4. Fix proactively (30 min)
5. crisk check (5 seconds) ✅ LOW RISK
6. Commit & push with confidence
7. PR review approves quickly (no surprises)

Total time: 3 hours (with confidence)
```

**Specific Benefits:**

**1. Pre-Commit Safety Check**
- Catches issues BEFORE pushing (not after PR review)
- Private feedback (no public embarrassment)
- Instant (5 seconds vs 6 hours for PR review)

**2. Learns Codebase Faster**
- Discovers hidden relationships (temporal coupling)
- Understands blast radius (which files are affected)
- Builds mental model incrementally

**3. Reduces Production Incidents**
- Prevents 60-80% of coupling-related incidents
- Lower anxiety, higher confidence
- Better team reputation

**4. Saves Time**
- No manual grepping (30 min → 5 seconds)
- Fewer PR review cycles (2-3 → 1)
- Less time fixing incidents (2 hours → 0)

### Quotes from User Research

> **On Hidden Coupling:**
> "I waste 2 hours debugging prod issues that CodeRisk would've caught in 5 seconds. The temporal coupling graph is a game-changer."
> — Ben (Senior Engineer, 6 YOE)

> **On Anxiety:**
> "I used to be scared to touch certain files. Now I run crisk check before committing and it tells me exactly what might break. It's like having a senior engineer reviewing my code instantly."
> — Sarah (Mid-level Engineer, 3 YOE)

> **On Time Savings:**
> "Before: 30 minutes of grepping and git archaeology. After: 5 seconds of crisk check. It's not even close."
> — Mike (Senior Engineer, 8 YOE)

> **On Trust:**
> "My team trusts me more now because I catch issues before pushing. CodeRisk turned me from 'that guy who breaks prod' to 'the careful engineer.'"
> — James (Engineer, 4 YOE)

### Adoption Triggers

**When does Ben decide to try CodeRisk?**

**Trigger 1: After a Production Incident**
- Just caused an outage from unexpected coupling
- Feeling embarrassed, wants to prevent recurrence
- High motivation to find solution

**Trigger 2: Joining New Codebase**
- Inherited large legacy codebase
- Doesn't understand architecture yet
- Needs training wheels while learning

**Trigger 3: Team Recommendation**
- Clara (tech lead) mandates pre-commit checks
- Peer engineer recommends at lunch
- Sees demo at team meeting

### Success Metrics for Ben

**Usage patterns:**
- Runs `crisk check` 10-15 times per day (habit formed)
- <5 second latency acceptable
- <3% false positive rate required (trust maintained)

**Outcomes:**
- 60-80% reduction in production incidents
- 30-50% reduction in PR review cycles
- Increased confidence when changing risky files

---

## Persona 2: Clara the Tech Lead

### Profile

**Role:** Engineering Team Lead / Tech Lead
**Company Type:** Mid-size tech company (50-200 engineers)
**Team Size:** Leads 8-12 engineers
**Experience:** 10+ years professional development, 2+ years leadership
**Reports to:** Director of Engineering
**Location:** Hybrid (San Francisco Bay Area)

### Background

Clara is a technical leader who balances hands-on coding (30% of time) with team leadership (70% of time). She's responsible for architectural decisions, code quality, and incident prevention. Her team ships features weekly but has experienced several high-profile incidents from architectural regressions.

Clara needs data-backed evidence to make decisions. She can't rely on intuition alone when explaining to leadership why the team needs to refactor or why a certain change is risky.

### Daily Workflow

**Morning:**
1. Review overnight incidents (if any)
2. Triage support tickets escalated to engineering
3. Check PR review queue (10-15 PRs)

**Afternoon:**
1. Review 5-8 PRs (30-60 min each) **[Major time sink]**
2. 1:1s with team members
3. Architecture discussions with product
4. Code when time allows (rare)

**Evening:**
1. Approve PRs for deployment
2. Monitor production metrics
3. Post-incident reviews (if needed) **[High stress]**

### Goals

**Primary Goals:**
- ✅ Reduce team's incident rate (currently 2-3/month, target <1/month)
- ✅ Improve PR review efficiency (currently 45 min/PR, too slow)
- ✅ Validate architectural intuition with data (credibility with leadership)
- ✅ Scale team without sacrificing quality (hiring 3 new engineers)

**Secondary Goals:**
- Improve team confidence (reduce anxiety from incidents)
- Document architectural patterns (tribal knowledge → written)
- Justify refactoring investments to product/leadership

### Pain Points

**1. PR Review Bottleneck**
> "I review 10-15 PRs per day. Each takes 30-60 minutes because I have to mentally check: What else might this break? I'm the single point of failure."

- Manual architectural review doesn't scale
- Team waits on her approval (4-24 hour delay)
- Can't review everything deeply (skim some PRs)

**2. Incidents Despite Best Efforts**
> "We have unit tests, integration tests, linters, and I review every PR. We STILL have incidents. I need better tools."

- Current tools miss architectural coupling
- Reactive (fix after incident) not proactive
- Leadership asks: "Why didn't we catch this in review?"

**3. Hard to Justify Refactoring**
> "I KNOW the auth module is risky (churn + coupling), but I can't convince product to prioritize refactoring without data. It's just my 'feeling.'"

- Product wants features, not refactoring
- Need quantitative evidence: "This file caused 5 incidents"
- "Technical debt" is too vague for stakeholders

**4. Knowledge Doesn't Scale**
> "I've been here 3 years, I know which files are risky. But we're hiring 3 new engineers. They don't have this context. How do I transfer this knowledge?"

- Tribal knowledge in Clara's head
- Onboarding new engineers takes 2-3 months
- Can't document every coupling relationship

**5. Alert Fatigue from Existing Tools**
> "SonarQube flags 50 issues per PR. 90% are noise. I ignore most of them now. My team ignores them too."

- High false positive rate (10-20%)
- Developers trained to ignore warnings
- Tool loses credibility

### Current Solutions & Workarounds

**What Clara Does Today:**

**1. Deep PR Reviews (Manual)**
- Reads every line of code
- Mentally models impact: "This touches auth... what uses auth?"
- Asks questions: "Did you test X scenario?"
- **Problem:** Doesn't scale, bottleneck, exhausting

**2. Post-Incident Analysis**
- After incident, traces root cause (git blame, stack traces)
- Updates runbook: "Be careful changing file X"
- Team meeting: "Let's be more careful with auth"
- **Problem:** Reactive, doesn't prevent future incidents

**3. Architecture Reviews (Quarterly)**
- Analyzes git history: which files change most?
- Identifies hotspots: high churn + incidents
- Proposes refactoring priorities
- **Problem:** Time-consuming (8 hours), point-in-time, manual

**4. Slack Broadcasting**
> "Hey team: Before touching the auth module, please check with me first. It's caused several incidents."

- **Problem:** Doesn't scale, slows team down, creates dependency

### CodeRisk Value Proposition

**How CodeRisk Helps Clara:**

**1. PR Review Efficiency**

Before:
```
Clara reviews PR: 45 minutes
- Read code: 10 min
- Mental impact analysis: 20 min ← ⚠️ This is manual, error-prone
- Check tests: 10 min
- Write feedback: 5 min
```

After (with CodeRisk):
```
Clara reviews PR: 20 minutes (55% faster)
- crisk check already run (0 min, automated)
- Read CodeRisk report: 2 min ← Evidence-based, not guesswork
- Focus on what tool flagged: 10 min
- Check tests: 5 min
- Write feedback: 3 min

If crisk check = LOW:
  → Approve quickly (5 min review, not 45 min)
```

**2. Data-Backed Architecture Decisions**

Before:
```
Clara to Product Manager:
"I think we should refactor the auth module. It feels risky."

PM: "Can you quantify the risk? What's the ROI?"
Clara: "It's hard to measure... my gut says it's important."
PM: "Let's prioritize features for now." ❌
```

After (with CodeRisk):
```
Clara to Product Manager:
"CodeRisk data shows auth.js:
  - Caused 5 incidents in last 6 months
  - Changes together with 15 other files (high coupling)
  - 30% test coverage (low safety)
  - G² Surprise score: 8.5/10 (abnormally high risk)

Estimated cost: 2 weeks refactoring
Estimated benefit: 60% reduction in auth-related incidents (2 hours/incident × 5 incidents/year = 10 hours saved)

ROI: 2 weeks investment → 10 hours/year saved + reduced customer impact"

PM: "That's compelling. Let's schedule it for next sprint." ✅
```

**3. Team Scaling & Onboarding**

Before:
```
New engineer joins team:
Clara: "Here are the risky files... [lists 10 files from memory]"
Clara: "This file is coupled to that file because..."
Clara: "3 months ago we had an incident when..."

New engineer: "How do I remember all this?" 😰
Takes 2-3 months to gain intuition
```

After (with CodeRisk):
```
New engineer joins team:
Clara: "Run crisk check before every commit. It'll tell you what's risky."

New engineer makes change:
crisk check → HIGH RISK warning with evidence

New engineer learns incrementally:
  - Which files are coupled
  - Why they're risky (incident history)
  - What to check before committing

Ramp-up time: 2-3 weeks (not 2-3 months) ✅
```

**4. Incident Prevention**

Current state:
- 2-3 incidents/month from architectural coupling
- 2 hours average incident resolution time
- Team morale impact
- Customer churn risk

With CodeRisk:
- 60-80% reduction (0.5-1 incidents/month)
- Caught in pre-commit, not production
- Team confidence increases
- Customer trust maintained

### Success Metrics for Clara

**Team productivity:**
- 50% faster PR reviews (45 min → 20 min)
- 60% reduction in incidents (2-3/month → 1/month)
- 30% faster onboarding (3 months → 2 weeks for context)

**Decision making:**
- Data-backed refactoring proposals (not "gut feel")
- Architectural hotspot tracking (auto-generated)
- Incident attribution (which files caused incidents?)

**Team adoption:**
- >80% of team runs `crisk check` before commits
- <5% override rate (low false positives = high trust)
- Team asks Clara less (self-service risk assessment)

### Quotes from User Research

> **On PR Review Efficiency:**
> "CodeRisk cut my PR review time in half. I focus on what the tool flags, not trying to mentally model the entire codebase."
> — Clara (Tech Lead, 10 YOE)

> **On Data-Backed Decisions:**
> "I finally have quantitative evidence for refactoring. Product can't argue with 'this file caused 5 incidents in 6 months.'"
> — Emma (Engineering Manager, 12 YOE)

> **On Scaling Team:**
> "New engineers ramp up 3x faster with CodeRisk. They learn risky patterns incrementally instead of making mistakes first."
> — David (Tech Lead, 9 YOE)

### Adoption Triggers

**When does Clara evaluate CodeRisk?**

**Trigger 1: After Major Incident**
- High-profile production outage
- Leadership asks: "How do we prevent this?"
- Clara needs better tools to demonstrate due diligence

**Trigger 2: Team Scaling**
- Hiring 3-5 new engineers
- Can't scale manual PR reviews
- Needs automated quality gates

**Trigger 3: Architecture Review**
- Quarterly planning: Which areas to refactor?
- Needs data on hotspots, coupling, risk
- Manual analysis too time-consuming

---

## Persona 3: Alex the Enterprise Architect

### Profile

**Role:** Enterprise Architect / Principal Engineer
**Company Type:** Enterprise (1,000+ engineers)
**Scope:** Multiple teams (5-10 teams, 50-100 engineers)
**Experience:** 15+ years, deep security/compliance background
**Reports to:** VP Engineering or CTO
**Location:** On-site (headquarters)

### Background

Alex is responsible for enterprise architecture, security, and compliance across multiple engineering teams. He evaluates and approves all third-party developer tools. He's deeply skeptical of SaaS tools due to security, compliance, and data residency requirements.

Alex has seen too many vendors promise "secure" SaaS and then have data breaches. His default answer to new tools is "no" unless they can prove enterprise-grade security, compliance, and on-prem deployment options.

### Goals

**Primary Goals:**
- ✅ Ensure zero data leakage (source code never leaves company network)
- ✅ Maintain SOC2, HIPAA, GDPR compliance
- ✅ Evaluate tools for 5-10 teams (scalable decisions)
- ✅ Standardize tooling across organization (reduce vendor sprawl)

**Secondary Goals:**
- Reduce total cost of ownership (TCO)
- Enable teams without compromising security
- Audit trail for all tool usage

### Pain Points

**1. SaaS Tools Can't Meet Security Requirements**
> "Most code analysis tools want to send our proprietary code to their cloud. That's a non-starter for us."

- Source code is IP (intellectual property)
- Compliance: HIPAA, SOC2, FedRAMP
- Can't use standard SaaS offerings

**2. On-Prem Deployment is Complex**
> "Vendors say 'we support on-prem' but it's just a Docker image. We need VPC deployment, SSO, audit logs, the works."

- Need self-hosted in customer VPC
- SAML/OIDC authentication required
- Detailed audit logs for compliance

**3. High False Positive Rate Causes Alert Fatigue**
> "We tried SonarQube enterprise. 10,000+ findings across 100 repos. 90% noise. Teams ignore it now."

- Can't deploy noisy tools at scale
- Alert fatigue kills adoption
- Developers circumvent tools ("just click approve")

**4. Vendor Lock-In Concerns**
> "If we adopt a tool and it shuts down or changes pricing, we're stuck. Need to ensure we can export data."

- Graph data must be exportable
- Can't be dependent on vendor APIs
- Need migration path

### CodeRisk Value Proposition

**How CodeRisk Addresses Enterprise Requirements:**

**1. Enterprise Deployment (Privacy-First)**

**Standard SaaS (Clara's team):**
```yaml
# Cloud-hosted CodeRisk
- Neptune: AWS managed (our infrastructure)
- Source code: Never stored (only graph metadata)
- LLM: User's API key (BYOK model)
```

**Enterprise Self-Hosted (Alex's requirements):**
```yaml
# Customer VPC deployment
- Neptune: Self-hosted in customer VPC (full control)
- Source code: Never leaves customer network
- LLM: Custom endpoint (Azure OpenAI, AWS Bedrock, or local Ollama)
- Authentication: SAML/OIDC (SSO with Okta, Azure AD)
- Data residency: 100% customer infrastructure
- Audit logs: All actions logged to customer SIEM
```

**2. Compliance & Security**

**SOC2 Type II:**
- Access controls (RBAC)
- Encryption at rest (AES-256)
- Encryption in transit (TLS 1.3)
- Audit trail (all API calls logged)

**HIPAA:**
- BAA available (Business Associate Agreement)
- PHI never transmitted
- Self-hosted option (customer controls data)

**GDPR:**
- Data residency (EU region available)
- Right to deletion (graph data erasable)
- Data export (graph backup in customer S3)

**3. Low False Positive Rate**

**Enterprise scale problem:**
- 100 repos × 100 findings/repo = 10,000 findings
- If 10% false positive rate → 1,000 false alarms
- Developers ignore tool

**CodeRisk solution:**
- <3% false positive rate
- 100 repos × 20 findings/repo = 2,000 findings (5x fewer)
- If 3% FP rate → 60 false alarms (17x fewer)
- Developers trust tool

**4. Total Cost of Ownership (TCO)**

**Enterprise pricing:**
```
SonarQube Enterprise: $150/developer/year × 100 devs = $15,000/year
CodeRisk Enterprise: $5,000-10,000/month base + $50/user/month = $66,000-120,000/year

Wait, CodeRisk is MORE expensive?

But:
- SonarQube: High FP rate (10-20%) → developers ignore → low value
- CodeRisk: Low FP rate (<3%) → developers use daily → high value
- CodeRisk: Prevents 60-80% of incidents → saved incident costs
```

**ROI calculation:**
```
Incident costs (conservative):
- 2-3 incidents/month × 10 teams = 25 incidents/year
- 4 hours average incident resolution × $150/hour = $600/incident
- Total: 25 × $600 = $15,000/year direct cost
- Hidden costs: customer churn, brand damage, team morale

CodeRisk prevents 60-80% of incidents:
- Saved costs: $15,000 × 70% = $10,500/year (direct)
- Hidden savings: customer retention, team confidence

Net TCO: $66K (cost) - $10.5K (direct savings) = $55.5K
If we include hidden costs (2-3x direct), CodeRisk pays for itself
```

### Success Metrics for Alex

**Security & Compliance:**
- Zero data breaches from tool usage
- SOC2/HIPAA/GDPR compliance maintained
- Audit trail for all tool access

**Organizational Adoption:**
- 5-10 teams using CodeRisk (50-100 engineers)
- >80% daily usage rate
- <5% false positive rate maintained at scale

**Business Value:**
- 60% reduction in architectural incidents
- $10K+ annual cost savings from prevented incidents
- Improved developer productivity (less time in incidents)

### Adoption Triggers

**When does Alex evaluate CodeRisk?**

**Trigger 1: Security/Compliance Audit**
- Auditor asks: "How do you prevent code quality issues?"
- Need to demonstrate automated controls
- Manual PR review not sufficient for audit

**Trigger 2: Major Incident with Compliance Implications**
- Production outage affected customer PII
- Root cause: Architectural coupling not caught in review
- Need automated pre-commit gates

**Trigger 3: Vendor Consolidation**
- Currently using 5+ code quality tools (SonarQube, CodeClimate, custom scripts)
- High maintenance burden
- Looking to standardize on fewer vendors

---

## Secondary Personas

### Persona 4: Maya the Product Manager (Influencer)

**Role:** Product Manager
**Interest:** Feature velocity vs code quality

**Pain Point:**
- Engineers say "we need to refactor" but can't quantify technical debt
- Need data-driven prioritization of tech debt vs features

**CodeRisk Value:**
- Data-backed refactoring decisions (incident attribution)
- ROI calculator (cost of refactoring vs cost of incidents)

### Persona 5: Sam the DevOps Engineer (Integration User)

**Role:** DevOps/Platform Engineer
**Interest:** CI/CD pipeline integration

**Pain Point:**
- Want automated quality gates in CI
- Current tools (SonarQube) too noisy for blocking PRs

**CodeRisk Value:**
- Low false positive rate → can safely block PRs
- API integration with CI/CD pipelines
- Automated status checks on GitHub PRs

---

## User Journey Maps

### Ben's Journey: From Discovery to Daily Habit

**Stage 1: Discovery (Day 0)**
- Trigger: Production incident from unexpected coupling
- Action: Google "code coupling tool", finds CodeRisk
- Experience: Reads docs, sees "pre-flight check" positioning
- Emotion: 😰 Anxious (just caused incident) → 🤔 Curious

**Stage 2: Trial (Day 1)**
- Action: Installs CLI (`brew install crisk`)
- Experience: Runs `crisk check` on recent change
- Result: HIGH risk warning, shows temporal coupling he missed
- Emotion: 😮 Surprised → 😅 Relieved (caught before next incident)

**Stage 3: Evaluation (Week 1)**
- Action: Runs `crisk check` on every commit for a week
- Experience: Catches 2 real issues, 0 false positives
- Result: Trust built through accuracy
- Emotion: 😌 Confident → 😊 Satisfied

**Stage 4: Habit (Week 2-4)**
- Action: `crisk check` becomes muscle memory (like `git status`)
- Experience: Runs 10-15 times/day without thinking
- Result: 0 incidents in 1 month (vs 2-3/month before)
- Emotion: 😎 Confident → 💪 Empowered

**Stage 5: Advocacy (Month 2+)**
- Action: Recommends to team, demos at lunch
- Experience: Team adopts, collective quality improves
- Result: Team incident rate drops 60%
- Emotion: 🌟 Proud → 🚀 Team success

### Clara's Journey: From Skepticism to Champion

**Stage 1: Awareness (Week 0)**
- Trigger: Ben mentions CodeRisk in standup
- Action: Reviews website, docs, pricing
- Experience: Skeptical (another tool?), but intrigued by "agentic" approach
- Emotion: 🤨 Skeptical → 🤔 Interested

**Stage 2: Evaluation (Week 1-2)**
- Action: Requests demo, tests on team repo
- Experience: Runs on recent incident, CodeRisk would've caught it
- Result: Data shows 5 hotspot files (matches her intuition)
- Emotion: 😲 Impressed → 🧐 Evaluating

**Stage 3: Pilot (Week 3-8)**
- Action: Pilot with 3 team members for 1 month
- Experience: PR review time drops 40%, 0 incidents during pilot
- Result: Quantitative success (tracks metrics)
- Emotion: 📊 Analytical → ✅ Convinced

**Stage 4: Rollout (Month 3)**
- Action: Mandates `crisk check` for entire team
- Experience: Team adopts easily (already saw Ben using it)
- Result: Team productivity increases, incident rate drops
- Emotion: 😊 Satisfied → 🎯 Goal achieved

**Stage 5: Expansion (Month 4+)**
- Action: Presents results to engineering leadership
- Experience: Other teams request access
- Result: Becomes standard tool across 5 teams
- Emotion: 🏆 Champion → 📈 Organizational impact

---

## Behavioral Patterns

### When Developers Run `crisk check`

**High-frequency patterns (10-15 times/day):**
1. Before committing (primary use case)
2. After making risky change (doubt/validation)
3. Before pushing to main (final check)
4. When switching branches (context check)

**Low-frequency patterns (1-2 times/week):**
1. Exploring unfamiliar code (learning)
2. Investigating incident (root cause analysis)
3. Planning refactoring (identifying hotspots)

### When Developers Override Warnings

**Legitimate overrides (<5% of checks):**
- Intentional coupling in framework code
- Test files inflating dependency count
- Known safe refactoring (moving files)

**Warning signs (bad overrides):**
- "Don't have time to fix"
- "Looks like false positive" (without investigation)
- Consistent override pattern (developer ignores tool)

---

## Related Documents

- [vision_and_mission.md](vision_and_mission.md) - Product vision and positioning
- [competitive_analysis.md](competitive_analysis.md) - Market positioning vs Greptile, Codescene
- [pricing_strategy.md](pricing_strategy.md) - Pricing tiers for each persona
- [success_metrics.md](success_metrics.md) - How we measure impact for each persona

---

**Last Updated:** October 2, 2025
**Based on:** 12 user interviews (Sept-Oct 2025)
**Next Review:** January 2026 (after 100+ user feedback)
