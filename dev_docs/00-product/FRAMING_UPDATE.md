# Product Framing Update: Due Diligence Over Technical Features

**Date:** October 21, 2025
**Context:** Customer discovery question review revealed misalignment with core value proposition

---

## The Problem with Original Framing

### Original Customer Discovery Question (WRONG):
> "If I showed you a tool that tells you 'this change is high risk because files X and Y always change together', would you use it?"

**Why this failed:**
- ❌ Feature-focused, not outcome-focused
- ❌ Abstract technical concept ("temporal coupling")
- ❌ No clear action implied
- ❌ Doesn't connect to developer pain points

---

## The Correct Framing (From Dev Docs)

### Core Value Proposition:
> **"Automate the due diligence developers should do before committing"**

### Critical Questions We Answer:
1. **Ownership:** Who owns this code? Should I coordinate with them?
2. **Blast Radius:** What files depend on this change? What might break?
3. **Incident History:** Has this pattern caused failures before? Am I repeating history?
4. **Forgotten Updates:** Did I forget to update related files that usually change together?

### NOT:
- ❌ "Temporal coupling detector"
- ❌ "Graph-based static analysis tool"
- ❌ "Pre-commit risk assessment"

### YES:
- ✅ "Pre-commit due diligence assistant"
- ✅ "Automated regression prevention through context awareness"
- ✅ "Prevents incidents by surfacing ownership, dependencies, and history"

---

## Revised Customer Discovery Questions

### Problem Discovery

**Question 1: Due Diligence Workflow**
> "Walk me through your last commit. What checks did you do BEFORE committing to make sure it was safe?"
>
> Look for: Manual due diligence they do (or skip), time spent, what they worry about

**Question 2: Regression Incident**
> "Tell me about a time when you committed code that broke something unexpectedly. What happened?"
>
> Look for: Was it a regression? Did they forget to update a related file? Ownership issue?
> Follow-up: "Could you have known this would break before committing?"

**Question 3: Critical File Changes**
> "Before making a change to a critical file (like payment processing), what do you wish you knew?"
>
> Look for: Who owns it? What depends on it? Past incidents? Test coverage?
> **This is YOUR value prop** - do they articulate this need?

**Question 4: Coordination Issues**
> "Have you ever changed a file and later realized someone else was working on the same file for a different feature?"
>
> Look for: Ownership/coordination pain, merge conflicts, stepping on toes

**Question 5: AI Code Trust**
> "What's the most frustrating part of using AI to generate code? How do you know it's safe?"
>
> Look for: Trust issues, uncertainty, fear of breaking things, time spent reviewing

### Solution Validation (3 Scenarios)

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

---

## Updated Product Messaging

### Elevator Pitch (Before)
> "CodeRisk uses graph analysis and temporal coupling detection to identify risky code changes before you commit."

### Elevator Pitch (After)
> "CodeRisk automates the due diligence you should do before committing: Who owns this code? What depends on it? Has this failed before? It's like having a senior developer review your changes in 5 seconds."

### Feature Descriptions (Before → After)

**Temporal Coupling Detection**
- Before: "Detects files that frequently change together using co-change analysis"
- After: "Warns you when you're about to change a file without updating related files that usually change together—prevents forgotten updates"

**Ownership Analysis**
- Before: "Identifies code owners using git blame and commit history"
- After: "Shows you who owns the code you're changing and who recently worked on it—helps you coordinate before making changes"

**Incident Linking**
- Before: "Links code changes to past production incidents using issue tracking integration"
- After: "Warns you when you're about to make a change similar to one that caused an incident before—prevents repeating history"

**Blast Radius Analysis**
- Before: "Calculates dependency graph to measure change impact"
- After: "Shows you what files depend on your change and what might break—helps you test the right things"

---

## Updated LLM Prompt Structure

### Old Prompt Focus:
```
Calculate risk metrics:
- Coupling score
- Co-change frequency
- Test coverage ratio
- Complexity metrics

Determine risk level: LOW/MEDIUM/HIGH
```

### New Prompt Focus (Due Diligence Questions):
```
Answer these due diligence questions:

1. OWNERSHIP: Should the developer coordinate with file owner or recent contributors?
2. BLAST RADIUS: What dependent files might break? What needs checking?
3. FORGOTTEN UPDATES: Based on co-change patterns, what files likely need updating?
4. INCIDENT PREVENTION: Is this similar to a past incident? What to do differently?
5. TEST COVERAGE: Is coverage adequate for this change?

Provide actionable coordination plan and forgotten update warnings.
```

---

## Updated Output Examples

