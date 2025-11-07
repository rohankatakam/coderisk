package linking

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/linking/types"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// Orchestrator coordinates all phases of issue-PR linking
type Orchestrator struct {
	stagingDB *database.StagingClient
	llmClient *llm.Client
	repoID    int64
	days      int

	// Phase processors
	phase0 *Phase0Preprocessor
	phase1 *Phase1Extractor
	phase2A *Phase2PathA
	phase2B *Phase2PathB

	// Shared state
	doraMetrics   *types.DORAMetrics
	timelineLinks map[int][]types.TimelineLink
	explicitRefs  map[int][]types.ExplicitReference
}

// NewOrchestrator creates a new linking orchestrator
func NewOrchestrator(stagingDB *database.StagingClient, llmClient *llm.Client, repoID int64, days int) *Orchestrator {
	return &Orchestrator{
		stagingDB: stagingDB,
		llmClient: llmClient,
		repoID:    repoID,
		days:      days,
	}
}

// Run executes the complete issue-PR linking pipeline
func (o *Orchestrator) Run(ctx context.Context) error {
	startTime := time.Now()

	log.Printf("========================================")
	log.Printf("Issue-PR Linking Pipeline Starting")
	log.Printf("========================================")
	log.Printf("Repository ID: %d", o.repoID)
	log.Printf("Time window: %d days", o.days)
	log.Printf("")

	// Phase 0: Pre-processing
	if err := o.runPhase0(ctx); err != nil {
		return fmt.Errorf("Phase 0 failed: %w", err)
	}

	// Phase 1: Explicit Reference Extraction
	if err := o.runPhase1(ctx); err != nil {
		return fmt.Errorf("Phase 1 failed: %w", err)
	}

	// Phase 2: Issue Processing Loop
	if err := o.runPhase2(ctx); err != nil {
		return fmt.Errorf("Phase 2 failed: %w", err)
	}

	totalDuration := time.Since(startTime)
	log.Printf("")
	log.Printf("========================================")
	log.Printf("Issue-PR Linking Complete")
	log.Printf("========================================")
	log.Printf("Total time: %v", totalDuration)

	return nil
}

// runPhase0 executes Phase 0: Pre-processing
func (o *Orchestrator) runPhase0(ctx context.Context) error {
	log.Printf("[Phase 0] Pre-processing")
	log.Printf("─────────────────────────────────────")

	o.phase0 = NewPhase0Preprocessor(o.stagingDB)

	var err error
	o.doraMetrics, o.timelineLinks, err = o.phase0.RunPreprocessing(ctx, o.repoID, o.days)
	if err != nil {
		return err
	}

	// Store DORA metrics
	if err := o.stagingDB.StoreDORAMetrics(ctx, o.repoID, o.doraMetrics); err != nil {
		log.Printf("  ⚠️  Failed to store DORA metrics: %v", err)
	}

	log.Printf("")
	return nil
}

// runPhase1 executes Phase 1: Explicit Reference Extraction
func (o *Orchestrator) runPhase1(ctx context.Context) error {
	log.Printf("[Phase 1] Explicit Reference Extraction")
	log.Printf("─────────────────────────────────────")

	o.phase1 = NewPhase1Extractor(o.stagingDB, o.llmClient)

	var err error
	llmRefs, err := o.phase1.ExtractExplicitReferences(ctx, o.repoID, o.timelineLinks)
	if err != nil {
		return err
	}

	// Convert timeline links to explicit refs
	timelineRefs := ConvertTimelineLinksToExplicitRefs(o.timelineLinks)

	// Merge timeline and LLM refs
	o.explicitRefs = MergeExplicitReferences(timelineRefs, llmRefs)

	totalRefs := 0
	for _, refs := range o.explicitRefs {
		totalRefs += len(refs)
	}

	log.Printf("  ✓ Total explicit references: %d", totalRefs)
	log.Printf("")
	return nil
}

