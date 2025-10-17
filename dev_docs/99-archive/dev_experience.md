# CodeRisk Developer Experience: The Pre-Commit Risk Oracle

## Strategic Positioning

CodeRisk operates in the competitive "code intelligence" space alongside tools like Greptile and Codescene, but with a fundamentally different developer experience philosophy:

- **Greptile: The Conversational PR Co-pilot** - Centers on augmenting pull requests with smart peer review and conversational feedback within the formal review cycle.

- **Codescene: The Architectural Health Guardian** - Focuses on strategic dashboards for team leads, visualizing technical debt and long-term maintainability trends.

- **CodeRisk: The Pre-Commit Risk Oracle** - A decisive, predictive tool that answers one critical question instantly: **"Is it safe to proceed?"** It targets the developer's chaotic inner loop, long before code reaches the polished PR stage.

**The Unaddressed Niche:** CodeRisk serves the rapid, local, pre-commit sanity check - the workflow of developers on teams who make dozens of small, iterative commits before polishing for peer review. They need a command they can run instinctively:

`git commit -am "wip" && crisk check`

This addresses the core pain of **developer uncertainty** at the moment of creation, preventing architectural and temporal regressions that traditional tools miss.

---

## Core User Stories

### **User Story 1: Ben, The Anxious Junior Developer**

- **Persona:** Ben, 6 months into his first software engineering job.
- **Company:** "Synapse," a 150-person, fast-growing AI startup.
- **Team:** Growth & Onboarding.
- **Git Habits (The "Save Frequently" Method):** Ben is terrified of breaking production. His local feature branch (`git checkout -b ben/JIRA-451-update-signup-flow`) is a chaotic history of dozens of tiny commits with messages like `"WIP"`, `"fix typo"`, `"trying something"`, and `"oops revert"`. He knows he's supposed to clean it up with `git rebase` before making a pull request, but he's not confident doing it and fears losing his work. His PRs are often hard to review because of this messy history.

**The Scenario (Saturday, 11:00 AM PDT):**

Ben is working on a weekend to catch up on a ticket that involves changing the user signup flow. He's touching a file called `onboarding_controller.py`, which seems straightforward. He's made about 15 small commits on his branch. Before attempting the dreaded interactive rebase, he remembers a tool his Tech Lead mentioned. He decides to run it as a final "sanity check."

Bash

# 

`# Ben is on his feature branch with a messy commit history
git branch
* ben/JIRA-451-update-signup-flow

# He runs the check on his changes compared to the main branch
crisk check --target main`

**The "Aha!" Moment:**

The output is a shock.

Plaintext

# 

`ANALYZING CHANGES...

CodeRisk Score: 78/100 🟠 HIGH RISK

Top Concerns:
1.  🔗 Hidden Coupling: Changes in `onboarding_controller.py` are highly correlated with changes in `analytics_event_tracker.py`. You have not modified the analytics tracker, which may break event logging for new user signups.
2.  👻 Ghost Dependency: The function `send_welcome_email()` you modified is also used by a legacy `AdminUserCreation` service, which is not covered by your tests.`

Ben feels a pit in his stomach, followed by immense relief. He had no idea the analytics tracker was connected; it wasn't imported anywhere he could see. He greps the codebase and finds a dynamic call to it. He would have *never* caught this. The "Ghost Dependency" is even scarier—he would have broken an internal tool the support team relies on.

**Adoption and Success:**

CodeRisk becomes Ben's personal safety net. He integrates it into his chaotic workflow: `write code` -> `git commit -am "WIP"` -> `crisk check`. The tool acts as an automated senior developer, pointing out the "unknown unknowns." This gives him the confidence to refactor his code *before* asking for a review. His PRs become cleaner, he learns the codebase's hidden pathways faster, and his fear of pushing code slowly transforms into confidence.

**Team Value:** Ben's team benefits from shared ingestion - the expensive repository analysis was performed once and cached for all team members. When Ben runs `crisk check`, he gets instant results (<2 seconds) using the team's shared risk data, making this powerful analysis economically viable for his entire team.

---

### **User Story 2: Clara, The Skeptical Principal Engineer**

- **Persona:** Clara, Principal Engineer with 15 years of experience.
- **Company:** "FinSecure," a large, publicly-traded financial services company.
- **Team:** Core Platform & Architecture.
- **Git Habits (The "Surgical Precision" Method):** Clara's Git hygiene is legendary. She works on a feature branch, but before she ever creates a pull request, she uses `git rebase -i main` to squash her work into a single, perfect, atomic commit with a beautifully written commit message. She believes a clean history is non-negotiable and gets frustrated reviewing PRs with 20+ trivial commits.