### Standard Output (Before)
```
⚠️  HIGH RISK - 2 files need attention

payment_processor.py (HIGH)
  • High coupling (15 dependencies)
  • Frequent co-changes with fraud_detector.py (0.85)
  • 3 past incidents

Recommendations:
  1. Review fraud_detector.py for coupled changes
  2. Add tests for payment_processor.py
```

### Standard Output (After - Due Diligence Framing)
```
⚠️  HIGH RISK - Pre-commit due diligence needed

payment_processor.py (HIGH)
  📋 DUE DILIGENCE CHECKLIST:

  👤 OWNERSHIP
     • Last modified by Alice 2 days ago (bug fix INC-453)
     • Bob owns this file (80% of commits)
     → Consider pinging @alice or @bob before making changes

  🔗 BLAST RADIUS
     • 15 files depend on this (fraud_detector.py, reporting.py, ...)
     → Changes here may break downstream systems

  🔄 FORGOTTEN UPDATES?
     • fraud_detector.py changed with this file in 17/20 commits (85%)
     → You likely need to update fraud_detector.py too

  ⚠️  INCIDENT HISTORY
     • 3 past incidents linked to this file
     • INC-453 (2 days ago): Timeout cascade in payment flow
     → Similar changes caused failures recently

RECOMMENDATIONS:
  1. Coordinate with @alice (she just fixed INC-453 here)
  2. Review fraud_detector.py - likely needs update too
  3. Add tests for payment_processor.py (0% coverage)
```

---

## Key Takeaways

### What We Learned:
1. **Technical features ≠ Value proposition**
   - Temporal coupling is HOW we deliver value, not the value itself
   - Developers care about outcomes (prevent incidents) not methods (graph analysis)

2. **Framing matters for customer discovery**
   - Abstract questions get polite responses
   - Concrete scenarios reveal genuine pain points

3. **Our docs were right, initial framing was wrong**
   - mvp_vision.md clearly states "Automated Due Diligence Before Code Review"
   - developer_experience.md shows concrete due diligence examples
   - Customer discovery questions should match this framing

### Actions Taken:
- ✅ Updated [strategic_decision_framework.md](strategic_decision_framework.md) with revised customer discovery questions
- ✅ Updated [mvp_development_plan.md](mvp_development_plan.md) with due diligence framing throughout
- ✅ Updated LLM prompt template to focus on due diligence questions
- ✅ Updated output examples to show due diligence checklist format
- ✅ Added core value proposition section to development plan header

### What Stays the Same:
- ✅ Technical implementation (graph construction, temporal analysis, LLM integration)
- ✅ Architecture (local-first, Neo4j, Phase 1 + Phase 2)
- ✅ Performance targets (<200ms, <5s)
- ✅ Competitive positioning (pre-commit vs. PR review)

**What Changes:**
- ✅ How we talk about features to customers
- ✅ How LLM interprets and presents results
- ✅ Output format emphasizes actionable due diligence
- ✅ Customer discovery questions focus on outcomes not features

---

## For Implementation

When building Phase 2 LLM integration, ensure:

1. **Context Fetching** focuses on due diligence data:
   - Ownership (who to coordinate with)
   - Blast radius (what might break)
   - Co-change patterns (what updates were forgotten)
   - Incident history (what failed before)

2. **Prompt Template** asks due diligence questions:
   - "Should developer coordinate with owner?"
   - "What dependent files might break?"
   - "What files likely need updating based on patterns?"
   - "Is this similar to past incidents?"

3. **Output Formatting** presents as actionable checklist:
   - ✅ Use checklist format (📋 DUE DILIGENCE CHECKLIST)
   - ✅ Section headers: OWNERSHIP, BLAST RADIUS, FORGOTTEN UPDATES, INCIDENT HISTORY
   - ✅ Actionable arrows (→) showing what to do
   - ✅ Priority emojis (🔴 CRITICAL, 🟡 HIGH, 🟢 MEDIUM)

4. **JSON Output** includes due diligence fields:
   - `ownership`, `coordination_needed`, `forgotten_updates`, `incident_risk`
   - NOT just: `risk_level`, `metrics`, `reasoning`

---

**Last Updated:** October 21, 2025
**Related Documents:**
- [strategic_decision_framework.md](strategic_decision_framework.md) - Customer discovery questions
- [mvp_development_plan.md](mvp_development_plan.md) - Technical requirements with due diligence framing
- [mvp_vision.md](mvp_vision.md) - Core product vision (source of truth)
- [developer_experience.md](developer_experience.md) - UX examples showing due diligence workflow
