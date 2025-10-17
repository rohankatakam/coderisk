# Documentation Workflow for AI-Assisted Updates

**Purpose:** Guide for Claude Code agents updating documentation intelligently
**Last Updated:** October 2, 2025
**Design Philosophy:** Based on [12-factor agents](12-factor-agents-main/README.md) principles

> This workflow implements context engineering principles (Factor 3) and structured control flow (Factor 8): providing AI agents with decision trees, reading strategies, and quality checks to maximize documentation accuracy while minimizing token usage and preventing redundancy.

---

## Quick Start for AI Agents

**When user asks to update documentation:**

1. **Read this file first** (you are here)
2. **Check 12-factor principles** (see [Consult 12-Factor Principles](#consult-12-factor-principles) below)
3. **Determine update type** (see Decision Tree below)
4. **Read relevant documents** (see Reading Strategy)
5. **Make updates** (see Update Guidelines)
6. **Update cross-references** (see Cross-Reference Checklist)
7. **Cite 12-factor principles** where applied (see Citation Guidelines)

> **For code implementation (not documentation):** Use [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md) instead of this guide

---

## Decision Tree: Where Does This Content Go?

```
START: New information or context to document
│
├─ Is it a REQUIREMENT or HIGH-LEVEL ARCHITECTURE change?
│  YES → Update spec.md (REQUIRED)
│  │     Then update relevant supporting docs
│  NO  → Continue
│
├─ Is it PRODUCT/BUSINESS related?
│  │  (user needs, market analysis, competitive intel, pricing strategy)
│  YES → 00-product/
│  │     ├─ Existing doc? → Update it
│  │     └─ New topic? → Create new file, add to 00-product/README.md
│  NO  → Continue
│
├─ Is it an ARCHITECTURE DECISION?
│  │  (technology choice, pattern adoption, significant trade-off)
│  YES → 01-architecture/decisions/
│  │     ├─ Create new ADR (next number in sequence)
│  │     ├─ Update 01-architecture/README.md
│  │     └─ Reference from spec.md if it changes requirements
│  NO  → Continue
│
├─ Is it TECHNICAL ARCHITECTURE details?
│  │  (system design, graph schema, agent algorithm, deployment)
│  YES → 01-architecture/
│  │     ├─ cloud_deployment.md? → Infrastructure, BYOK, pricing
│  │     ├─ graph_ontology.md? → Graph structure, relationships
│  │     ├─ agentic_design.md? → Agent strategy, investigation
│  │     └─ New architectural area? → Create new doc, update README
│  NO  → Continue
│
├─ Is it OPERATIONAL strategy?
│  │  (scaling, costs, multi-tenancy, caching, lifecycle management)
│  YES → 02-operations/
│  │     ├─ team_and_branching.md? → Team sharing, branch deltas
│  │     ├─ public_caching.md? → Public repos, garbage collection
│  │     └─ New ops topic? → Create new doc, update README
│  NO  → Continue
│
├─ Is it IMPLEMENTATION guidance?
│  │  (status update, how-to, integration guide, roadmap)
│  YES → 03-implementation/
│  │     ├─ status.md? → Component status, blockers, next steps
│  │     ├─ phases/? → Phase roadmap details
│  │     ├─ integration_guides/? → Step-by-step tutorials
│  │     └─ Update README if adding new guide
│  NO  → Continue
│
├─ Is it RESEARCH or EXPERIMENT?
│  │  (hypothesis, prototype, performance test, exploration)
│  YES → 04-research/active/
│  │     ├─ Create new research doc (use template in 04-research/README.md)
│  │     ├─ When complete → Move to 04-research/archive/
│  │     └─ If successful → Create ADR in 01-architecture/decisions/
│  NO  → Continue
│
└─ Is it DEPRECATING existing content?
   YES → 99-archive/
         ├─ Add deprecation notice to top of file
         ├─ Move to 99-archive/
         ├─ Remove references from active docs
         └─ Update 99-archive/README.md
```

---

## Consult 12-Factor Principles

Before proceeding with documentation CRUD operations, check if any 12-factor principles apply to your specific task. This ensures our documentation aligns with proven AI agent development practices.

### How to Use 12-Factor Principles

1. **Start here:** Read [12-factor-agents-main/README.md](12-factor-agents-main/README.md) to see all 12 factors
2. **Identify relevant factors** based on your documentation task (see mapping below)
3. **Read applicable factor documents** from [12-factor-agents-main/content/](12-factor-agents-main/content/)
4. **Apply principles** to your documentation update
5. **Cite the factor** in your documentation (see Citation Guidelines below)

### Factor Relevance Mapping

Use this guide to determine which factors to consult based on your documentation task:

| Documentation Topic | Relevant 12-Factor Principles | Files to Read |
|-------------------|-------------------------------|---------------|
| **Context Engineering** (prompt design, RAG, documentation structure) | Factor 3: Own your context window | [factor-03-own-your-context-window.md](12-factor-agents-main/content/factor-03-own-your-context-window.md) |
| **Prompt Design** (system prompts, instructions, templates) | Factor 2: Own your prompts | [factor-02-own-your-prompts.md](12-factor-agents-main/content/factor-02-own-your-prompts.md) |
| **Tool Design** (API design, structured outputs) | Factor 4: Tools are structured outputs | [factor-04-tools-are-structured-outputs.md](12-factor-agents-main/content/factor-04-tools-are-structured-outputs.md) |
| **Control Flow** (workflow design, decision trees, state machines) | Factor 8: Own your control flow | [factor-08-own-your-control-flow.md](12-factor-agents-main/content/factor-08-own-your-control-flow.md) |
| **Error Handling** (error messages, debugging, recovery) | Factor 9: Compact Errors into Context Window | [factor-09-compact-errors.md](12-factor-agents-main/content/factor-09-compact-errors.md) |
| **Agent Architecture** (agent decomposition, specialization) | Factor 10: Small, Focused Agents | [factor-10-small-focused-agents.md](12-factor-agents-main/content/factor-10-small-focused-agents.md) |
| **State Management** (execution state, business logic) | Factor 5: Unify execution state and business state | [factor-05-unify-execution-state.md](12-factor-agents-main/content/factor-05-unify-execution-state.md) |
| **Human-in-Loop** (approvals, feedback, escalation) | Factor 7: Contact humans with tool calls | [factor-07-contact-humans-with-tools.md](12-factor-agents-main/content/factor-07-contact-humans-with-tools.md) |

**Note:** You don't need to follow every principle rigidly - use them as **guidance and structure** for making informed decisions.

### Citation Guidelines

When you apply a 12-factor principle in documentation, add a citation like this:

**In-line citation:**
```markdown
Our agent uses a decision tree to route requests to specialized sub-agents *(12-factor: Factor 8 - Own your control flow)*.
```

**Footnote citation:**
```markdown
## Architecture Decision

We structured our prompt templates to maximize token efficiency[^1].

[^1]: See [Factor 3: Own your context window](12-factor-agents-main/content/factor-03-own-your-context-window.md)
```

**Section header citation:**
```markdown
## Context Engineering *(Factor 3)*

[Content about context engineering...]
```

---

## Reading Strategy

### Before Making ANY Changes

**ALWAYS read these first:**
1. **spec.md** - Check if change affects requirements
2. **README.md** - Understand current structure
3. **DOCUMENTATION_WORKFLOW.md** - This file (ensures you follow process)

> **Note:** For code implementation changes, use [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md) reading strategy instead

### For Specific Update Types

**Requirement changes:**
```
Read: spec.md
Update: spec.md (REQUIRED)
Then read: All docs referenced in spec.md section 1.6
Then update: Relevant supporting docs
```

**Architecture decision:**
```
Read: spec.md (constraints)
Read: 01-architecture/README.md (current architecture)
Read: 01-architecture/decisions/README.md (ADR template)
Create: New ADR in 01-architecture/decisions/NNN-title.md
Update: 01-architecture/README.md (add to list)
If affects requirements: Update spec.md
```

**Operational strategy:**
```
Read: spec.md (relevant NFRs)
Read: 02-operations/README.md
Read: Relevant existing ops docs
Update or create: Document in 02-operations/
Update: 02-operations/README.md if new doc
```

**Implementation update:**
```
Read: 03-implementation/status.md (current state)
Update: status.md or create guide in integration_guides/
Update: 03-implementation/README.md if new guide
```

**Research/experiment:**
```
Read: 04-research/README.md (template)
Create: New doc in 04-research/active/
Use: Research template from README
When done: Move to archive, create ADR if successful
```

---

## Update Guidelines

### Rule 1: spec.md is Ground Truth

**CRITICAL:** If update affects:
- Requirements (functional or non-functional)
- System architecture (high-level)
- Technology stack
- Constraints
- Scope

**MUST update spec.md FIRST**, then supporting docs.

**Example:**
```
User: "We're adding real-time notifications"
Agent:
1. Update spec.md section 1.5 (add to scope)
2. Update spec.md section 3.X (add FRs)
3. Update spec.md section 4.X (add NFRs)
4. Create ADR in 01-architecture/decisions/
5. Update cloud_deployment.md (infrastructure)
```

### Rule 2: No Redundancy

**Before creating new document, check:**
- Does this content fit in existing doc?
- Would it duplicate existing content?
- Is there a better home in current structure?

**Examples of redundancy to AVOID:**
- ❌ Creating "cost_optimization.md" when team_and_branching.md already covers costs
- ❌ Creating "graph_schema.md" when graph_ontology.md exists
- ❌ Duplicating infrastructure details across multiple docs

**Merge instead:**
- ✅ Add cost section to existing operational doc
- ✅ Expand existing architecture doc with new details
- ✅ Reference other docs instead of duplicating

### Rule 3: One Source of Truth Per Topic

**Each concept should have ONE authoritative location:**

| Topic | Authoritative Doc | Other Docs |
|-------|------------------|------------|
| Requirements | spec.md | Others reference spec.md |
| Graph ontology | 01-architecture/graph_ontology.md | Others reference it |
| Team sharing | 02-operations/team_and_branching.md | Others reference it |
| Public caching | 02-operations/public_caching.md | Others reference it |
| Implementation status | 03-implementation/status.md | Others reference it |

**When updating a topic:**
1. Find authoritative doc
2. Update it
3. Check all docs that reference it
4. Update cross-references if needed

### Rule 4: High-Level Concepts Only

**Documentation should contain:**
- ✅ Concepts and decisions
- ✅ Architecture diagrams (ASCII/Mermaid)
- ✅ Data models (high-level schemas)
- ✅ Workflow descriptions
- ✅ Design rationale

**Documentation should NOT contain:**
- ❌ Implementation code (goes in codebase)
- ❌ Detailed code examples (minimal examples only)
- ❌ Line-by-line algorithms (describe approach, not code)
- ❌ SQL/code snippets (describe schema, don't implement)

**Exception:** Minimal code for clarity is OK, but keep it conceptual.

### Rule 5: Cross-Reference Maintenance

**After ANY update, check:**
1. Does spec.md reference this doc? Update if needed.
2. Do other docs reference this doc? Verify links still accurate.
3. Did you create new doc? Add to parent README.md.
4. Did you change doc name/location? Update ALL references.

**Tools for finding references:**
```bash
# Find all references to a document
grep -r "filename.md" dev_docs/

# Find broken links
grep -r "](.*\.md)" dev_docs/ | grep -v "http"
```

---

## Cross-Reference Checklist

### When Creating New Document

- [ ] Added to parent directory README.md
- [ ] Added to dev_docs/README.md (if major doc)
- [ ] Referenced from spec.md (if architectural/requirements-related)
- [ ] Used correct relative paths in links
- [ ] Added "Last Updated" date
- [ ] Linked to related documents

### When Updating Existing Document

- [ ] Updated "Last Updated" date
- [ ] Checked if spec.md needs updating
- [ ] Verified cross-references still accurate
- [ ] Updated parent README.md if major changes

### When Deprecating Document

- [ ] Added deprecation notice to top
- [ ] Moved to 99-archive/
- [ ] Removed references from active docs
- [ ] Updated spec.md to remove references
- [ ] Added to 99-archive/README.md

### When Renaming/Moving Document

- [ ] Updated ALL references in all docs
- [ ] Updated parent README.md
- [ ] Updated spec.md if referenced there
- [ ] Verified no broken links

**Critical paths to check:**
```
spec.md section 1.6
dev_docs/README.md
01-architecture/README.md
02-operations/README.md
03-implementation/README.md
04-research/README.md
```

---

## Anti-Patterns to Avoid

### ❌ Creating Documentation Bloat

**DON'T:**
- Create new doc when existing doc covers topic
- Duplicate content across multiple files
- Create docs for minor implementation details
- Add excessive code examples

**DO:**
- Expand existing documents
- Reference other docs instead of duplicating
- Keep docs high-level and conceptual
- Link to code for details

### ❌ Breaking Single Source of Truth

**DON'T:**
- Update supporting doc without updating spec.md (if requirements changed)
- Duplicate requirements across multiple docs
- Have conflicting information in different docs

**DO:**
- Update spec.md first for requirement changes
- Reference spec.md from supporting docs
- Maintain consistency across all docs

### ❌ Orphaning Documents

**DON'T:**
- Create doc without adding to README
- Leave broken cross-references
- Forget to update "Last Updated" date

**DO:**
- Add all new docs to parent README
- Verify all links after changes
- Update timestamps on edits

### ❌ Losing Historical Context

**DON'T:**
- Delete deprecated docs (move to 99-archive/)
- Remove ADRs (mark as deprecated instead)
- Erase failed experiments (document learnings)

**DO:**
- Preserve history in 99-archive/
- Mark ADRs as superseded, don't delete
- Archive research with results

---

## AI Agent Workflow

### Step-by-Step Process

**1. Understand the Request**
```
User request → Parse intent → Classify update type
Example: "We need to add webhooks for branch updates"
Classification: Implementation guide + architecture decision
```

**2. Read Context**
```
Read: spec.md (check if in scope)
Read: 03-implementation/status.md (current state)
Read: 01-architecture/cloud_deployment.md (infrastructure)
Read: 02-operations/team_and_branching.md (branch lifecycle)
```

**3. Determine Action**
```
Decision tree result:
- Update spec.md? YES (new NFR for webhook reliability)
- Create ADR? YES (webhook technology choice)
- Update existing doc? YES (team_and_branching.md)
- Create new doc? NO (fits in existing docs)
```

**4. Make Updates**
```
1. Update spec.md section 4.4 (add webhook NFR)
2. Create 01-architecture/decisions/002-webhook-strategy.md
3. Update 02-operations/team_and_branching.md (webhook section)
4. Update 03-implementation/status.md (add webhook to roadmap)
5. Add to 03-implementation/integration_guides/webhooks.md
```

**5. Update Cross-References**
```
1. Add ADR to 01-architecture/decisions/README.md
2. Add integration guide to 03-implementation/README.md
3. Update spec.md section 1.6 (if new major doc)
4. Verify all links work
5. Update "Last Updated" dates
```

**6. Verify Consistency**
```
Check: spec.md aligns with supporting docs
Check: No redundant content created
Check: All cross-references work
Check: READMEs updated
```

### Example Prompts for Claude Code

**For requirement changes:**
```
"I need to update the documentation to reflect that we're now supporting
GitLab in addition to GitHub. Please:
1. Start with spec.md and identify all sections that need updating
2. Update all relevant architecture and operations documents
3. Create any necessary ADRs for technology decisions
4. Ensure all cross-references are updated"
```

**For new features:**
```
"We're adding a graph visualization dashboard. Please:
1. Read spec.md to see if this is in/out of scope
2. If out of scope, update spec.md to add it
3. Create necessary architecture documents or update existing ones
4. Create implementation guides
5. Update all cross-references"
```

**For deprecating features:**
```
"We're removing the local-first mode. Please:
1. Update spec.md to remove it from scope
2. Move related architecture docs to 99-archive/
3. Add deprecation notices
4. Remove all cross-references from active docs
5. Update implementation status"
```

---

## Document Templates

### Quick Reference

| Document Type | Template Location |
|--------------|-------------------|
| ADR | 01-architecture/decisions/README.md |
| Research | 04-research/README.md |
| Product doc | 00-product/README.md |
| Integration guide | 03-implementation/README.md |

### Minimal Document Template

```markdown
# [Title]

**Purpose:** [One sentence description]
**Last Updated:** YYYY-MM-DD

---

## [Section 1]

[Content]

---

## Key Concepts

[High-level concepts only]

---

## Integration Points

[How this relates to other docs/systems]

---

**See also:**
- [spec.md](spec.md)
- [Related doc](path/to/doc.md)
```

---

## Quality Checks

### Before Committing Changes

**Run through this checklist:**

- [ ] **Accuracy:** Information is correct and up-to-date
- [ ] **Consistency:** Aligns with spec.md and other docs
- [ ] **Completeness:** No missing cross-references
- [ ] **Clarity:** High-level and understandable
- [ ] **Conciseness:** No redundant content
- [ ] **Structure:** Follows existing patterns
- [ ] **Links:** All cross-references work
- [ ] **Timestamps:** "Last Updated" dates updated

### Red Flags

**Stop and reconsider if:**
- Creating >3 new documents at once (likely redundant)
- Document >20KB (likely too detailed or should be split)
- Duplicating content from another doc (reference instead)
- Creating doc not fitting any directory (structure issue)
- Adding implementation code (belongs in codebase)

---

## Maintenance Schedule

### Weekly
- [ ] Update 03-implementation/status.md (component progress)

### Monthly
- [ ] Review 04-research/active/ (move completed to archive)
- [ ] Check for broken links across all docs
- [ ] Verify spec.md aligns with recent changes

### Quarterly
- [ ] Review 99-archive/ (permanently delete very old docs if appropriate)
- [ ] Update all "Last Updated" dates for reviewed docs
- [ ] Verify all ADRs still reflect current decisions

---

## Emergency Procedures

### If Documentation Becomes Inconsistent

1. **Identify source of truth:** spec.md
2. **Compare all docs to spec.md**
3. **Update divergent docs to align**
4. **Add consistency checks to this guide**

### If Structure Breaks Down

1. **Don't panic, don't delete**
2. **Document current state** in a new file in 04-research/active/
3. **Propose new structure** with rationale and migration plan
4. **Get approval before major restructuring**
5. **Update all docs in single commit**

---

## Contact & Questions

**If this guide is unclear:**
- Add clarification to this document
- Update examples to be more specific
- Create issue/discussion for team review

**Remember:** This guide should evolve. If you find better patterns, update this document.

---

**This workflow ensures:**
✅ Consistent documentation structure
✅ No redundancy or bloat
✅ Single source of truth maintained
✅ Cross-references always current
✅ AI agents can update intelligently
✅ Human reviewers can easily verify
