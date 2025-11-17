-- ============================================================================
-- AGENT-P3C: Temporal Risk Calculator Verification Queries
-- ============================================================================
-- Purpose: Verify that temporal risk calculation is working correctly
-- Usage: psql -d coderisk -f verify_temporal_calculator.sql
-- ============================================================================

\echo '================================'
\echo 'AGENT-P3C Verification Report'
\echo '================================'
\echo ''

-- ============================================================================
-- Query 1: Check incident links
-- ============================================================================
\echo '1. Total Incident Links Created'
\echo '--------------------------------'

SELECT
    COUNT(*) AS total_incident_links,
    COUNT(DISTINCT code_block_id) AS unique_blocks_affected,
    COUNT(DISTINCT issue_id) AS unique_issues_linked
FROM code_block_incidents
WHERE repo_id = 1;

\echo ''

-- ============================================================================
-- Query 2: Check incident counts on blocks
-- ============================================================================
\echo '2. Incident Count Distribution'
\echo '--------------------------------'

SELECT
    incident_count,
    COUNT(*) AS number_of_blocks
FROM code_blocks
WHERE repo_id = 1
GROUP BY incident_count
ORDER BY incident_count DESC;

\echo ''

-- ============================================================================
-- Query 3: Overall statistics
-- ============================================================================
\echo '3. Overall Statistics'
\echo '--------------------------------'

SELECT
    COUNT(*) AS total_blocks,
    SUM(incident_count) AS total_incidents,
    AVG(incident_count) AS avg_incidents_per_block,
    MAX(incident_count) AS max_incidents_per_block,
    COUNT(*) FILTER (WHERE incident_count > 0) AS blocks_with_incidents,
    COUNT(*) FILTER (WHERE incident_count = 0 OR incident_count IS NULL) AS blocks_without_incidents
FROM code_blocks
WHERE repo_id = 1;

\echo ''

-- ============================================================================
-- Query 4: Verify count accuracy
-- ============================================================================
\echo '4. Count Accuracy Verification'
\echo '--------------------------------'
\echo 'Checking for any blocks where stored count does not match actual count...'

SELECT
    cb.id,
    cb.file_path,
    cb.block_name,
    cb.incident_count AS stored_count,
    COUNT(cbi.id) AS actual_count,
    (cb.incident_count - COUNT(cbi.id)) AS discrepancy
FROM code_blocks cb
LEFT JOIN code_block_incidents cbi ON cbi.code_block_id = cb.id
WHERE cb.repo_id = 1
GROUP BY cb.id, cb.file_path, cb.block_name, cb.incident_count
HAVING cb.incident_count != COUNT(cbi.id)
ORDER BY ABS(cb.incident_count - COUNT(cbi.id)) DESC;

\echo ''
\echo 'If no rows returned above, all counts are accurate!'
\echo ''

-- ============================================================================
-- Query 5: Top incident hotspots
-- ============================================================================
\echo '5. Top 10 Incident Hotspots'
\echo '--------------------------------'

SELECT
    cb.file_path,
    cb.block_name,
    cb.block_type,
    cb.incident_count,
    cb.last_modified_at,
    cb.last_modifier_email
FROM code_blocks cb
WHERE cb.repo_id = 1
  AND cb.incident_count > 0
ORDER BY cb.incident_count DESC, cb.last_modified_at DESC
LIMIT 10;

\echo ''

-- ============================================================================
-- Query 6: Recent incident links (last 10)
-- ============================================================================
\echo '6. Recent Incident Links (Last 10)'
\echo '--------------------------------'

SELECT
    cb.block_name,
    cb.file_path,
    gi.number AS issue_number,
    gi.title AS issue_title,
    gi.state AS issue_state,
    cbi.confidence,
    cbi.evidence_source,
    cbi.fix_commit_sha,
    cbi.created_at AS linked_at
FROM code_block_incidents cbi
JOIN code_blocks cb ON cb.id = cbi.code_block_id
JOIN github_issues gi ON gi.id = cbi.issue_id
WHERE cbi.repo_id = 1
ORDER BY cbi.created_at DESC
LIMIT 10;

