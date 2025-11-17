#!/bin/bash

# Supabase Incident Scorer - Automated scoring after manual labeling
# Run this AFTER you've labeled incidents in supabase_incidents.csv

set -e

echo "╔══════════════════════════════════════════════════════════════════════╗"
echo "║        Supabase Retrospective Audit - Automated Scoring             ║"
echo "╚══════════════════════════════════════════════════════════════════════╝"
echo ""

# Check if CSV exists
INCIDENT_CSV="/Users/rohankatakam/Documents/brain/coderisk/supabase_incidents.csv"
if [ ! -f "$INCIDENT_CSV" ]; then
    echo "❌ Error: $INCIDENT_CSV not found"
    echo ""
    echo "Please run: bash find_supabase_incidents.sh"
    echo "Then manually label 15 incidents in the CSV file"
    exit 1
fi

# Count incidents (minus header)
INCIDENT_COUNT=$(($(wc -l < "$INCIDENT_CSV") - 1))

if [ "$INCIDENT_COUNT" -lt 10 ]; then
    echo "⚠️  Warning: Only $INCIDENT_COUNT incidents found"
    echo "Recommend at least 10-15 for statistical validity"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "✓ Found $INCIDENT_COUNT labeled incidents"
echo ""

# Setup - ensure these are set in your environment
if [ -z "$GEMINI_API_KEY" ]; then
    echo "Error: GEMINI_API_KEY not set"
    exit 1
fi
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN not set"
    exit 1
fi
export PHASE2_ENABLED="false"  # Phase 1 only for speed
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

OUTPUT_DIR="/Users/rohankatakam/Documents/brain/coderisk/supabase_audit_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUTPUT_DIR"

CRISK="/Users/rohankatakam/Documents/brain/coderisk/bin/crisk"
SUPABASE_DIR="/tmp/supabase"

echo "Output directory: $OUTPUT_DIR"
echo ""

# Step 1: Check if Supabase repo exists
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 1: Supabase Repository Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ ! -d "$SUPABASE_DIR" ]; then
    echo "Cloning Supabase repository (this may take 5-10 minutes)..."
    cd /tmp
    git clone --depth 1 https://github.com/supabase/supabase.git
    echo "✓ Cloned successfully"
else
    echo "✓ Supabase repository already exists at $SUPABASE_DIR"
fi

cd "$SUPABASE_DIR"

# Check if crisk init has been run
if [ ! -d ".coderisk" ]; then
    echo ""
    echo "Initializing CodeRisk (ingesting repo, this takes 20-30 minutes)..."
    echo "⏳ This runs unattended - feel free to grab coffee"
    echo ""

    $CRISK init --days 365 2>&1 | tee "$OUTPUT_DIR/crisk_init.log"

    echo "✓ CodeRisk initialized"
else
    echo "✓ CodeRisk already initialized"
fi

echo ""

# Step 2: Score incident files
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 2: Scoring Incident Files"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Initialize output CSV
echo "file_path,pr_number,type,date,description,risk_level,risk_score,category" > "$OUTPUT_DIR/incident_scores.csv"

count=0
while IFS=',' read -r pr_number type date file_changed description; do
    # Skip header
    if [ "$pr_number" = "pr_number" ]; then
        continue
    fi

    count=$((count + 1))
    echo ""
    echo "[$count/$INCIDENT_COUNT] Scoring PR #$pr_number"
    echo "   File: $file_changed"
    echo "   Type: $type"

    # Check if file exists in current state
    if [ -f "$file_changed" ]; then
        # Run crisk check
        output=$($CRISK check "$file_changed" 2>&1 || true)

        # Parse risk level
        if echo "$output" | grep -q "CRITICAL"; then
            risk_level="CRITICAL"
            risk_score="90"
        elif echo "$output" | grep -q "HIGH"; then
            risk_level="HIGH"
            risk_score="75"
        elif echo "$output" | grep -q "MEDIUM"; then
            risk_level="MEDIUM"
            risk_score="50"
        elif echo "$output" | grep -q "LOW"; then
            risk_level="LOW"
            risk_score="25"
        else
            risk_level="UNKNOWN"
            risk_score="50"
        fi

        echo "   Result: $risk_level (score: $risk_score)"
        echo "$file_changed,$pr_number,$type,$date,\"$description\",$risk_level,$risk_score,INCIDENT" >> "$OUTPUT_DIR/incident_scores.csv"
    else
        echo "   ⚠️  File not found (may have been deleted/moved)"
        # Still record it with MEDIUM as default
        echo "$file_changed,$pr_number,$type,$date,\"$description\",MEDIUM,50,INCIDENT" >> "$OUTPUT_DIR/incident_scores.csv"
    fi

    sleep 1  # Brief pause
