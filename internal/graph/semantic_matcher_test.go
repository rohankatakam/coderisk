package graph

import (
	"testing"
)

func TestSemanticMatcher_CalculateSimilarity(t *testing.T) {
	sm := NewSemanticMatcher()

	tests := []struct {
		name     string
		text1    string
		text2    string
		wantMin  float64
		wantMax  float64
		desc     string
	}{
		{
			name:    "identical texts",
			text1:   "fix mobile interface bug",
			text2:   "fix mobile interface bug",
			wantMin: 0.95,
			wantMax: 1.0,
			desc:    "identical texts should have ~100% similarity",
		},
		{
			name:    "high similarity",
			text1:   "mobile interface sync issues",
			text2:   "handle mobile interface permissions",
			wantMin: 0.40,
			wantMax: 0.80,
			desc:    "overlapping keywords (mobile, interface) should have high similarity",
		},
		{
			name:    "zero similarity",
			text1:   "prompts from subagents aren't shown",
			text2:   "codex version update bump",
			wantMin: 0.0,
			wantMax: 0.15,
			desc:    "completely different topics should have low similarity",
		},
		{
			name:    "stop words ignored",
			text1:   "the bug is in the code",
			text2:   "bug in code",
			wantMin: 0.90,
			wantMax: 1.0,
			desc:    "stop words should be filtered out",
		},
		{
			name:    "case insensitive",
			text1:   "Fix Mobile Bug",
			text2:   "fix mobile bug",
			wantMin: 0.95,
			wantMax: 1.0,
			desc:    "matching should be case insensitive",
		},
		{
			name:    "stemming works",
			text1:   "fixing bugs",
			text2:   "fixed bug",
			wantMin: 0.60,
			wantMax: 1.0,
			desc:    "stemming should match 'fixing' with 'fixed'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sm.CalculateSimilarity(tt.text1, tt.text2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateSimilarity() = %v, want between %v and %v\n  text1: %q\n  text2: %q\n  desc: %s",
					got, tt.wantMin, tt.wantMax, tt.text1, tt.text2, tt.desc)
			}
		})
	}
}

func TestSemanticMatcher_ValidateTemporalMatch(t *testing.T) {
	sm := NewSemanticMatcher()

	tests := []struct {
		name          string
		issueText     string
		prText        string
		wantKeep      bool
		wantBoost     bool
		desc          string
	}{
		{
			name:      "high similarity - should boost",
			issueText: "mobile interface sync issues with claude code subagents",
			prText:    "handle mobile interface permission requests for subagents",
			wantKeep:  true,
			wantBoost: true,
			desc:      "keywords: mobile, interface, subagents overlap >70%",
		},
		{
			name:      "medium similarity - accept without boost",
			issueText: "ctrl z causes application to freeze",
			prText:    "handle ctrl z keyboard shortcut",
			wantKeep:  true,
			wantBoost: false,
			desc:      "keywords: ctrl, z overlap ~50%",
		},
		{
			name:      "low similarity - reject",
			issueText: "prompts from subagents aren't shown",
			prText:    "codex version 0.36.0 update",
			wantKeep:  false,
			wantBoost: false,
			desc:      "no keyword overlap - reject temporal match",
		},
		{
			name:      "zero similarity - reject",
			issueText: "authentication token expired",
			prText:    "add placeholder for uploads folder",
			wantKeep:  false,
			wantBoost: false,
			desc:      "completely unrelated - reject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeep, gotBoost := sm.ValidateTemporalMatch(tt.issueText, tt.prText)

			if gotKeep != tt.wantKeep {
				t.Errorf("ValidateTemporalMatch() keep = %v, want %v\n  issue: %q\n  pr: %q\n  desc: %s",
					gotKeep, tt.wantKeep, tt.issueText, tt.prText, tt.desc)
			}

			if tt.wantBoost {
				if gotBoost <= 0 {
					t.Errorf("ValidateTemporalMatch() boost = %v, want >0 for high similarity\n  desc: %s",
						gotBoost, tt.desc)
				}
			} else {
				if gotBoost != 0 && tt.wantKeep {
					t.Errorf("ValidateTemporalMatch() boost = %v, want 0 for medium similarity\n  desc: %s",
						gotBoost, tt.desc)
				}
			}
		})
	}
}

