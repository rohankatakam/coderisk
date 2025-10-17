# User Personas (MVP Focus)

**Last Updated:** October 17, 2025
**Status:** Active - Simplified for MVP launch
**Target Users:** Solo developers + small teams (2-10 people)

> **📘 Strategic Simplification:** Focused on solo developers and small teams for MVP. Enterprise personas (Alex, Maya, Sam) archived to [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/) for v2-v4.

---

## MVP Target Users

CodeRisk MVP targets two primary user personas:

1. **Ben the Developer** - Solo developer, main end user (PRIMARY)
2. **Clara the Tech Lead** - Small team lead (2-10 engineers), evaluates tools (SECONDARY)

**NOT targeting for MVP:**
- ❌ Enterprise architects (50-500+ engineers) - v2
- ❌ Product managers - influencer only
- ❌ DevOps engineers - integration user only

**Why:** Focus on developers who commit code daily, validate demand with simple local tool before building complex enterprise features.

---

## Persona 1: Ben the Developer (Solo Developer)

### Profile

**Role:** Software Engineer (solo or small team)
**Company Type:** Startup or small tech company (2-20 engineers)
**Team Size:** Solo developer OR 2-8 person team
**Experience:** 3+ years professional development
**Tech Stack:** Modern web stack (React, Node.js, Python, Go, etc.)
**Tools:** AI coding assistants (Claude Code, Cursor, Copilot)
**Location:** Remote or hybrid

### Background

Ben is a developer who uses AI coding assistants daily. He generates code 5-10x faster than before, but faces a new problem: **"Is it safe to commit this AI-generated code?"**

He's working on a codebase (500-5,000 files) with complex dependencies. AI helps him write code quickly, but doesn't understand the codebase's architectural risks. Tests pass, linters are happy, but he's still uncertain about hidden coupling or risky patterns.

### Daily Workflow

**AI-Accelerated Development:**
1. Use Claude Code/Cursor to generate code (10-30 min)
2. Review AI output, make tweaks (5-10 min)
3. Run unit tests locally (1-3 min) - pass ✅
4. **[Current pain point]** Commit → Push → Hope AI didn't introduce risky coupling
5. Wait for PR review (if on a team) OR deploy directly (if solo)
6. **[Major pain point]** Discover issues in production (happens ~1-2x/month)

**Time pressure:**
- Shipping features fast (AI velocity)
- Can't manually review all AI-generated code deeply
- Need fast pre-commit safety check

### Goals

**Primary Goals:**
- ✅ Ship AI-generated code quickly without breaking production
- ✅ Understand architectural risks BEFORE committing
- ✅ Avoid embarrassing incidents from AI-generated code
- ✅ Build confidence in AI output (reduce anxiety)

**Secondary Goals:**
- Learn codebase patterns faster
- Reduce time spent debugging AI mistakes
- Use AI coding assistants more confidently

### Pain Points

**1. Uncertainty About AI-Generated Code**
> "Claude Code just generated 200 lines in 30 seconds. It looks good, tests pass... but is it safe? I have no idea what else it might break."

- AI generates code fast but doesn't know codebase architecture
- Tests only validate expected behavior, not hidden coupling
- No tool validates architectural safety of AI output

**2. Manual Review is Time-Consuming**
> "I spend 20 minutes grepping for dependencies, checking git history, trying to understand if this change is risky. Defeats the purpose of AI coding."

- Manual investigation defeats AI velocity gains
- Can't manually review every AI-generated change deeply
- Tribal knowledge about risky files not accessible

**3. Production Incidents from AI Code**
> "Last week AI generated code that worked perfectly... except it broke authentication in a subtle way. Took 2 hours to debug in production."

- AI doesn't understand temporal coupling
- Hard to trace issues back to AI-generated changes
- Anxiety about using AI for critical features

**4. No Pre-Commit Safety Check**
> "I want a 'spell check' for architectural risks. Just tell me before I commit: Is this safe?"

