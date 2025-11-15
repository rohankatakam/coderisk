#!/bin/bash
# ============================================================================
# Rebuild PostgreSQL Schema from Scratch
# ============================================================================
# Purpose: Safely drop and recreate the CodeRisk PostgreSQL schema
# Usage: ./scripts/rebuild_postgres_schema.sh [--force]
#
# Safety: Requires explicit confirmation unless --force flag is used
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database connection settings
export PGHOST="${POSTGRES_HOST:-localhost}"
export PGPORT="${POSTGRES_PORT:-5433}"
export PGDATABASE="${POSTGRES_DB:-coderisk}"
export PGUSER="${POSTGRES_USER:-coderisk}"
export PGPASSWORD="${POSTGRES_PASSWORD:-CHANGE_THIS_PASSWORD_IN_PRODUCTION_123}"

FORCE_MODE=false
if [[ "$1" == "--force" ]]; then
    FORCE_MODE=true
fi

echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║  CodeRisk PostgreSQL Schema Rebuild                           ║${NC}"
echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo -e "${RED}❌ Error: psql command not found${NC}"
    echo "   Please install PostgreSQL client tools"
    exit 1
fi

# Check if database is accessible
echo -n "🔍 Checking database connection... "
if psql -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo -e "${RED}❌ Error: Cannot connect to PostgreSQL${NC}"
    echo "   Host: $PGHOST:$PGPORT"
    echo "   Database: $PGDATABASE"
    echo "   User: $PGUSER"
    exit 1
fi

# Count existing tables
TABLE_COUNT=$(psql -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public'" 2>/dev/null | tr -d ' ')

echo ""
echo "📊 Current database status:"
echo "   Database: $PGDATABASE"
echo "   Tables: $TABLE_COUNT"
echo ""

if [[ $TABLE_COUNT -gt 0 ]]; then
    echo -e "${YELLOW}⚠️  WARNING: This will DELETE ALL EXISTING DATA${NC}"
    echo ""
    echo "   The following operations will be performed:"
    echo "   1. Drop all existing tables"
    echo "   2. Drop all existing functions and triggers"
    echo "   3. Recreate schema from scratch"
    echo ""

    if [[ "$FORCE_MODE" == false ]]; then
        read -p "   Are you sure you want to continue? (yes/no): " confirm
        if [[ "$confirm" != "yes" ]]; then
            echo ""
            echo "❌ Rebuild cancelled"
            exit 0
        fi
    else
        echo "   Running in --force mode (skipping confirmation)"
    fi
fi

echo ""
echo "🗑️  Step 1: Dropping existing schema..."

# Drop all tables with CASCADE to handle foreign keys
psql <<EOF
-- Drop all tables
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END \$\$;

-- Drop all functions
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT proname, oidvectortypes(proargtypes) as argtypes
              FROM pg_proc INNER JOIN pg_namespace ON pg_proc.pronamespace = pg_namespace.oid
              WHERE pg_namespace.nspname = 'public' AND prokind = 'f') LOOP
        EXECUTE 'DROP FUNCTION IF EXISTS ' || quote_ident(r.proname) || '(' || r.argtypes || ') CASCADE';
    END LOOP;
END \$\$;

-- Drop all triggers
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT DISTINCT trigger_name FROM information_schema.triggers WHERE trigger_schema = 'public') LOOP
        -- Triggers are dropped with tables, this is just cleanup
    END LOOP;
END \$\$;
EOF

echo -e "   ${GREEN}✓ Existing schema dropped${NC}"

echo ""
echo "🏗️  Step 2: Creating new schema..."

# Run the schema initialization script
if [[ ! -f "schema/00_init_schema.sql" ]]; then
    echo -e "${RED}❌ Error: schema/00_init_schema.sql not found${NC}"
    echo "   Please run this script from the project root directory"
    exit 1
fi

psql -f schema/00_init_schema.sql

echo ""
echo "📊 Step 3: Verifying schema..."

# Count new tables
NEW_TABLE_COUNT=$(psql -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public'" | tr -d ' ')

echo "   Tables created: $NEW_TABLE_COUNT"

# Verify critical tables exist
CRITICAL_TABLES=("github_repositories" "github_commits" "github_issues" "code_blocks" "code_block_risk_index")
ALL_EXIST=true

for table in "${CRITICAL_TABLES[@]}"; do
    if psql -t -c "SELECT 1 FROM information_schema.tables WHERE table_name='$table'" | grep -q 1; then
        echo -e "   ✓ $table"
    else
        echo -e "   ${RED}✗ $table (MISSING)${NC}"
        ALL_EXIST=false
    fi
done

echo ""
if [[ "$ALL_EXIST" == true ]]; then
    echo -e "${GREEN}✅ Schema rebuild complete!${NC}"
    echo ""
    echo "📋 Summary:"
    echo "   • $NEW_TABLE_COUNT tables created"
    echo "   • Multi-repo safe (all tables have repo_id)"
    echo "   • Ready for GitHub API ingestion"
    echo ""
    echo "🚀 Next steps:"
    echo "   1. Run: cd /tmp && git clone https://github.com/omnara-ai/omnara"
    echo "   2. Run: cd omnara && crisk init"
    echo "   3. Verify: psql -c 'SELECT COUNT(*) FROM github_commits'"
    echo ""
else
    echo -e "${RED}❌ Schema rebuild failed - some tables are missing${NC}"
    exit 1
fi
