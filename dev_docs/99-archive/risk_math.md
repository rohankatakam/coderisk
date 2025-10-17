## **1) Purpose**

Give developers instant, **local**, explainable risk on **uncommitted or PR** changes across a large, moving repo. Core focus: **regression risk**, plus adjacent risks (API, schema, deps, perf, concurrency, security, config, tests, merge).

---

## **2) Inputs**

- **Working tree / PR diff:** file list, hunks, churn, renamed/moved info.
- **90-day sample (or chosen window W):** commits/PRs/issues/owners, import & co-change degrees, CODEOWNERS history.
- **Indexes:** IMPORTS & CO_CHANGED (Memgraph), embeddings/FT search (Postgres+pgvector).
- **Optional:** tests→files map (or coverage export), lockfiles, migration files, service configs.

---

## **3) Core Signals (Blueprint v4, bounded)**

*All graph queries ≤2 hops, degree-capped, ≤500 ms.*

1. **Δ-Diffusion Blast Radius (ΔDBR):** local PPR/HKPR delta from proposed edge deltas over IMPORTS/CO_CHANGED.
2. **Hawkes-Decayed Co-Change (HDCC):** two-timescale (fast/slow) decays for evolutionary coupling.
3. **G² Surprise:** Dunning log-likelihood for unusual file pairs (bounded triples on small degrees).
4. **Ownership Authority Mismatch (OAM):** owner_churn, minor_edit_ratio, experience_gap, bus_entropy.
5. **Span-Core & Bridge Risk:** temporal core persistence; bridging centrality in 2-hop IMPORTS subgraph.
6. **Incident Adjacency (GB-RRF):** fuse BM25 + vector kNN + local PPR ranks via Reciprocal Rank Fusion.
7. **JIT Baselines:** size (touched files; auto-High cutoff), churn, diff entropy.

*All features emit grounded evidence (file/edge/issue IDs).*

---

## **4) Micro-Detectors (new category features)**

Each runs on the **working-tree diff**, time-boxed, deterministic; returns {score∈[0,1], reasons[], anchors[]}.

1. **api_break_risk (50–150 ms):** AST diff of **public surface**; score = breaking-delta severity × importer count (from IMPORTS).
2. **schema_risk (10–40 ms):** migration ops (DROP/NOT NULL/type) × backfill/down/flag checks × table fan-out proxy.
3. **dep_risk (10–30 ms):** lockfile diff; major bumps, transitive pin churn (+ optional offline OSV).
4. **perf_risk (20–60 ms):** loop+I/O/DB call, string-concat-in-loop; weight by ΔDBR/centrality.
5. **concurrency_risk (20–60 ms):** new shared-state writes, lock order changes, missing await/mutex around shared data.
6. **security_risk (30–120 ms):** mini-SAST (unsafe YAML, SQL concat, path joins, JWT none) + secrets (regex+entropy).
7. **config_risk (10–40 ms):** k8s/tf/yaml/json risky key deltas × service fan-out proxy.
8. **test_gap_risk (10–20 ms):** (tests near T)/(churn in T) with smoothing; low ratio ⇒ high risk.
9. **merge_risk (10–20 ms):** overlap with upstream on **high-HDCC** hotspots; rebase hint.

---

## **5) Scoring & Tiering**

- **Feature vector:** [v4 signals …, api_break_risk, schema_risk, …, merge_risk] → normalize per repo/window.
- **Monotone scorer:** linear or monotone-boosted; all new features **non-decreasing** w.r.t. risk.
- **Tiers:** repo-local quantiles + **Conformal Risk Control (CRC)** to bound false-escalations (e.g., ≤5%) per segment (e.g., size bucket).
- **Auto-High rules (conservative):**
    - api_break_risk ≥0.9 with importer_count ≥ P95
    - schema_risk ≥0.9 with (DROP | NOT NULL) and no backfill
    - security_risk ≥0.9 (secret or critical sink)
- **Policy:** LLMs can **explain or escalate**, **never de-escalate**.

---

## **6) LLM Usage (strict, bounded)**

