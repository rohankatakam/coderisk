# Strategic Decision Framework: Customer Discovery vs. Development

**Created:** October 21, 2025
**Purpose:** Strategic guidance for next 4 weeks - parallel customer discovery and MVP completion
**Decision Point:** Week 4 - GO/NO-GO based on customer validation + working MVP

---

## Current State Assessment

### What You've Built (75% Complete)
- ✅ **Core graph construction working** - 3-layer ingestion (AST, temporal, incidents)
- ✅ **CLI foundation solid** - Cobra, Neo4j, Git utilities, Tree-sitter parsing
- ✅ **Infrastructure base** - Docker Compose, local deployment, 35,759 LOC
- ✅ **Development tooling** - GoReleaser, Homebrew formula, CI/CD ready
- ⚠️ **Risk assessment incomplete** - Phase 1 metrics exist, Phase 2 LLM integration missing
- ⚠️ **Not production-ready** - 45% test coverage, performance not optimized, no real user validation

### What You DON'T Have Yet
- ❌ End-to-end `crisk check` with reliable risk scores
- ❌ LLM integration for high-risk file analysis
- ❌ Proven false positive rate <10%
- ❌ Performance benchmarks showing <5s responses
- ❌ Real user feedback validating the value proposition

### The Critical Question
**Is "pre-commit regression prevention" a real pain point or a solution looking for a problem?**

Your competitive analysis is excellent, but it's theoretical. You need to validate:
- Do developers actually want pre-commit checks?
- Is temporal coupling valuable or just noise?
- Is your positioning vs. Greptile clear or confusing?
- Will people pay for this?

---

## Recommendation: PARALLEL PATH

### Why BOTH Customer Discovery AND Development?

**You're at the PERFECT inflection point:**
1. **You have enough to demonstrate** - Graph construction works, you can show the concept
2. **You DON'T have enough to ship** - Risk assessment isn't complete, no validation
3. **Competitive positioning is UNCLEAR** - Is there really a gap between SonarQube and Greptile?
4. **The problem is ASSUMED** - You believe developers need this, but have you validated the pain?

**Key Insight:** You NEED a working prototype to validate value in customer conversations, AND you NEED customer validation before investing 6+ more months building.

---

## Track 1: Focused Customer Discovery (20% time, 3-4 weeks)

### Who to Talk To
- **5-8 developer contacts** who use Claude Code, Cursor, or Copilot daily
- **2-3 tech leads** managing small teams (2-10 people)
- **NOT enterprises** (out of scope for MVP)

### Critical Questions to Answer

**Problem Validation:**
1. ✅ Is "pre-commit regression prevention" a real pain point?
2. ✅ Do developers actually care about temporal coupling and co-change patterns?
3. ✅ Is your positioning vs. Greptile/SonarQube compelling or confusing?
4. ✅ Will people pay $9-29/user/month or is BYOK the only viable model?

### Customer Discovery Script

#### Opening (5 min)
> "Hey [name], I'm building a tool for developers who use AI coding assistants like Claude Code or Cursor. I'd love 30 minutes to understand your workflow - not selling anything, just learning. Cool if I ask you some questions?"

#### Problem Discovery (15 min)
1. **"Walk me through your last commit. What checks did you do BEFORE committing to make sure it was safe?"**
   - Look for: Manual due diligence they do (or skip), time spent, what they worry about

2. **"Tell me about a time when you committed code that broke something unexpectedly. What happened?"**
   - Look for: Was it a regression? Did they forget to update a related file? Ownership issue?
   - Follow-up: "Could you have known this would break before committing?"

3. **"Before making a change to a critical file (like payment processing), what do you wish you knew?"**
   - Look for: Who owns it? What depends on it? Past incidents? Test coverage?
   - This is YOUR value prop - do they articulate this need?

4. **"Have you ever changed a file and later realized someone else was working on the same file for a different feature?"**
   - Look for: Ownership/coordination pain, merge conflicts, stepping on toes

