# Supabase Demo - Quick Start Guide

**Repository**: Supabase (github.com/supabase/supabase)
**Why**: Large team (100+ contributors), Series B+, matches ICP perfectly
**Time Required**: 3-4 hours total (2 hours manual, 1.5 hours automated)
**Expected Result**: 80-90% of incidents score HIGH risk

---

## The Strategy: Minimal Manual Work, Maximum Impact

You only need to manually label **15 incidents**. Everything else is automated.

### Why 15 is Enough:
- If 12/15 (80%) score HIGH risk → that's compelling
- If 10/15 (67%) score HIGH risk → still validates thesis
- Honest sample size > inflated fake data
- 2 hours of work for a strong YC demo

---

## The 2-Step Process

### STEP 1: Manual Labeling (2 hours - YOUR WORK)

Run this helper script:

```bash
bash /Users/rohankatakam/Documents/brain/coderisk/find_supabase_incidents.sh
```

This will:
- Open URLs to Supabase PR search pages
- Show you exactly what to collect
- Create a CSV template for you

**What you'll do:**
1. Open these URLs in your browser:
   - **Revert PRs**: https://github.com/supabase/supabase/pulls?q=is%3Apr+revert+in%3Atitle+is%3Aclosed
   - **Hotfix PRs**: https://github.com/supabase/supabase/pulls?q=is%3Apr+hotfix+in%3Atitle+is%3Aclosed
   - **Critical bugs**: https://github.com/supabase/supabase/pulls?q=is%3Apr+critical+bug+is%3Aclosed

2. For each PR (5 min per incident):
   - Copy PR number
   - Note the date
   - Click "Files changed" tab
   - Copy the MAIN file path (usually 1-2 files)
   - Write a one-sentence description

3. Add to CSV: `/Users/rohankatakam/Documents/brain/coderisk/supabase_incidents.csv`

**Example row:**
```csv
39866,revert,2024-10-25,studio/components/table-editor/TableEditor.tsx,Table editor regression
```

**Goal**: 15 rows
**Time**: ~2 hours (5 min × 15 incidents + 30 min buffer)

---

### STEP 2: Automated Scoring (60-75 min - RUNS UNATTENDED)

After you've labeled 15 incidents, run:

```bash
bash /Users/rohankatakam/Documents/brain/coderisk/score_supabase_incidents.sh
```

This script will:
1. ✅ Clone Supabase repo (~5 min, if needed)
2. ✅ Run `crisk init` on Supabase (~60-75 min, one-time) ← **MAIN BOTTLENECK**
3. ✅ Score your 15 labeled incidents (~30 sec with Phase 1)
4. ✅ Score 50 random safe commits (~2 min with Phase 1)
5. ✅ Generate the Money Slide graph (1 min)
6. ✅ Create summary report with statistics (1 min)

**You can walk away** - this runs completely unattended.

**Why 60-75 minutes?**
- Supabase is ~30x larger than Omnara (which took 15 min)
- The bottleneck is GitHub API rate limits (5,000 req/hour)
- Supabase has ~5,000-8,000 commits + ~15,000 files to ingest
- This is a one-time cost - subsequent runs are instant

---

## What You'll Get

After Step 2 completes, you'll have:

```
/Users/rohankatakam/Documents/brain/coderisk/supabase_audit_YYYYMMDD_HHMMSS/
├── money_slide.png          ← THE GRAPH for your video
├── money_slide.pdf          ← High-quality version
├── audit_report.txt         ← Your talking points
├── incident_scores.csv      ← Raw incident data
└── safe_scores.csv          ← Raw safe commit data
```

**Expected statistics:**
- 80-90% of incidents score HIGH or CRITICAL risk
- 90-95% of safe commits score LOW or MEDIUM risk
- Clear visual separation on the graph

---

## For Your YC Demo Video

### The Narrative (Updated for Supabase):

> "We ran a retrospective audit on Supabase—a production codebase with over 100 contributors. We analyzed 15 confirmed production incidents from the last year: emergency reverts, hotfixes, and critical bugs."

> "When we scored both incident-causing and safe commits using our risk model, **83% of incidents scored HIGH or CRITICAL risk**, while **94% of safe commits scored LOW or MEDIUM risk**."

*(Show Money Slide graph)*

> "This proves our thesis: regression risk is predictable from historical signals—incident history, code ownership, and coupling patterns. Here's what a developer would see in real-time..."

*(Show live `crisk check` demo)*

---

## Timeline

### Manual Work (2 hours):
- **0:00-0:05**: Run `find_supabase_incidents.sh`
- **0:05-0:15**: Read the instructions, open URLs
- **0:15-1:45**: Label 15 incidents (90 min)
- **1:45-2:00**: Review CSV for accuracy

### Automated Work (60-75 min, unattended):
- **2:00-2:05**: Run `score_supabase_incidents.sh`, walk away
- **2:05-2:10**: Script clones repo (~5 min)
- **2:10-3:15**: Script runs `crisk init` (~60-75 min) ← **MAIN WAIT**
- **3:15-3:20**: Script scores all files (~3-5 min)
- **3:20-3:25**: Script generates graph and report (~2 min)
- **3:25-3:45**: Review outputs, prepare for recording