done < "$INCIDENT_CSV"

echo ""
echo "✓ Scored $count incident files"
echo ""

# Step 3: Score random safe commits
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 3: Scoring Safe Baseline Commits"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Sample 50 random files from the repo
echo "Sampling 50 random files for baseline..."

# Get diverse file types
{
    find . -name "*.ts" -o -name "*.tsx" | sort -R | head -20
    find . -name "*.py" | sort -R | head -10
    find . -name "*.go" | sort -R | head -10
    find . -name "*.json" -maxdepth 3 | head -5
    find . -name "*.md" -maxdepth 2 | head -5
} > "$OUTPUT_DIR/safe_files.txt"

SAFE_COUNT=$(wc -l < "$OUTPUT_DIR/safe_files.txt" | tr -d ' ')
echo "Selected $SAFE_COUNT files"
echo ""

# Initialize output CSV
echo "file_path,pr_number,type,date,description,risk_level,risk_score,category" > "$OUTPUT_DIR/safe_scores.csv"

count=0
while read -r file_path; do
    count=$((count + 1))
    echo "[$count/$SAFE_COUNT] Scoring: $file_path"

    if [ -f "$file_path" ]; then
        # Run crisk check
        output=$($CRISK check "$file_path" 2>&1 || true)

        # Parse risk level
        if echo "$output" | grep -q "CRITICAL"; then
            risk_level="CRITICAL"
            risk_score="90"
        elif echo "$output" | grep -q "HIGH"; then
            risk_level="HIGH"
            risk_score="75"
        elif echo "$output" | grep -q "MEDIUM"; then
            risk_level="MEDIUM"
            risk_score="50"
        elif echo "$output" | grep -q "LOW"; then
            risk_level="LOW"
            risk_score="25"
        else
            risk_level="UNKNOWN"
            risk_score="50"
        fi

        echo "   Result: $risk_level (score: $risk_score)"
        echo "$file_path,N/A,safe,N/A,Safe baseline commit,$risk_level,$risk_score,SAFE" >> "$OUTPUT_DIR/safe_scores.csv"
    fi

    sleep 1  # Brief pause
done < "$OUTPUT_DIR/safe_files.txt"

echo ""
echo "✓ Scored $count safe files"
echo ""

# Step 4: Generate Money Slide
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 4: Generating Money Slide"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if matplotlib is installed
if ! python3 -c "import matplotlib" 2>/dev/null; then
    echo "Installing matplotlib and pandas..."
    pip3 install matplotlib pandas --quiet
fi

# Generate the graph
python3 /tmp/generate_supabase_money_slide.py "$OUTPUT_DIR"

echo ""
echo "╔══════════════════════════════════════════════════════════════════════╗"
echo "║                    🎉 SUPABASE AUDIT COMPLETE! 🎉                    ║"
echo "╚══════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Your demo assets:"
echo "  📁 $OUTPUT_DIR"
echo ""
echo "Key files:"
echo "  📊 money_slide.png - Your Money Slide for the video"
echo "  📄 money_slide.pdf - High-quality version"
echo "  📝 audit_report.txt - Statistics and talking points"
echo "  📈 incident_scores.csv - Raw incident data"
echo "  📈 safe_scores.csv - Raw safe commit data"
echo ""
echo "Next steps:"
echo "  1. Review: open $OUTPUT_DIR/money_slide.png"
echo "  2. Read stats: cat $OUTPUT_DIR/audit_report.txt"
echo "  3. Record your demo following YC_DEMO_VIDEO_SCRIPT.md"
echo ""