\echo ''

-- ============================================================================
-- Query 7: Incident evidence quality
-- ============================================================================
\echo '7. Incident Evidence Quality'
\echo '--------------------------------'

SELECT
    evidence_source,
    COUNT(*) AS incident_count,
    AVG(confidence) AS avg_confidence,
    MIN(confidence) AS min_confidence,
    MAX(confidence) AS max_confidence
FROM code_block_incidents
WHERE repo_id = 1
GROUP BY evidence_source
ORDER BY incident_count DESC;

\echo ''

-- ============================================================================
-- Query 8: Blocks with multiple incidents (potential problem areas)
-- ============================================================================
\echo '8. High-Risk Blocks (3+ Incidents)'
\echo '--------------------------------'

SELECT
    cb.file_path,
    cb.block_name,
    cb.block_type,
    cb.incident_count,
    STRING_AGG(DISTINCT gi.title, ' | ' ORDER BY gi.title) AS issue_titles,
    STRING_AGG(DISTINCT gi.number::text, ', ' ORDER BY gi.number::text) AS issue_numbers
FROM code_blocks cb
JOIN code_block_incidents cbi ON cbi.code_block_id = cb.id
JOIN github_issues gi ON gi.id = cbi.issue_id
WHERE cb.repo_id = 1
  AND cb.incident_count >= 3
GROUP BY cb.id, cb.file_path, cb.block_name, cb.block_type, cb.incident_count
ORDER BY cb.incident_count DESC;

\echo ''

-- ============================================================================
-- Query 9: Timeline event coverage
-- ============================================================================
\echo '9. Timeline Event Coverage'
\echo '--------------------------------'
\echo 'Checking how many closed issues have corresponding incident links...'

WITH closed_issues AS (
    SELECT COUNT(*) AS total_closed
    FROM timeline_events te
    WHERE te.event_type = 'closed'
      AND te.commit_sha IS NOT NULL
      AND te.repo_id = 1
),
linked_issues AS (
    SELECT COUNT(DISTINCT cbi.issue_id) AS total_linked
    FROM code_block_incidents cbi
    WHERE cbi.repo_id = 1
      AND cbi.evidence_source = 'timeline_event'
)
SELECT
    ci.total_closed AS closed_issues_with_commits,
    li.total_linked AS issues_linked_to_blocks,
    CASE
        WHEN ci.total_closed > 0
        THEN ROUND((li.total_linked::NUMERIC / ci.total_closed * 100), 2)
        ELSE 0
    END AS coverage_percentage
FROM closed_issues ci, linked_issues li;

\echo ''

-- ============================================================================
-- Query 10: Validation summary
-- ============================================================================
\echo '10. Validation Summary'
\echo '--------------------------------'

SELECT
    'PASSED' AS status,
    'Temporal calculator implementation' AS test,
    'All blocks have incident_count set' AS condition
WHERE NOT EXISTS (
    SELECT 1 FROM code_blocks
    WHERE repo_id = 1 AND incident_count IS NULL
)
UNION ALL
SELECT
    CASE
        WHEN COUNT(*) = 0 THEN 'PASSED'
        ELSE 'FAILED'
    END AS status,
    'Count accuracy' AS test,
    'All counts match actual incident links' AS condition
FROM (
    SELECT cb.id
    FROM code_blocks cb
    LEFT JOIN code_block_incidents cbi ON cbi.code_block_id = cb.id
    WHERE cb.repo_id = 1
    GROUP BY cb.id, cb.incident_count
    HAVING cb.incident_count != COUNT(cbi.id)
) AS mismatched
UNION ALL
SELECT
    CASE
        WHEN COUNT(*) > 0 THEN 'PASSED'
        ELSE 'WARNING'
    END AS status,
    'Data presence' AS test,
    'At least one incident link exists' AS condition
FROM code_block_incidents
WHERE repo_id = 1;

\echo ''
\echo '================================'
\echo 'Verification Complete!'
\echo '================================'