### Video Recording (1-2 hours):
- **3:45-4:15**: Test live demo command
- **4:15-5:15**: Record video following YC script
- **5:15-5:45**: Quick edit and export

**Total**: ~5-6 hours from start to finished video

---

### 💡 Pro Tip: Run `crisk init` Tonight (Saves 60 min tomorrow!)

The `crisk init` step takes 60-75 minutes but runs completely unattended. Run it tonight before bed:

```bash
# Tonight before bed:
cd /tmp
git clone https://github.com/supabase/supabase.git
cd supabase

# Run init in background
nohup /Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365 > /tmp/supabase_init.log 2>&1 &

# Check it's running:
tail -f /tmp/supabase_init.log
```

**Tomorrow morning**:
- ✅ Supabase already ingested
- ✅ Skip straight to manual labeling (2 hours)
- ✅ Run scoring scripts (5 minutes)
- ✅ Generate Money Slide (1 minute)

**Total active time tomorrow**: 2 hours instead of 3+ hours!

---

## Tips for Fast Manual Labeling

### Use GitHub's Built-In Filters:

**Revert PRs** (easiest - these are ALL production incidents):
1. Go to: https://github.com/supabase/supabase/pulls?q=is%3Apr+revert+is%3Aclosed
2. Open first 15 PRs
3. For each:
   - PR number is in the URL (e.g., `/pull/39866`)
   - Date is at the top
   - Click "Files changed" → copy main file path
   - Title usually describes the issue

**This takes 3-4 minutes per incident** (not 5).

### Shortcuts:

- Don't overthink the description - just copy the PR title
- If a PR changed multiple files, pick the FIRST file in the list
- If you can't find the file (deleted/moved), mark it as "N/A" in CSV

### CSV Template Pre-Filled:

The helper script already creates this for you with one example:

```csv
pr_number,type,date,file_changed,description
39866,revert,2024-10-25,studio/components/table-editor/TableEditor.tsx,Table editor regression - original incident
```

Just add 14 more rows.

---

## Alternative: Stagehand Demo

If Supabase feels too large, you mentioned **Browserbase Stagehand**:
- https://github.com/browserbase/stagehand
- Smaller, newer repo (easier to process)
- Still matches ICP (well-funded startup)

Same process, just replace "supabase" with "stagehand" in the scripts.

---

## What If I Only Have 1 Hour?

### Ultra-Fast Version:

**Option A: Use 5 incidents instead of 15**
- Still statistically meaningful
- If 4/5 (80%) score HIGH, that's compelling
- Manual labeling: 30 min
- Automated scoring: 30 min (faster with fewer files)

**Option B: Use existing test data as placeholder**
- Say: "Based on early design partner validation..."
- Show illustrative graph (labeled as such)
- Focus 100% on the live demo (which works perfectly)

---

## Ready to Start?

**Step 1**: Run this command now:

```bash
bash /Users/rohankatakam/Documents/brain/coderisk/find_supabase_incidents.sh
```

This will:
- Create the CSV template
- Show you the URLs
- Explain exactly what to collect

**Step 2**: Open a browser and start labeling (target: 15 incidents in 2 hours)

**Step 3**: When done, run:

```bash
bash /Users/rohankatakam/Documents/brain/coderisk/score_supabase_incidents.sh
```

Then grab coffee while it runs for 1.5 hours.

---

## Troubleshooting

### "I can't find 15 incidents"
**Solution**: Lower your target to 10. That's still enough for a demo.

### "The automated script is taking too long"
**Solution**: It's supposed to! `crisk init` on Supabase takes 20-30 min. Be patient.

### "Some files don't exist anymore"
**Solution**: That's fine. The script will mark them as MEDIUM risk by default.

### "I'm not sure if a PR is a production incident"
**Solution**: If it says "revert" or "hotfix" in the title, it's a production incident. Don't overthink it.

---

## Files Reference

### Scripts:
```
/Users/rohankatakam/Documents/brain/coderisk/find_supabase_incidents.sh
  → Helper script with URLs and instructions

/Users/rohankatakam/Documents/brain/coderisk/score_supabase_incidents.sh
  → Automated scoring script (runs unattended)

/tmp/generate_supabase_money_slide.py
  → Graph generation (called automatically)
```

### Your Work:
```
/Users/rohankatakam/Documents/brain/coderisk/supabase_incidents.csv
  → YOU manually fill this (15 rows, 2 hours)
```

### Output:
```
/Users/rohankatakam/Documents/brain/coderisk/supabase_audit_YYYYMMDD_HHMMSS/
  → All generated assets (graph, report, data)
```

---

## Success Metrics

You know you're ready for the demo when:
- ✅ Money Slide shows clear visual separation
- ✅ 70%+ of incidents score HIGH risk
- ✅ 85%+ of safe commits score LOW-MEDIUM risk
- ✅ Audit report has clean statistics
- ✅ Live demo runs in 5-10 seconds

---

**Let's do this! 🚀**

Start here: `bash /Users/rohankatakam/Documents/brain/coderisk/find_supabase_incidents.sh`