// runPhase2 executes Phase 2: Issue Processing Loop
func (o *Orchestrator) runPhase2(ctx context.Context) error {
	log.Printf("[Phase 2] Issue Processing Loop")
	log.Printf("─────────────────────────────────────")

	// Get all closed issues
	issues, err := o.stagingDB.GetAllClosedIssues(ctx, o.repoID)
	if err != nil {
		return fmt.Errorf("failed to get closed issues: %w", err)
	}

	log.Printf("Processing %d closed issues...", len(issues))
	log.Printf("")

	// Initialize phase processors
	o.phase2A = NewPhase2PathA(o.stagingDB, o.llmClient)
	o.phase2B = NewPhase2PathB(o.stagingDB, o.llmClient, o.doraMetrics)

	// Statistics
	stats := &ProcessingStats{}

	// Process each issue
	for i, issueData := range issues {
		log.Printf("Issue %d/%d: #%d - %s", i+1, len(issues), issueData.IssueNumber, truncateText(issueData.Title, 60))

		// Get full issue data with comments
		issue, err := o.stagingDB.GetIssueByNumber(ctx, o.repoID, issueData.IssueNumber)
		if err != nil {
			log.Printf("  ⚠️  Failed to get issue data: %v", err)
			stats.Failed++
			continue
		}

		// Check if issue has explicit references (Path A) or not (Path B)
		refs, hasExplicitRefs := o.explicitRefs[issue.IssueNumber]

		if hasExplicitRefs {
			// Path A: Process each explicit reference
			log.Printf("  Path A: %d explicit reference(s)", len(refs))
			stats.PathA++

			for _, ref := range refs {
				// Get PR data using PRNumber from the explicit reference
				pr, err := o.stagingDB.GetPRByNumber(ctx, o.repoID, ref.PRNumber)
				if err != nil {
					log.Printf("    ⚠️  Failed to get PR #%d: %v", ref.PRNumber, err)
					continue
				}

				// Process explicit link
				link, err := o.phase2A.ProcessExplicitLink(ctx, o.repoID, issue, pr, ref)
				if err != nil {
					log.Printf("    ⚠️  Failed to process link: %v", err)
					continue
				}

				// Store link
				if err := o.stagingDB.StoreLinkOutput(ctx, o.repoID, *link); err != nil {
					log.Printf("    ⚠️  Failed to store link: %v", err)
					continue
				}

				log.Printf("    ✓ Link to PR #%d: confidence=%.2f, quality=%s", pr.PRNumber, link.FinalConfidence, link.LinkQuality)
				stats.LinksCreated++
			}

		} else {
			// Path B: Deep link finder
			log.Printf("  Path B: No explicit references, running deep finder...")
			stats.PathB++

			links, noLink, err := o.phase2B.ProcessDeepLink(ctx, o.repoID, issue)
			if err != nil {
				log.Printf("  ⚠️  Deep finder failed: %v", err)
				stats.Failed++
				continue
			}

			if len(links) > 0 {
				// Store deep links
				for _, link := range links {
					if err := o.stagingDB.StoreLinkOutput(ctx, o.repoID, link); err != nil {
						log.Printf("    ⚠️  Failed to store link: %v", err)
						continue
					}

					log.Printf("    ✓ Deep link to PR #%d: confidence=%.2f, quality=%s", link.PRNumber, link.FinalConfidence, link.LinkQuality)
					stats.LinksCreated++
				}
			} else if noLink != nil {
				// Store no-link record
				if err := o.stagingDB.StoreNoLinkOutput(ctx, o.repoID, *noLink); err != nil {
					log.Printf("    ⚠️  Failed to store no-link: %v", err)
					continue
				}

				log.Printf("    ✓ No link: reason=%s", noLink.NoLinksReason)
				stats.NoLinks++
			}
		}

		log.Printf("")
	}

	// Print statistics
	log.Printf("Processing Statistics:")
	log.Printf("  Total issues: %d", len(issues))
	log.Printf("  Path A (explicit): %d", stats.PathA)
	log.Printf("  Path B (deep finder): %d", stats.PathB)
	log.Printf("  Links created: %d", stats.LinksCreated)
	log.Printf("  No links: %d", stats.NoLinks)
	log.Printf("  Failed: %d", stats.Failed)

	return nil
}

// ProcessingStats tracks processing statistics
type ProcessingStats struct {
	PathA        int
	PathB        int
	LinksCreated int
	NoLinks      int
	Failed       int
}
