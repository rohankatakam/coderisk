package fixtures

// LowRiskFunction has minimal dependencies, great test coverage
// Expected risk: LOW
// - Coupling: 0 dependencies (LOW)
// - Co-change: No co-change pattern (LOW)
// - Test ratio: ~0.75 (3 test functions vs 1 source function) (LOW)
func LowRiskFunction(x int) int {
	return x * 2
}
