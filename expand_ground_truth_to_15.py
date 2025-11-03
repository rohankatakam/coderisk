#!/usr/bin/env python3
"""Expand ground truth from 11 to 15 test cases"""

import json

# Read current ground truth
with open("test_data/omnara_ground_truth_expanded.json", "r") as f:
    data = json.load(f)

# New test cases to add
new_cases = [
    {
      "issue_number": 111,
      "title": "[BUG] CONTRIBUTING guide link in readme broken",
      "issue_url": "https://github.com/omnara-ai/omnara/issues/111",
      "expected_links": {
        "fixed_by_commits": [],
        "associated_prs": [112],
        "associated_issues": []
      },
      "linking_patterns": ["explicit"],
      "primary_evidence": {
        "pr_body_contains": "Address #111",
        "reference_type": "address",
        "explicit": True,
        "issue_closed_at": "2025-08-14T00:00:00Z",
        "pr_merged_at": "2025-08-14T00:00:00Z"
      },
      "link_quality": "high",
      "difficulty": "easy",
      "notes": "PR #112 explicitly states 'Address #111'. Added CONTRIBUTING.md file to fix broken README link.",
      "expected_confidence": 0.75,
      "should_detect": True,
      "github_verification": {
        "issue_state": "closed",
        "pr_state": "merged",
        "verified_manually": True,
        "verified_date": "2025-11-02"
      }
    },
    {
      "issue_number": 62,
      "title": "[FEATURE] Other modes of Claude Code",
      "issue_url": "https://github.com/omnara-ai/omnara/issues/62",
      "expected_links": {
        "fixed_by_commits": [],
        "associated_prs": [133],
        "associated_issues": []
      },
      "linking_patterns": ["explicit"],
      "primary_evidence": {
        "pr_mentioned_in": "ksarangmath mentioned this pull request Aug 17, 2025 [FEATURE] Other modes of Claude Code #62",
        "reference_type": "mentioned",
        "explicit": True,
        "issue_closed_at": "2025-08-17T00:00:00Z",
        "pr_merged_at": "2025-08-17T00:00:00Z"
      },
      "link_quality": "high",
      "difficulty": "easy",
      "notes": "PR #133 mentioned Issue #62. Implemented --permission-mode and --dangerously-skip-permissions flags for Claude operational modes.",
      "expected_confidence": 0.75,
      "should_detect": True,
      "github_verification": {
        "issue_state": "closed",
        "pr_state": "merged",
        "verified_manually": True,
        "verified_date": "2025-11-02"
      }
    },
    {
      "issue_number": 164,
      "title": "[FEATURE] codex cli integration",
      "issue_url": "https://github.com/omnara-ai/omnara/issues/164",
      "expected_links": {
        "fixed_by_commits": [],
        "associated_prs": [],
        "associated_issues": []
      },
      "linking_patterns": ["temporal"],
      "primary_evidence": {
        "temporal_correlation": True,
        "note": "Database shows 6 PRs via temporal correlation, but no explicit references found on GitHub. Complex feature implemented incrementally.",
        "issue_closed_at": "2025-09-03T00:00:00Z"
      },
      "link_quality": "medium",
      "difficulty": "hard",
      "notes": "Complex multi-PR feature. Database shows 6 temporal PR matches. Feature request for Codex CLI integration implemented incrementally. Test case for multi-PR scenarios.",
      "expected_confidence": 0.65,
      "should_detect": True,
      "github_verification": {
        "issue_state": "closed",
        "pr_state": None,
        "verified_manually": True,
        "verified_date": "2025-11-02",
        "note": "No explicit PRs mentioned on issue page, relying on temporal correlation"
      }
    },
    {
      "issue_number": 206,
      "title": "[Proposal] Rename Omnara",
      "issue_url": "https://github.com/omnara-ai/omnara/issues/206",
      "expected_links": {
        "fixed_by_commits": [],
        "associated_prs": [],
        "associated_issues": []
      },
      "linking_patterns": ["none"],
      "primary_evidence": {
        "issue_state": "closed",
        "close_reason": "completed",
        "proposal_rejected": True,
        "note": "Proposal rejected due to existing .com domain ownership"
      },
      "link_quality": "n/a",
      "difficulty": "easy",
      "notes": "True negative - proposal rejected. No PRs created. Closed by original poster after maintainer response. Tests rejection of non-actionable proposals.",
      "expected_confidence": None,
      "should_detect": False,
      "github_verification": {
        "issue_state": "closed",
        "pr_state": None,
        "verified_manually": True,
        "verified_date": "2025-11-02"
      }
    }
]

# Add new cases
data["test_cases"].extend(new_cases)

# Update metadata
data["total_cases"] = 15
data["pattern_distribution"] = {
    "explicit": 7,  # was 5, +2 (#111, #62)
    "temporal": 4,  # was 3, +1 (#164)
    "true_negative": 3,  # was 2, +1 (#206)
    "internal_fix": 1  # unchanged
}

data["validation_metrics"] = {
    "expected_true_positives": 11,  # was 8, +3
    "expected_true_negatives": 3,   # was 2, +1
    "expected_false_negatives": 1,  # unchanged (Issue #188)
    "expected_false_positives": 0,  # unchanged
    "target_precision": 1.0,
    "target_recall": 0.92,  # 11/12 = 91.67%
    "target_f1": 0.96
}

data["notes"] = "Expanded to 15 cases for stronger statistical confidence. Added 2 explicit references (#111, #62), 1 complex temporal case (#164), and 1 true negative (#206). Validates both simple and complex linking scenarios."

# Write updated ground truth
with open("test_data/omnara_ground_truth_15cases.json", "w") as f:
    json.dump(data, f, indent=2)

print("✅ Created omnara_ground_truth_15cases.json with 15 test cases")
print(f"   - Explicit: {data['pattern_distribution']['explicit']}")
print(f"   - Temporal: {data['pattern_distribution']['temporal']}")
print(f"   - True Negatives: {data['pattern_distribution']['true_negative']}")
print(f"   - Expected Recall: {data['validation_metrics']['target_recall']:.1%}")