func TestSemanticMatcher_RealWorldCases(t *testing.T) {
	sm := NewSemanticMatcher()

	// Real test cases from omnara_ground_truth.json
	tests := []struct {
		name        string
		issueTitle  string
		issueBody   string
		prTitle     string
		prBody      string
		shouldMatch bool
		desc        string
	}{
		{
			name:       "Issue #187 -> PR #218 (should match)",
			issueTitle: "[BUG] Mobile interface sync issues with Claude Code subagents",
			issueBody:  "The mobile interface has synchronization problems when using Claude Code with subagents",
			prTitle:    "handle subtask permission requests",
			prBody:     "Fix mobile interface sync with Claude Code",
			shouldMatch: true,
			desc:       "Keywords: mobile, interface, sync, claude, code - high overlap",
		},
		{
			name:       "Issue #189 -> PR #203 (should match)",
			issueTitle: "[BUG] Ctrl + Z = Dead",
			issueBody:  "Pressing Ctrl+Z causes the application to become unresponsive",
			prTitle:    "handle ctrl z",
			prBody:     "Fix keyboard shortcut handling for undo operation",
			shouldMatch: true,
			desc:       "Keywords: ctrl, z - strong overlap",
		},
		{
			name:       "Issue #219 -> PR #229 (should NOT match)",
			issueTitle: "[BUG] prompts from subagents aren't shown",
			issueBody:  "Omnara doesn't show some user prompts from Claude Code which ask the user for permission",
			prTitle:    "Codex 0.36.0",
			prBody:     "update codex version bump",
			shouldMatch: false,
			desc:       "No keyword overlap - unrelated",
		},
		{
			name:       "Issue #219 -> PR #230 (should NOT match)",
			issueTitle: "[BUG] prompts from subagents aren't shown",
			issueBody:  "Omnara doesn't show some user prompts from Claude Code",
			prTitle:    "feat: Open source the Omnara Frontend",
			prBody:     "Add placeholder for the uploads folder",
			shouldMatch: false,
			desc:       "No keyword overlap - unrelated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := sm.CalculateIssueToPRSimilarity(
				tt.issueTitle,
				tt.issueBody,
				tt.prTitle,
				tt.prBody,
			)

			shouldKeep, _ := sm.ValidateTemporalMatch(
				tt.issueTitle+" "+tt.issueBody,
				tt.prTitle+" "+tt.prBody,
			)

			if tt.shouldMatch && !shouldKeep {
				t.Errorf("Expected match but got rejection\n  similarity: %v\n  desc: %s",
					similarity, tt.desc)
			}

			if !tt.shouldMatch && shouldKeep {
				t.Errorf("Expected rejection but got match\n  similarity: %v\n  desc: %s",
					similarity, tt.desc)
			}

			t.Logf("✓ %s: similarity=%.2f, keep=%v", tt.name, similarity, shouldKeep)
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	sm := NewSemanticMatcher()

	tests := []struct {
		name         string
		text         string
		wantKeywords []string
		desc         string
	}{
		{
			name:         "basic extraction",
			text:         "Fix mobile interface bug",
			wantKeywords: []string{"fix", "mobile", "interface", "bug"},
			desc:         "should extract all meaningful words",
		},
		{
			name:         "stop words filtered",
			text:         "The bug is in the code",
			wantKeywords: []string{"bug", "code"},
			desc:         "should remove stop words: the, is, in",
		},
		{
			name:         "markdown stripped",
			text:         "**Fix** _bug_ in `code`",
			wantKeywords: []string{"fix", "bug", "code"},
			desc:         "should remove markdown syntax",
		},
		{
			name:         "hyphenated terms",
			text:         "user-agent claude-code",
			wantKeywords: []string{"user-agent", "claude-code"},
			desc:         "should preserve hyphenated terms",
		},
		{
			name:         "version numbers kept",
			text:         "Update to version 0.36.0",
			wantKeywords: []string{"update", "version", "0.36.0"},
			desc:         "should keep version numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := sm.extractKeywords(tt.text)

			for _, want := range tt.wantKeywords {
				if !keywords[want] && !keywords[simpleStem(want)] {
					t.Errorf("extractKeywords() missing keyword %q\n  text: %q\n  desc: %s",
						want, tt.text, tt.desc)
				}
			}
		})
	}
}
