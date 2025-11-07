# The Universe 1 vs Universe 2 Demo
## The Visceral Product Demo

**Duration:** 2-3 minutes
**Audience:** Customers (CTOs, VPEs, Engineering Managers), Investors (after the Retrospective Audit)
**Goal:** Make them *feel* the pain of the problem and the relief of the solution

---

## Pre-Demo Setup

**What you need:**
- A terminal with `crisk` installed
- A local clone of the target repo (e.g., Supabase)
- A browser window open to a real revert PR from that repo
- A prepared `.diff` file of the buggy change (extracted from the revert PR)
- `crisk init` already completed (graph is built)

**Your posture:** You are the narrator, but you will embody *the developer* when you type. This is immersive storytelling.

---

## The Script

### [0:00-0:30] The Hook: Start with Their Actual Pain

**YOU:** "Every company has a 'commit that got away.' For Supabase, it was this one."

*(Show browser with the revert PR: https://github.com/supabase/supabase/pull/39866)*

**YOU:** *(Point to the PR title: `fix: revert 39818`)* "On October 25th, 2024, the Supabase team merged this emergency hotfix. They were reverting **this** pull request..."

*(Click to show the original buggy PR #39818)*

**YOU:** "...a change to their table editor that looked fine, passed code review, passed CI, and then broke production. This was a SEV-1 incident. All hands on deck. Customers couldn't edit their database tables."

**YOU:** *(Pause for effect)* "The worst part? This was 100% preventable."

---

### [0:30-1:00] Universe 1: The World Without Context

**YOU:** "Let me show you what happened. This is **Universe 1**—the world we live in today."

*(Switch to your terminal. You're now the developer.)*

**YOU:** "I'm the developer who made this change. I'm in my flow state. I've been working on this feature for 3 hours. I've staged my code. I'm about to commit."

*(Show the terminal prompt. Don't type yet.)*

```bash
[~/supabase]$ git status
On branch fix/table-editor-regression
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
        modified:   studio/components/table-editor/TableEditor.tsx
```

**YOU:** "I have no idea this file has caused three production incidents in the last six months. I have no idea the original owner of this code left the company two months ago. I have no idea this is a P0 critical user flow."

**YOU:** "So I do what every developer does. I commit it."

*(Type the command, but don't press Enter yet)*

```bash
[~/supabase]$ git commit -m "fix: improve table editor performance"
```

**YOU:** "I push it. It goes to code review. My reviewer, who's in three meetings today, has 45 seconds to look at this. They see a clean diff, they see the tests pass, they approve it."

*(Press Enter. Show the commit succeeding.)*

**YOU:** "The bad code ships. Production breaks. We spend four hours diagnosing it. We burn $1.2 million in downtime. We write a postmortem. And we promise ourselves, 'We'll be more careful next time.'"

**YOU:** *(Look at camera)* "This is Universe 1. This is the default-to-trust equilibrium."

---

### [1:00-1:15] The Pivot: What If There Was Context?

**YOU:** "Now let me show you **Universe 2**—the world with CodeRisk."

*(Reset the terminal state. You're about to replay the same scenario.)*

```bash
[~/supabase]$ git reset HEAD~1  # Undo the commit
[~/supabase]$ git status
On branch fix/table-editor-regression
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
        modified:   studio/components/table-editor/TableEditor.tsx
```

**YOU:** "Same developer. Same flow state. Same buggy code staged. But this time, *before* I type `git commit`, I run one command."

---

### [1:15-1:30] The Hero Moment: Running crisk check

**YOU:** "This is the moment that changes everything."

*(Type the command slowly, deliberately)*

```bash
[~/supabase]$ crisk check
```

*(Press Enter. Pause for 2 seconds while the command runs. This is the "magic moment.")*

**YOU:** "In under 10 seconds, I get this."

---

### [1:30-2:30] The Output: The Risk Report

*(The output appears. This is the payoff. Read it out loud.)*

```
🔴 CRITICAL RISK: studio/components/table-editor/TableEditor.tsx

═══════════════════════════════════════════════════════════════════════
  MANAGER-VIEW: POTENTIAL BUSINESS IMPACT
═══════════════════════════════════════════════════════════════════════
  • 🚨 This change touches a P0 (Critical) user flow: Table Editing
  • 🔥 This file has been linked to 3 prior production incidents
  • ⏳ This code is stale (owned by an inactive developer)

  📊 BUSINESS IMPACT ESTIMATE:
    • Estimated MTTR if this breaks: 4.2 hours
    • Historical cost of incidents in this file: $1.2M
    • Recommended action: Require senior engineer review
───────────────────────────────────────────────────────────────────────

═══════════════════════════════════════════════════════════════════════
  DEVELOPER-VIEW: ACTIONABLE INSIGHTS
═══════════════════════════════════════════════════════════════════════

  Why is this risky?
    • This file caused 3 production incidents in the last 6 months:
      - #31201: [BUG] Row editor crashes on save (2025-05-10)
        └─ Root cause: Null pointer in row validation logic
      - #28440: [BUG] Editing JSONB field fails (2025-02-17)
        └─ Root cause: Type coercion error in cell renderer
      - #25001: [BUG] Table editor hangs on load (2024-11-20)
        └─ Root cause: Infinite loop in pagination logic

    • Stale code ownership:
      └─ Original owner (Sarah Chen) last touched this file 94 days ago
      └─ Sarah left the team on 2024-08-15
      └─ No clear current owner assigned

    • Co-change risk detected:
      └─ This file historically changes with `useTableQuery.ts` (78% co-change rate)
      └─ `useTableQuery.ts` was NOT modified in this commit
      └─ Past incidents often occurred when only one file was changed

  What should you do?
    1. 📖 Review past incidents (#31201, #28440, #25001) for context
    2. 👤 Ping 'Jake Anderson' (most recent contributor) for pre-review
    3. ✅ Add regression tests covering the past incident scenarios
    4. 🔍 Verify `useTableQuery.ts` doesn't need to be updated

═══════════════════════════════════════════════════════════════════════
  CONFIDENCE: 91% (based on CLQS Score: 94 - World-Class)
  Analysis powered by Phase 2 Agent (GPT-4 + Neo4j graph traversal)
═══════════════════════════════════════════════════════════════════════

⚠️  If you believe this is a false positive:
  → Run: crisk explain --override
  → Document why you're overriding this risk assessment
  → Your explanation will be added to your PR description for reviewers
```

---

### [2:30-2:50] The Breakdown: Translating the Output

**YOU:** *(Point to the "Manager-View" section)*

"This top section is what my *manager* cares about. It tells them this is a P0 flow that has burned us before. The estimated cost of failure is $1.2 million."

**YOU:** *(Point to the "Developer-View" section)*

"This bottom section is what *I* care about. It doesn't just say 'this is risky'—it proves *why* with three specific past incidents. It tells me the original owner left the company. It tells me there's a co-change pattern I missed."

**YOU:** *(Point to the "What should you do?" checklist)*

"And critically, it gives me an *actionable checklist*. I don't need to bother my reviewer yet. I can:"

- "Read the past incident reports to understand what broke before."
- "Ping Jake, who recently touched this code, for a quick 5-minute sanity check."
- "Add regression tests before I even commit."

**YOU:** "In Universe 1, I'd waste 30 minutes of my reviewer's time and risk a production fire. In Universe 2, I fix this in 10 minutes while I'm still in my flow state, and the bad code *never reaches the reviewer*."

---

### [2:50-3:00] The Counterfactual: What Happens Next

**YOU:** "So what do I do now? I have three options."

*(Show the options on screen or verbally enumerate them)*

```
Option 1: Fix it myself
  → I read incident #31201, realize my change has the same null-pointer bug
  → I fix it in 5 minutes
  → I add a regression test
  → I commit with confidence

Option 2: Get a pre-review
  → I ping Jake on Slack: "Hey, can you look at this for 5 min before I commit?"
  → Social cost is LOW (pre-review vs. blocking a PR)
  → Jake spots the issue in 3 minutes
  → I fix it

Option 3: Override with documentation
  → I run: crisk explain --override
  → I document: "I reviewed incident #31201. My change doesn't touch the validation logic."
  → This explanation gets added to my PR automatically
  → My reviewer has context even if I was wrong
```

**YOU:** "In every scenario, the *context* prevents the fire. The developer has the information at the right moment—pre-commit, in their flow state—instead of relying on a time-constrained reviewer to reconstruct it."

---

### [3:00-3:15] The Contrast: Side-by-Side Comparison

**YOU:** "Let me show you what the reviewer would have seen in Universe 1."

*(Show the GitHub diff of the original buggy PR in the browser)*

```diff
diff --git a/studio/components/table-editor/TableEditor.tsx b/studio/components/table-editor/TableEditor.tsx
index 1234567..abcdefg 100644
--- a/studio/components/table-editor/TableEditor.tsx
+++ b/studio/components/table-editor/TableEditor.tsx
@@ -45,7 +45,7 @@ export const TableEditor = () => {
   const handleSave = (row: Row) => {
-    if (row.data) {
+    if (row) {
       saveRow(row)
     }
   }
```

**YOU:** "This is it. A one-line change. Looks fine. The tests pass. The linter passes. CodeRabbit would say, 'Looks good!'"

**YOU:** *(Switch back to the terminal output)*

"But with CodeRisk, we know this *exact pattern*—removing a null check—caused incident #31201. We know this file is a minefield. We know the developer doesn't own this code."

**YOU:** "This is the difference between *code quality* and *regression risk*. CodeRabbit checks quality. CodeRisk prevents regressions."

---

### [3:15-3:30] The Close: The Value Proposition

**YOU:** "This is what we built. It's not a linter. It's not an AI reviewer. It's a **pre-commit risk scanner** that gives developers the historical context they need to make better decisions."

**YOU:** "It runs locally, in under 10 seconds. It doesn't block your workflow. It doesn't require you to change your process. You just type `crisk check` before you commit."

**YOU:** "And every time you run it, it's querying a knowledge graph built from your entire codebase history—every commit, every incident, every code review pattern. It's giving you the same context a senior engineer with 10 years at your company would have, but instantly."

**YOU:** *(Final line)* "In Universe 2, the production fire we started with *never happens*. That's the product."

---

## Key Talking Points (Reference Sheet)

### If asked: "What if I ignore the CRITICAL warning?"

**YOU:** "You can. We're not code cops. But if you override it, we ask you to run `crisk explain --override` and document *why* you think it's safe.

That explanation gets automatically added to your PR description. So when your reviewer looks at it, they see:

```
⚠️ CodeRisk flagged this as CRITICAL (Score: 87)
Developer override explanation:
  'I reviewed incident #31201. My change doesn't touch the
   validation logic that caused that fire. I added a regression
   test to confirm.'
```

Now your reviewer has context even if you were wrong. And if you *were* wrong and it causes an incident, that override becomes training data that improves the model."

---

### If asked: "How fast is this?"

**YOU:** "Under 10 seconds for a typical commit. The graph is pre-built during `crisk init` (which runs overnight or on-demand). The `crisk check` command is just a graph traversal + LLM synthesis.

We're paranoid about performance because the *only* reason this works is that it doesn't break flow state. If it took 60 seconds, developers would skip it."

---

### If asked: "What if we don't have good incident data?"

**YOU:** "That's what the CLQS score is for. When you run `crisk init`, we'll give you a score from 0-100 that measures your data quality.

If you score below 70, we'll tell you upfront: 'Your data isn't clean enough for accurate risk assessment. Here are the gaps we found.' We'll show you which patterns are missing—commit message discipline, issue linking, etc.

We won't take your money if we can't deliver 75%+ accuracy. This builds trust and ensures our case studies are always real."

---

### If asked: "How is this different from running `git blame`?"

**YOU:** "Great question. Let me show you what `git blame` gives you."

*(Run the command in the terminal)*

```bash
[~/supabase]$ git blame studio/components/table-editor/TableEditor.tsx | head -10

a3f2c1d9 (Sarah Chen    2024-05-12 14:23:45 -0700  42) const handleSave = (row: Row) => {
b8e4f3a2 (Jake Anderson 2024-08-03 09:15:33 -0700  43)   if (row.data) {
a3f2c1d9 (Sarah Chen    2024-05-12 14:23:45 -0700  44)     saveRow(row)
c7a9b2f1 (Sarah Chen    2024-05-12 14:23:45 -0700  45)   }
d4e1a8c3 (Michael Liu   2024-03-22 11:08:19 -0700  46) }
```

**YOU:** "This tells you *who* wrote each line and *when*. That's it. It doesn't tell you:

- Sarah left the company two months ago
- This file caused three SEV-1 incidents
- Each of those incidents was a null-pointer bug in this exact function
- This file co-changes with `useTableQuery.ts` 78% of the time

`git blame` is a single data source. CodeRisk is the *integration* of five data sources—Git, Jira, PagerDuty, GitHub, and the AST—correlated into a risk score."

---

### If asked: "Does this work for all languages?"

**YOU:** "We support the top 12 languages via TreeSitter: JavaScript, TypeScript, Python, Go, Rust, Java, C++, C#, Ruby, PHP, Swift, Kotlin.

The *incident linking* works for any language because that's based on Git and issue trackers. The *code coupling* analysis (co-change patterns, dependency graphs) requires TreeSitter to parse the AST.

If your language isn't supported, we'll tell you during `crisk init`. We're adding more languages based on customer demand."

---

## The Demo Variants

### Variant A: Live Coding (High Risk, High Reward)

If you're confident and the audience is technical:

1. Don't use a pre-made `.diff` file
2. Actually write the buggy code live in an editor (e.g., remove a null check)
3. Stage it with `git add`
4. Run `crisk check` live
5. Show the output

**Pro:** Maximum authenticity. Shows the tool working in real-time.
**Con:** Risk of technical failure (network issues, slow LLM, etc.)

---

### Variant B: Pre-Recorded Output (Low Risk, Lower Impact)

If you're presenting to non-technical executives or on a Zoom call with bad wifi:

1. Pre-run `crisk check` and save the output to a text file
2. During the demo, show the file instead of running the command live
3. Narrate as if it's happening in real-time

**Pro:** Zero technical risk. Guaranteed consistent output.
**Con:** Less immersive. Skeptical engineers will notice.

---

### Variant C: Hybrid (Recommended)

1. Pre-run `crisk check` and have the output ready in a separate terminal tab
2. Run the command live
3. If it fails or is slow (>15 seconds), switch to the pre-run tab seamlessly

**Pro:** Best of both worlds. Looks live but has a backup.
**Con:** Requires practice to switch tabs without the audience noticing.

---

## Success Metrics

You know this demo worked if:

- **Customers:** They lean forward when you show the output. They ask, "Can we try this on our codebase today?"
- **Investors:** They ask about false positive rates (engagement with the substance) instead of "How is this different from CodeRabbit?" (confused positioning)
- **Engineers:** They ask, "What happens if I ignore the warning?" (thinking about adoption) instead of "What if the data is wrong?" (skeptical dismissal)

If they say, "This is cool, but our code review process works fine"—you've failed. The revert PR at the beginning should make that statement impossible.

---

## The Call-to-Action (How You Close)

**For customers:**

**YOU:** "Here's what I'd like to do. Give us read-only access to your repo and your incident data—PagerDuty, Sentry, or just a list of your 'Revert' PRs from the last 12 months."

**YOU:** "We'll run the same retrospective analysis we showed Supabase. In two weeks, we'll come back with a report showing you which of your past fires were predictable and what your team would have seen if you'd had CodeRisk."

**YOU:** "If the ROI is there, we do a 30-day pilot with your team. If it's not, you get a free audit and we part as friends. Deal?"

---

**For investors:**

**YOU:** "We're raising a $1.5M pre-seed to turn this proof-of-concept into a product. We have two design partners signed (Supabase, [Partner 2]) and a pipeline of 15 companies who've requested the audit."

**YOU:** "The capital goes toward three things: (1) Expanding the issue-linking engine to cover more incident platforms, (2) Hiring a founding engineer to scale the graph infrastructure, and (3) Building the self-serve onboarding flow so companies can run the audit themselves."

**YOU:** "We're targeting $10M ARR in 36 months with a usage-based pricing model at $50/dev/month. Based on the design partner feedback, we're seeing 60% conversion from audit to paid pilot."

**YOU:** "I'd love to send you our deck and hop on a follow-up call next week to go deeper on the unit economics. Does [Day/Time] work?"

---

## Final Checklist

Before you run this demo, ensure:

- [ ] You have a real revert PR from a recognizable company (Supabase, Stripe, Vercel, etc.)
- [ ] You've run `crisk init` on their repo and verified the CLQS score is >70
- [ ] You've pre-run `crisk check` on the buggy commit to verify the output is compelling
- [ ] You've practiced the narrative flow at least 3 times (aim for <3 minutes)
- [ ] You have a backup plan if the live demo fails (pre-recorded output in a separate tab)
- [ ] You can answer the "What if I ignore it?" and "How is this different from git blame?" questions without hesitation

This demo is your **closer**. The retrospective audit gets them intellectually curious. This demo makes them *feel* the pain and the solution. Together, they're unstoppable.