5. **"What's the most frustrating part of using AI to generate code? How do you know it's safe?"**
   - Look for: Trust issues, uncertainty, fear of breaking things, time spent reviewing

#### Solution Validation (5 min)

**Scenario 1: Ownership Context**
> "Imagine you're about to edit `payment_processor.py` to add a new feature. Before you start coding, a tool tells you:
> - 'Alice last modified this file 2 days ago for bug INC-453'
> - 'Bob owns this file (80% of commits)'
> - 'This file has 0% test coverage'
>
> Would this change how you approach the task? Would you ping Alice or Bob before starting?"

**Scenario 2: Incident Prevention**
> "You're about to commit a change to authentication logic. A tool tells you:
> - 'Warning: Similar change caused incident INC-789 last month (timeout cascade)'
> - 'auth.py changes usually require updating session_manager.py (they changed together in 15/20 commits)'
> - 'Recommendation: Add timeout handling, review session_manager.py'
>
> Would you find this useful? Would you act on it?"

**Scenario 3: Pre-commit Due Diligence**
> "You just used Claude Code to generate 500 lines of payment processing code. Before committing, a tool analyzes it and says:
> - 'HIGH risk: payment.py has no tests and handles money'
> - 'Files that depend on this: fraud_detector.py, reporting.py, webhooks.py'
> - 'Recommendation: Add integration tests, ping @sarah (fraud expert)'
>
> Would you fix these issues before committing? Or would you commit anyway?"

**Follow-up:** "Which of these three scenarios is most valuable to you? Why?"

#### Pricing (3 min)
1. "Would you pay $1-2/month (just LLM costs)?"
2. "What about $9/month for a full product?"
3. "If your company paid, what's the max per developer?"

#### Closing (2 min)
1. "Would you beta test this if I build it?"
2. "Who else should I talk to?"

### Deliverable: Problem Validation Report (1-2 pages)

**Must Include:**
- Number of contacts who expressed STRONG pain with current workflow
- Evidence of incidents that your tool would have prevented
- Willingness to pay data
- Quotes demonstrating genuine excitement or skepticism
- **GO/NO-GO criteria**: If <50% express genuine pain, PIVOT positioning or product

---

## Track 2: Complete Risk Assessment MVP (80% time, 3-4 weeks)

### Why Continue Building

**You NEED a working prototype to validate value in customer conversations.**
- Graph construction alone isn't useful without risk assessment
- You're 75% done - abandoning now wastes 3+ months of work
- Can't validate "pre-commit checks work" without working checks

### Critical Path (MVP Blockers)

#### Week 1: LLM Integration + Phase 2 Risk Assessment
**Tasks:**
- Implement OpenAI/Anthropic client (`internal/llm/`)
- Connect Phase 2 to `crisk check` command
- Simple escalation: If Phase 1 finds high coupling (>10) → Call LLM once
- Basic error handling (no API key, rate limits, timeouts)

**Validation:** Can analyze high-risk file and provide actionable feedback

#### Week 2: Integration Testing + Performance
**Tasks:**
- Test with 3 real repos (small: commander.js, medium: omnara, large: next.js)
- Measure actual latency (target: <200ms Phase 1, <5s Phase 2)
- Fix performance bottlenecks (query optimization, caching)
- Add progress indicators for slow operations

**Validation:** Meets performance targets on real codebases