- **Search labeling & triage:** (Modes A/B) name risky conduits; propose **test names** when mapping is sparse.
- **Action planning (High only):** (Mode C) suggest mitigations; deterministic simulator scores expected Δrisk.
- **Summaries:** 1 short explanation for High citing IDs.
- **Offline rule mining:** distill recurring incidents into new deterministic rulelets.
- **Guardrails:** tool-bounded JSON I/O; token/tool budgets; no code exec; no de-escalation.

---

## **7) LLM-Guided Search (no score ownership change)**

- **Mode A — Beam (≤300 ms):** risk-directed beam over IMPORTS∪CO_CHANGED; heuristic
    
    h(n)=αΔDBR+βHDCC+γBridge+δIncidentSim+εOAM+ζCategoryHint.
    
- **Mode B — Bidirectional (≤600 ms, High only):** meet-in-the-middle S→G path proof.
- **Mode C — Constrained MCTS (≤1.5 s, High on click):** optimize tests/reviewers/canary under budgets using deterministic Δscore.

---

## **8) APIs**

- POST /assess_worktree → tier, score, **categories** (see payload), top_signals, evidence links.
- POST /score_pr → v4 features + detectors for PR SHA.
- POST /search/beam | /search/bidir → risk paths (IDs) + trace (nodes, ms).
- POST /plan/mcts → ranked mitigation plan with expected Δrisk.

**Categories payload (example):**

```
{
  "categories": {
    "regression": { "score": 0.81, "reasons": ["High ΔDBR near INC-2481"], "anchors": ["src/cache/index.ts"] },
    "api": { "score": 0.72, "reasons": ["Removed param from export foo()"], "anchors": ["src/api/foo.ts:42"] },
    "schema": { "score": 0.66, "reasons": ["ADD NOT NULL without backfill"], "anchors": ["migrations/2025_09_14.sql:12"] },
    "deps": { "score": 0.40, "reasons": ["lodash 4→5"], "anchors": ["package.json"] },
    "perf": { "score": 0.30, "reasons": ["String concat in loop"], "anchors": ["core/hotpath.ts:88"] },
    "concurrency": { "score": 0.10, "reasons": [], "anchors": [] },
    "security": { "score": 0.00, "reasons": [], "anchors": [] },
    "config": { "score": 0.20, "reasons": ["k8s timeout -60%"], "anchors": ["deploy/service.yaml:31"] },
    "tests": { "score": 0.55, "reasons": ["0 tests near changed files"], "anchors": [] },
    "merge": { "score": 0.35, "reasons": ["Upstream changed same HDCC hotspot"], "anchors": ["src/cache/index.ts"] }
  }
}
```

---

## **9) Surfaces (same engine, same thresholds)**

- **GitHub App/Action:** single required check (tier + badges) + one maintained “shadow” comment linking to Evidence.
- **Workbench (local UI):** Evidence (ΔDBR/HDCC, paths, detector reasons), Graph neighborhood viewer, Scoring Lab, record/replay.
- **MCP Server (IDE/agents):** assess_worktree, score_pr, explain_high, graph://neighborhood.

---

## **10) Performance & SLAs**

- **Score path (warm):** ≤2 s p50 / ≤5 s p95.
- **Detectors (all 9):** ~150–450 ms p50; ~700–900 ms p95 (independently skippable).
- **Graph ops:** ≤2 hops, degree caps (~200), ≤10k nodes scanned, ≤500 ms per query.
- **Storage:** unchanged from v4 (Memgraph + Postgres/pgvector); optional tiny migrations table.

---

## **11) Telemetry & Acceptance**

- Log: nodes_expanded/ms for search, detector timeouts, auto-High triggers (with anchors), CRC version/segment.
- **Ship criteria:**
    - /assess_worktree ≤2.5 s p95; ≥6/9 detector scores present.
    - Beam yields ≥1 path **or** “no path within 2 hops” badge with trace.
    - High PRs show a causal path ≥60% where recent incidents exist.
    - No LLM de-escalations; attempts logged.

---

**One-liner:** *A deterministic, local, bounded risk engine (v4) extended with nine micro-detectors and LLM-guided search for paths & plans—exposed via GitHub checks, a local Workbench, and MCP tools for instant, in-editor feedback on uncommitted diffs.*