- Linters check syntax, not architecture
- Tests check behavior, not coupling
- No tool for pre-commit architectural risk assessment

### Current Solutions & Workarounds

**What Ben Does Today (Imperfect Solutions):**

**1. Carefully Review AI Output**
- Read every line AI generated (slow, defeats AI velocity)
- Try to understand architectural impact (error-prone)
- **Problem:** Time-consuming, incomplete, slows down AI coding benefits

**2. Ask AI to Explain**
```
Ben: "Claude, explain what risks this code might introduce"
AI: "This looks safe, it's just updating the authentication logic..."
[Later in production: breaks reporting dashboard due to hidden coupling]
```
- **Problem:** AI doesn't know codebase architecture, can't see temporal coupling

**3. Run All Tests (Hope for the Best)**
```bash
npm test  # All pass ✅
git commit -am "AI generated auth update"
git push
# 🤞 Cross fingers
```
- **Problem:** Tests don't catch architectural coupling, false confidence

**4. Avoid AI for Risky Features**
> "I only use AI for simple stuff. For auth, payments, core features... I write it manually (slow)."

- **Problem:** Can't use AI for high-value work, loses productivity gains

### CodeRisk Value Proposition

**How CodeRisk Helps Ben:**

**Before (Current State):**
```bash
# Ben's AI coding workflow:
1. Claude Code generates auth update (2 min)
2. Tests pass (1 min) ✅
3. Looks good visually (2 min)
4. Commit & push
5. [Later] Production breaks - auth change broke reporting module ❌
6. Debug for 2 hours, rollback, fix properly

Total time: 2+ hours (with anxiety + incident stress)
```

**After (With CodeRisk):**
```bash
# Ben's workflow with CodeRisk:
1. Claude Code generates auth update (2 min)
2. crisk check (5 seconds) ⚠️ HIGH RISK

   Evidence:
   - auth.js changed together with reporting.js in 70% of commits
   - Pattern detected: Risky authentication pattern (shared session state)
   - 8 files import authenticate()
   - Test coverage: 35% (low for critical file)

   Recommendations:
   - Review coupled file: src/reporting/dashboard.js
   - Consider: Extract session logic to separate module
   - Add integration tests for auth + reporting

3. Ask AI to refactor based on CodeRisk feedback (5 min)
4. crisk check (5 seconds) ✅ LOW RISK
5. Commit with confidence
6. No production incident ✅

Total time: 10 minutes (with confidence)
```

**Specific Benefits:**

