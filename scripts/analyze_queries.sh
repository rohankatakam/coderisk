#!/bin/bash
# Analyze Cypher queries to identify index candidates
# Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 1

echo "=== Query Pattern Analysis ==="
echo ""

echo "1. File path lookups (Layer 1):"
grep -r "File {path:" internal/ cmd/ 2>/dev/null | wc -l

echo "2. File file_path lookups (Layer 2):"
grep -r "File {file_path:" internal/ cmd/ 2>/dev/null | wc -l

echo "3. Commit SHA lookups (Layer 2):"
grep -r "Commit {sha:" internal/ cmd/ 2>/dev/null | wc -l

echo "4. Developer email lookups (Layer 2):"
grep -r "Developer {email:" internal/ cmd/ 2>/dev/null | wc -l

echo "5. Function unique_id lookups (Layer 1):"
grep -r "Function {unique_id:" internal/ cmd/ 2>/dev/null | wc -l

echo "6. Incident ID lookups (Layer 3):"
grep -r "Incident {id:" internal/ cmd/ 2>/dev/null | wc -l

echo ""
echo "=== Index Recommendations ==="
echo "Each lookup above benefits from a unique constraint or index."
echo ""
echo "Recommended indexes:"
echo "  - File.path (unique constraint)"
echo "  - File.file_path (index)"
echo "  - Commit.sha (unique constraint)"
echo "  - Developer.email (unique constraint)"
echo "  - Commit.author_date (range index for temporal queries)"
echo "  - Incident.id (unique constraint)"