#### Week 3: False Positive Tracking + Hardening
**Tasks:**
- Add feedback mechanism (`crisk feedback --false-positive`)
- Test with KNOWN risky commits (from your contacts' incident stories)
- Calculate false positive rate
- Edge case handling, error messages

**Validation:** FP rate <15% (lower threshold since MVP)

#### Week 4: Beta-Ready Release
**Tasks:**
- Test coverage >60% (not 70%, lower bar for speed)
- Homebrew formula working
- Basic documentation (README, installation, troubleshooting)
- 3-5 beta users can install and use successfully

**Validation:** Beta users successfully running `crisk check` on their codebases

### What to SKIP (for now)
- ❌ Complex agent orchestration (just one LLM call)
- ❌ Advanced metrics (stick to 5 core: coupling, co-change, test ratio, churn, incidents)
- ❌ Cloud deployment (local-first for MVP)
- ❌ Settings portal (CLI config sufficient)
- ❌ Perfect test coverage (60% is enough for beta)
- ❌ Branch delta graphs (main branch only)
- ❌ Public repository caching (defer to v2)

---

## 4-Week Execution Plan

### Week 1
- **Mon-Wed**: Finish LLM integration + Phase 2 risk assessment
- **Thu-Fri**: Reach out to 5 contacts, schedule 3 customer discovery calls

### Week 2
- **Mon-Tue**: Run 3 customer discovery calls, document findings
- **Wed-Fri**: Integration testing + performance optimization

### Week 3
- **Mon-Wed**: False positive tracking + hardening based on customer stories
- **Thu-Fri**: Run 2 more customer calls, update problem validation

### Week 4
- **Mon-Tue**: Beta-ready release (Homebrew, docs)
- **Wed**: Synthesize customer discovery findings
- **Thu-Fri**: **GO/NO-GO DECISION**

---

## Decision Framework: After 4 Weeks

### Scenario A: Strong Customer Validation + Working MVP ✅

**Signals:**
- 5+ contacts express STRONG pain ("I need this NOW")
- 2+ contacts offer to beta test immediately
- You can demo `crisk check` finding REAL issues they've had
- False positive rate <15%
- Performance targets met (<200ms Phase 1, <5s Phase 2)

**Next Steps:**
1. Private beta with 5-10 users (your contacts) - 2 weeks
2. Iterate based on feedback - 2-4 weeks
3. Public launch (Homebrew, Product Hunt, HN) - Week 7-8
4. Measure adoption: 100 stars, 50 weekly users by month 3

**Outcome:** GO - Continue to production launch

---

### Scenario B: Weak Customer Validation + Working MVP ⚠️

**Signals:**
- Contacts are polite but not excited ("yeah, that's cool I guess")
- No one volunteers to beta test
- Positioning is confusing ("isn't this what Greptile does?")
- FP rate 15-25%
- MVP works but doesn't solve stated problem

**Next Steps:**
1. **PIVOT positioning** - Maybe it's not "pre-commit" but "AI code review" or "incident prevention"
2. **Focus on specific niche** - Only for teams using AI assistants heavily
3. **Consider partnerships** - Your tech might be valuable to Greptile/Cursor/GitHub
4. **Shelve and learn** - Document learnings, open source, move to next idea

**Outcome:** PIVOT - Change positioning or consider acquisition

---

### Scenario C: Strong Customer Validation + MVP Doesn't Work 🔄

**Signals:**
- Contacts are excited about problem ("I desperately need this")
- Multiple specific pain points validated
- Your tool doesn't solve it (FP rate >30%, too slow, wrong approach)

**Next Steps:**
1. **PIVOT implementation** - Keep problem, change solution
2. **Example**: Maybe static analysis is better than graph construction
3. **Example**: Maybe focus on just temporal coupling, skip AST parsing
4. **Example**: Maybe simple heuristics beat LLM reasoning

**Outcome:** PIVOT - Keep problem, rebuild solution

---

### Scenario D: Weak Validation + MVP Doesn't Work ❌

**Signals:**
- No one cares about the problem
- AND your tool doesn't work well
- FP rate >30%, slow, confusing

**Next Steps:**
1. **Hard stop** - Thank your contacts, open source what you've built
2. **Reflect on learnings** - Why didn't this work? What did you learn?
3. **Next idea** - Apply learnings to new product

**Outcome:** NO-GO - Stop development, archive project

---

## Success Criteria (End of Week 4)

### Minimum Viable Validation
- ✅ **Working `crisk check`** that finds real issues in <5s
- ✅ **5-8 customer conversations** with documented pain points
- ✅ **2-3 beta users** actively testing
- ✅ **Clear evidence** of real problem worth solving
- ✅ **GO/NO-GO decision** supported by data, not hope

### Technical Milestones
- ✅ Phase 1 + Phase 2 risk assessment functional
- ✅ LLM integration (OpenAI/Anthropic)
- ✅ Performance: <200ms (Phase 1), <5s (Phase 2)
- ✅ False positive rate measured (<15% target)
- ✅ Test coverage >60%
- ✅ Homebrew installation working
- ✅ Basic documentation complete

### Customer Validation Milestones
- ✅ 5-8 interviews completed
- ✅ Problem validation report written
- ✅ Evidence of real pain points documented
- ✅ Willingness to pay data collected
- ✅ Beta user commitments (2-3 minimum)

---

## Why NOT Just Build to Production First?

**You risk building something no one wants.**

Your competitive analysis is EXCELLENT, but it's theoretical. You assume:
- Developers want pre-commit checks (maybe they don't?)
- Temporal coupling is valuable (maybe it's noise?)
- Your positioning vs. Greptile is clear (maybe it's confusing?)
- People will pay $9-29/month (maybe BYOK only works?)

**Cursor didn't succeed because they built fast** - they succeeded because they solved a REAL pain point (slow coding) that developers DESPERATELY felt.

**You need to validate:** Do developers DESPERATELY want pre-commit regression prevention? Or is this a "nice to have"?

---

## Key Insights from Competitive Analysis

### Your Positioning (from competitive_analysis.md)
- **Pre-commit timing** (private) vs. Greptile (PR review, public)
- **Regression prevention** vs. SonarQube (security) vs. Codescene (health dashboard)
- **Local-first** vs. cloud-based competitors
- **BYOK model** ($1-2/month) vs. $30-150/month competitors

### Critical Assumptions to Validate
1. **Timing moat**: Do developers value pre-commit feedback over PR feedback?
2. **Regression focus**: Is this more valuable than code quality review?
3. **Local-first**: Do developers prefer local tools over cloud convenience?
4. **BYOK economics**: Is transparent pricing ($1-2/month) more attractive than all-in pricing?

### Questions for Customer Discovery
- "What due diligence do you do before committing critical changes? How long does it take?"
- "Have you ever forgotten to update a related file and caused an incident?"
- "Do you know who owns the code you're about to change? Do you ask them before changing it?"
- "Have you ever repeated a past incident because you didn't know it happened?"
- "When using AI to generate code, how do you know it's safe to commit?"

---

## Related Documents

**Product Strategy:**
- [mvp_vision.md](mvp_vision.md) - MVP scope and vision
- [competitive_analysis.md](competitive_analysis.md) - Market positioning
- [user_personas.md](user_personas.md) - Ben (solo dev), Clara (tech lead)

**Implementation:**
- [../03-implementation/status.md](../03-implementation/status.md) - Current implementation status
- [../03-implementation/NEXT_STEPS.md](../03-implementation/NEXT_STEPS.md) - Technical roadmap
- **[mvp_development_plan.md](mvp_development_plan.md)** - Functional requirements (NEW)

**Customer Discovery:**
- **[customer_discovery_findings.md](customer_discovery_findings.md)** - Interview notes (CREATE THIS)

---

## Action Items (Start This Week)

### Immediate (Today/Tomorrow)
1. **Create customer list** - Identify 5-8 developer contacts to interview
2. **Schedule first 3 calls** - Email contacts, schedule for next week
3. **Start LLM integration** - Begin implementing OpenAI/Anthropic client

### This Week
1. Complete LLM integration
2. Run 3 customer discovery calls
3. Document findings in [customer_discovery_findings.md](customer_discovery_findings.md)

### Next Week
1. Integration testing with real repos
2. Run 2 more customer calls
3. Performance optimization

---

**Last Updated:** October 21, 2025
**Next Review:** Weekly - update with customer discovery findings and technical progress
**Decision Point:** Week 4 (November 18, 2025) - GO/NO-GO based on data
