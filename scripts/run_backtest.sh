#!/bin/bash

# CodeRisk Backtesting Runner
# This script runs comprehensive backtesting against ground truth data

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "🚀 CodeRisk Backtesting Framework"
echo "═══════════════════════════════════"

# Parse arguments
GROUND_TRUTH="${PROJECT_ROOT}/test_data/omnara_ground_truth.json"
REPO_ID=1
OUTPUT_DIR="${PROJECT_ROOT}/test_results"
VERBOSE=false
RUN_TEMPORAL=true
RUN_SEMANTIC=true
RUN_CLQS=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --ground-truth)
            GROUND_TRUTH="$2"
            shift 2
            ;;
        --repo-id)
            REPO_ID="$2"
            shift 2
            ;;
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --no-temporal)
            RUN_TEMPORAL=false
            shift
            ;;
        --no-semantic)
            RUN_SEMANTIC=false
            shift
            ;;
        --no-clqs)
            RUN_CLQS=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --ground-truth PATH   Path to ground truth JSON file (default: test_data/omnara_ground_truth.json)"
            echo "  --repo-id ID          Repository ID in database (default: 1)"
            echo "  --output DIR          Output directory for reports (default: test_results)"
            echo "  --verbose             Enable verbose logging"
            echo "  --no-temporal         Skip temporal validation"
            echo "  --no-semantic         Skip semantic validation"
            echo "  --no-clqs             Skip CLQS benchmark"
            echo "  --help                Show this help message"
            echo ""
            echo "Example:"
            echo "  $0 --ground-truth test_data/omnara_ground_truth.json --repo-id 1"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
done

# Verify ground truth file exists
if [ ! -f "$GROUND_TRUTH" ]; then
    echo "❌ Error: Ground truth file not found: $GROUND_TRUTH"
    exit 1
fi

echo "📖 Configuration:"
echo "  Ground Truth: $GROUND_TRUTH"
echo "  Repository ID: $REPO_ID"
echo "  Output Directory: $OUTPUT_DIR"
echo "  Verbose: $VERBOSE"
echo "  Run Temporal: $RUN_TEMPORAL"
echo "  Run Semantic: $RUN_SEMANTIC"
echo "  Run CLQS: $RUN_CLQS"
echo ""

# Build the backtest binary
echo "🔨 Building backtest binary..."
cd "$PROJECT_ROOT"
go build -o bin/backtest cmd/backtest/main.go
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "  ✓ Build successful"
echo ""

# Prepare arguments
ARGS="--ground-truth=$GROUND_TRUTH --repo-id=$REPO_ID --output=$OUTPUT_DIR"

if [ "$VERBOSE" = true ]; then
    ARGS="$ARGS --verbose"
fi

if [ "$RUN_TEMPORAL" = false ]; then
    ARGS="$ARGS --temporal=false"
fi

if [ "$RUN_SEMANTIC" = false ]; then
    ARGS="$ARGS --semantic=false"
fi

if [ "$RUN_CLQS" = false ]; then
    ARGS="$ARGS --clqs=false"
fi

# Run the backtest
echo "🧪 Running backtesting..."
echo "═══════════════════════════════════"
echo ""

./bin/backtest $ARGS

EXIT_CODE=$?

echo ""
echo "═══════════════════════════════════"
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Backtesting completed successfully!"
else
    echo "❌ Backtesting failed (exit code: $EXIT_CODE)"
fi
echo "═══════════════════════════════════"

exit $EXIT_CODE