**The Scenario (Tuesday, 2:00 PM PDT):**

Clara is reviewing a pull request from a different team that touches her team's core API. The PR itself is clean—a single, well-explained commit. The code *looks* fine, the tests pass, and it follows all style guides. Yet, she has a nagging feeling of unease. The changes are in a notoriously tricky part of the system. While pondering, she remembers an internal blog post about a new tool, CodeRisk, being integrated into their GitHub checks. She clicks "Details" on the CodeRisk check.

**The "Aha!" Moment:**

The check is bright red: `CRITICAL RISK`.

Plaintext

# 

`CodeRisk Score: 96/100 🔴 CRITICAL RISK

Top Concerns:
1.  🎲 G² Surprise: The files `transaction_ledger.go` and `currency_exchange_rate_cache.go` have been modified together. Our historical analysis shows these two files have *never* been changed in the same commit before. This represents a statistically significant and highly unusual coupling that suggests a new, unvetted architectural pattern is being introduced.
2.  🏛️ Architectural Violation: This change introduces a direct dependency from the `Ledger` (a core system of record) to a `Cache` (a transient data store). This violates the architectural principle of data flow direction and could lead to data integrity issues under high load.`

Clara is stunned. CodeRisk didn't just find a bug; it found a deep, philosophical flaw in the change. The "G² Surprise" metric gave her the concrete data to back up her gut feeling. No linter or unit test would ever have caught this. It was a risk that existed purely in the realm of architectural experience.

**Adoption and Success:**

Clara becomes CodeRisk's biggest champion. She mandates it as a required check for any PR touching critical systems. For her, the tool elevates code review. It automates the "first pass" of architectural and historical analysis, freeing up her and other senior engineers to focus on the truly strategic questions. It becomes her "second brain," instantly recalling years of project history to find subtle, dangerous patterns she might have otherwise missed on a busy afternoon.

**Enterprise Value:** Clara's company benefits from enterprise deployment with custom LLM endpoints and on-premises data storage. The shared repository ingestion means her 100-person engineering team gets sophisticated risk analysis without per-developer API costs, making advanced code intelligence economically sustainable at scale.

---

### **User Story 3: Team Collaboration (Open Source Context)**

- **Scenario:** React.js open source project with hundreds of contributors
- **Challenge:** Maintainers need to review PRs from unknown contributors quickly and safely
- **Solution:** Sponsor-funded CodeRisk cache provides instant risk assessment

**The Team Development Flow:**

```bash
# New contributor clones repo
git clone github.com/facebook/react
cd react

# CodeRisk auto-discovers public cache on first check
crisk check  # No setup needed!
> ✓ Found public cache for facebook/react
> Sponsored by Meta • Last updated: 6 hours ago
> Current risk: LOW ✓

# After making changes
crisk check
> Risk: MEDIUM
> Concerns: Modifying core reconciler without tests
> Suggestion: Add test coverage before submitting PR
```

**GitHub Integration:**

When the PR is opened, CodeRisk posts a simple status check:
- ✅ **CodeRisk: Low Risk** (for safe changes)
- 🔴 **CodeRisk: Critical Risk** (with link to detailed analysis)

**Value for Teams:**

1. **Maintainers** get instant risk assessment without running analysis
2. **Contributors** get immediate feedback before submitting PRs
3. **Community** benefits from shared, sponsor-funded infrastructure
4. **Quality** improves through early detection of risky patterns

---

## Key Differentiators

### Workflow Moat
CodeRisk becomes as reflexive as `git status` - a command developers run without thinking. The sub-2-second response time creates a habit-forming experience that's difficult for competitors to replicate.

### Economic Model
The shared ingestion architecture makes sophisticated analysis affordable for teams:
- **One-time ingestion cost** (~$15-50) shared across entire team
- **Individual checks** cost $0 (local cache) to $0.04 (cloud enhancement)
- **ROI** through prevented incidents and faster development cycles

### Technical Depth
Unlike simple linters, CodeRisk analyzes:
- **Hidden coupling** through temporal co-change patterns
- **Architectural violations** via historical pattern analysis
- **Blast radius** calculations using graph theory
- **Incident correlation** with past production issues

This creates a pre-commit experience that no other tool currently provides.