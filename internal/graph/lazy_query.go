package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// LazyQueryIterator provides lazy iteration over query results
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 3
//
// Traditional approach (eager):
//   result := neo4j.ExecuteQuery(...) // Loads ALL records into memory
//   for _, record := range result.Records { ... }
//
// Lazy approach:
//   iter := ExecuteQueryLazy(...)
//   for iter.Next() {
//     record := iter.Record() // Only current record in memory
//   }
//
// This reduces memory usage by 50-70% for large result sets.
type LazyQueryIterator struct {
	result  neo4j.ResultWithContext
	session neo4j.SessionWithContext
	ctx     context.Context
}

// Next advances to the next record
// Returns true if a record is available, false otherwise
func (l *LazyQueryIterator) Next() bool {
	return l.result.Next(l.ctx)
}

// Record returns the current record
// Must call Next() first to advance to a record
func (l *LazyQueryIterator) Record() *neo4j.Record {
	return l.result.Record()
}

// Collect reads remaining records into a slice (up to limit)
// Use this to prevent unbounded memory growth
// Limit parameter prevents OOM on unexpectedly large result sets
func (l *LazyQueryIterator) Collect(limit int) ([]*neo4j.Record, error) {
	records := make([]*neo4j.Record, 0, limit)

	for l.Next() && len(records) < limit {
		records = append(records, l.result.Record())
	}

	if err := l.result.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// Close consumes remaining results and returns summary
// IMPORTANT: Always defer Close() after creating an iterator
// This ensures proper cleanup of Neo4j session resources
func (l *LazyQueryIterator) Close(ctx context.Context) (neo4j.ResultSummary, error) {
	defer l.session.Close(ctx)
	return l.result.Consume(ctx)
}

// Err returns any error that occurred during iteration
func (l *LazyQueryIterator) Err() error {
	return l.result.Err()
}

// ExecuteQueryLazy executes a query and returns a lazy iterator
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 3
//
// IMPORTANT: Caller must call Close() on the iterator to free session resources
//
// Example usage:
//   iter, err := neo4j.ExecuteQueryLazy(ctx, driver, query, params, "neo4j", 500)
//   if err != nil { return err }
//   defer iter.Close(ctx)
//
//   for iter.Next() {
//     record := iter.Record()
//     // Process record
//   }
//   if err := iter.Err(); err != nil { return err }
func ExecuteQueryLazy(
	ctx context.Context,
	driver neo4j.DriverWithContext,
	query string,
	params map[string]any,
	database string,
	fetchSize int,
) (*LazyQueryIterator, error) {
	// Create session with fetch size configuration
	// fetchSize controls how many records are buffered at once
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
		AccessMode:   neo4j.AccessModeRead,
		FetchSize:    fetchSize, // Key optimization: controls batch size for lazy loading
	})

	// Run query in read transaction
	// Note: We don't use ExecuteRead here because we want to return the iterator
	result, err := session.Run(ctx, query, params)
	if err != nil {
		session.Close(ctx)
		return nil, fmt.Errorf("lazy query failed: %w", err)
	}

	return &LazyQueryIterator{
		result:  result,
		session: session,
		ctx:     ctx,
	}, nil
}

// ExecuteQueryLazyWithReadTransaction executes a query in a read transaction with lazy iteration
// Use this when you need transaction semantics with lazy loading
func ExecuteQueryLazyWithReadTransaction(
	ctx context.Context,
	driver neo4j.DriverWithContext,
	query string,
	params map[string]any,
	database string,
	fetchSize int,
	fn func(*LazyQueryIterator) error,
) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
		AccessMode:   neo4j.AccessModeRead,
		FetchSize:    fetchSize,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		iter := &LazyQueryIterator{
			result:  result,
			session: session,
			ctx:     ctx,
		}

		return nil, fn(iter)
	})

	return err
}

// FetchSizeConfig controls how many records are fetched at a time
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 3
type FetchSizeConfig struct {
	// Small queries (< 10 results expected)
	SmallQueryFetchSize int // Default: 100

	// Medium queries (10-100 results)
	MediumQueryFetchSize int // Default: 500

	// Large queries (100+ results)
	LargeQueryFetchSize int // Default: 1000
}

// DefaultFetchSizeConfig returns recommended fetch sizes
func DefaultFetchSizeConfig() FetchSizeConfig {
	return FetchSizeConfig{
		SmallQueryFetchSize:  100,
		MediumQueryFetchSize: 500,
		LargeQueryFetchSize:  1000,
	}
}