**1. Fast Pre-Commit Safety Check**
- 5 second check (doesn't slow down AI velocity)
- Private feedback (before committing)
- Actionable recommendations (what to fix)

**2. Validates AI-Generated Code**
- Catches architectural risks AI doesn't know about
- Pattern library detects risky code patterns
- Graph analysis shows hidden coupling

**3. Reduces Production Incidents**
- 60-80% reduction in AI-related coupling incidents
- Catches issues in development, not production
- Lower anxiety, more confident AI usage

**4. Enables AI for Critical Features**
- Use AI for auth, payments, core features safely
- Pre-flight check validates AI output
- Maximize AI productivity without sacrificing quality

**5. Local-First & Free**
- Runs locally (fast, private)
- BYOK model (transparent costs: ~$1-2/month)
- No cloud account needed

### Success Metrics for Ben

**Usage patterns:**
- Runs `crisk check` 5-10 times per day (habit formed)
- <5 second latency (acceptable for AI workflow)
- <5% false positive rate (trust maintained)

**Outcomes:**
- 60-80% reduction in production incidents
- 50% time saved debugging AI-generated code
- Increased confidence using AI for critical features

### Quotes from User Research

> **On AI Coding Uncertainty:**
> "CodeRisk is like spell-check for architectural risks. Before I commit AI-generated code, I run crisk check. Takes 5 seconds, saves 2 hours debugging."
> — Ben (Developer, AI coding daily)

> **On Confidence:**
> "I used to be scared to use Claude Code for critical features. Now I use it confidently - CodeRisk validates the architecture."
> — Sarah (Solo Developer, 4 YOE)

> **On Speed:**
> "5 second check vs 2 hour production incident. No-brainer."
> — Mike (Small Team Developer, 6 YOE)

### Adoption Triggers

**When does Ben try CodeRisk?**

**Trigger 1: After AI-Caused Incident**
- AI generated code that broke production
- Feeling uncertain about AI coding tools
- High motivation to validate AI output

**Trigger 2: Using AI Coding Assistant**
- Just started using Claude Code, Cursor, or Copilot
- Generating code 5-10x faster
- Needs safety check for AI output

**Trigger 3: Solo Developer**
- No team to review code
- No senior engineer to ask
- Needs automated safety check

---

## Persona 2: Clara the Tech Lead (Small Team)

### Profile

**Role:** Engineering Team Lead / Tech Lead
**Company Type:** Small tech company or startup (10-50 employees)
**Team Size:** Leads 2-10 engineers
**Experience:** 8+ years professional development, 1-2 years leadership
**Reports to:** CTO or VP Engineering
**Location:** Remote or hybrid

### Background

Clara is a technical leader managing a small team (2-10 engineers). Her team uses AI coding assistants to ship faster, but she's concerned about code quality. She reviews PRs but can't deeply review every AI-generated change - there's too much code.

Clara needs a way to automatically catch architectural risks in AI-generated code WITHOUT slowing down the team. She wants data-backed evidence to prioritize refactoring and prevent incidents.

### Daily Workflow

**Morning:**
1. Review overnight deploys (check for incidents)
2. Triage support tickets
3. Check PR review queue (5-10 PRs, many AI-generated)

**Afternoon:**
1. Review PRs (30-45 min each) **[Major time sink]**
2. 1:1s with team members
3. Architecture discussions with product
4. Code when time allows (20-30% of time)

**Evening:**
1. Approve PRs for deployment
2. Monitor production metrics
3. Post-incident reviews (if needed) **[High stress]**

### Goals

**Primary Goals:**
- ✅ Catch architectural risks in AI-generated code BEFORE production
- ✅ Scale PR reviews without becoming bottleneck
- ✅ Prevent incidents while maintaining team velocity
- ✅ Improve team's AI coding practices

**Secondary Goals:**
- Data-backed refactoring decisions
- Team learning (which patterns are risky?)
- Reduce time in incident response

### Pain Points

**1. Can't Deeply Review All AI-Generated Code**
> "My team generates 10x more code now with AI. I can't review it all deeply. I spot-check... and hope for the best."

- Team ships 5-10 PRs per day (AI velocity)
- Each PR has 100-500 lines (AI-generated)
- Can't manually review every architectural decision

**2. AI Introduces Subtle Coupling**
> "AI generates code that works perfectly in isolation but introduces coupling I don't catch until production."

- AI doesn't understand codebase architecture
- Coupling invisible in code review
- Discovered only after incident

**3. Need Data to Prioritize Refactoring**
> "I know some files are risky, but I can't convince the team to refactor without data. 'It feels risky' isn't enough."

- Product wants features, not refactoring
- Need quantitative evidence for tech debt
- "Trust me" doesn't work with data-driven teams

**4. Team Scales, Knowledge Doesn't**
> "I'm hiring 2 new engineers. They'll use AI but don't know which files are risky. How do I transfer this knowledge?"

- Tribal knowledge in Clara's head
- New engineers don't know risky patterns
- Can't scale manual knowledge transfer

### Current Solutions & Workarounds

**What Clara Does Today:**

**1. Spot-Check PR Reviews**
- Reviews 30% of PRs deeply, skims 70%
- Focuses on "critical" files (auth, payments)
- **Problem:** Misses issues in "non-critical" files, incomplete coverage

**2. Post-Incident Analysis**
- After incident, updates team wiki: "Don't do X"
- Team meeting: "Be careful with AI for Y"
- **Problem:** Reactive, doesn't prevent future incidents

**3. Manual Architecture Reviews (Monthly)**
- Analyzes git history for hotspots
- Identifies risky files manually
- **Problem:** Time-consuming (4-8 hours), point-in-time, outdated quickly

### CodeRisk Value Proposition

**How CodeRisk Helps Clara:**

**1. Automated PR Pre-Flight Checks**

Before:
```
Clara reviews 10 PRs/day × 30 min each = 5 hours/day
- Can't review all deeply
- Misses architectural issues
- Team waits on her approval
```

After (with CodeRisk):
```
Team runs crisk check before PR
Clara reviews 10 PRs/day × 15 min each = 2.5 hours/day (50% faster)

If crisk check = LOW RISK:
  → Quick review (10 min)
If crisk check = HIGH RISK:
  → Deep review based on CodeRisk findings (20 min)

Benefits:
- Focus on what matters
- Evidence-based review
- Team unblocked faster
```

**2. Data-Backed Refactoring Decisions**

Before:
```
Clara to Team:
"I think we should refactor auth.js. It feels risky."

Team: "Can you quantify the risk? We have feature deadlines."
Clara: "It's based on my experience... trust me."
Team: "Let's do features first." ❌
```

After (with CodeRisk):
```
Clara to Team:
"CodeRisk data shows auth.js:
  - Caused 3 incidents in last 3 months
  - Changes with 8 other files (high coupling)
  - Pattern matches: Risky shared session state
  - Test coverage: 30% (low for critical file)

Estimated cost: 1 week refactoring
Estimated benefit: Prevent 2-3 incidents/month (4 hours each = 12 hours saved)

ROI: 1 week investment → 12 hours/month saved"

Team: "That's compelling. Let's do it." ✅
```

**3. Team Knowledge Scaling**

Before:
```
New engineer joins:
Clara: "Be careful with auth.js, payment.js, these 5 other files..."
Clara: "They're coupled in subtle ways because..."

New engineer: "How do I remember all this?" 😰
Ramp-up time: 1-2 months
```

After (with CodeRisk):
```
New engineer joins:
Clara: "Run crisk check before every commit. It'll warn you about risky changes."

New engineer makes change:
crisk check → HIGH RISK (explains why)

New engineer learns incrementally:
  - Which files are risky
  - Why they're coupled
  - What patterns to avoid

Ramp-up time: 2-3 weeks ✅
```

**4. Incident Prevention**

Current state:
- 1-2 incidents/month from AI-generated code
- 2-4 hours average incident resolution
- Team morale impact

With CodeRisk:
- 60-80% reduction (0.2-0.5 incidents/month)
- Caught pre-commit, not production
- Team confidence increases

### Success Metrics for Clara

**Team productivity:**
- 50% faster PR reviews (30 min → 15 min average)
- 60% reduction in incidents (1-2/month → 0.5/month)
- 40% faster onboarding (2 months → 3 weeks)

**Decision making:**
- Data-backed refactoring proposals
- Automated hotspot tracking
- Incident attribution (which files caused incidents)

**Team adoption:**
- >80% of team runs `crisk check` before commits
- <5% override rate (high trust)
- Team self-service (asks Clara less)

### Quotes from User Research

> **On PR Review Efficiency:**
> "CodeRisk cut my PR review time in half. I focus on what the tool flags, not trying to guess what AI might have broken."
> — Clara (Tech Lead, 10 YOE)

> **On Data-Backed Decisions:**
> "I finally have evidence for refactoring. 'This file caused 3 incidents' beats 'I have a feeling' every time."
> — Emma (Team Lead, 9 YOE)

> **On Team Scaling:**
> "New engineers ramp up 2x faster. CodeRisk teaches them risky patterns as they code."
> — David (Tech Lead, 8 YOE)

### Adoption Triggers

**When does Clara evaluate CodeRisk?**

**Trigger 1: After AI-Related Incident**
- Production outage from AI-generated code
- Team needs better quality gates
- Leadership asks: "How do we prevent this?"

**Trigger 2: Team Adopting AI Coding**
- Team starts using Claude Code, Cursor, Copilot
- Code velocity increases 5-10x
- Need automated architectural review

**Trigger 3: Team Scaling**
- Hiring 2-3 new engineers
- Can't scale manual PR reviews
- Need automated quality gates

---

## User Journey Maps

### Ben's Journey: From Discovery to Daily Habit

**Week 1: Discovery & Trial**
- **Trigger:** AI-generated code broke production (auth incident)
- **Action:** Googles "validate AI code architecture", finds CodeRisk
- **Trial:** `brew install crisk`, runs on recent AI changes
- **Result:** HIGH risk warning on exact change that caused incident
- **Emotion:** 😮 "This would've caught my bug!"

**Week 2-3: Building Trust**
- **Action:** Runs `crisk check` on every AI-generated commit
- **Experience:** Catches 2 real issues, 0 false positives
- **Result:** Trust built, becomes habit
- **Emotion:** 😌 Confident → 💪 Empowered

**Month 2+: Advocate**
- **Action:** Uses AI more confidently, recommends CodeRisk
- **Result:** 0 incidents in 2 months (vs 2-3/month before)
- **Emotion:** 🚀 "AI coding is safe now"

### Clara's Journey: From Evaluation to Rollout

**Week 1: Awareness**
- **Trigger:** Ben mentions CodeRisk after preventing incident
- **Action:** Reviews website, tries on team repo
- **Result:** Shows 3 hotspot files (matches intuition)
- **Emotion:** 🤔 Interested → 🧐 Evaluating

**Week 2-4: Pilot**
- **Action:** 3 team members use for 2 weeks
- **Result:** PR review time drops 40%, 0 incidents during pilot
- **Emotion:** 📊 Data looks good → ✅ Convinced

**Month 2+: Team Rollout**
- **Action:** Entire team (8 people) uses CodeRisk
- **Result:** Team incident rate drops 70%, velocity maintained
- **Emotion:** 🎯 Goal achieved

---

## Pricing Alignment

### Ben (Solo Developer)
- **Tier:** Free (BYOK)
- **Cost:** $1-2/month (LLM API costs)
- **Value:** Prevents 1-2 incidents/month (saves 4+ hours debugging)
- **ROI:** $1 spent → 4 hours saved = massive ROI

### Clara (Small Team: 8 people)
- **Tier:** Free (MVP) OR Pro ($10/user/month if team features requested)
- **Cost:**
  - Free tier: 8 × $1-2/month = $8-16/month (LLM only)
  - Pro tier (if needed): 8 × $10 = $80/month + $16 LLM = $96/month
- **Value:**
  - Prevents 1-2 incidents/month × 4 hours × $150/hour = $600-1,200/month saved
  - PR review time savings: 2.5 hours/day × 20 days × $150 = $7,500/month
- **ROI:**
  - Free tier: $16 → $7,500+ value (470x ROI)
  - Pro tier: $96 → $7,500+ value (78x ROI)

---

## Related Documents

**Product:**
- [mvp_vision.md](mvp_vision.md) - MVP vision and scope
- [competitive_analysis.md](competitive_analysis.md) - Local-first positioning
- [simplified_pricing.md](simplified_pricing.md) - Free BYOK + optional pro

**User Experience:**
- [developer_experience.md](developer_experience.md) - Local tool UX
- [developer_workflows.md](developer_workflows.md) - Git workflows

**Archived (Future):**
- [../99-archive/00-product-future-vision/](../99-archive/00-product-future-vision/) - Enterprise personas (v2-v4)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP launch (Week 7-8), after 50+ user feedback
