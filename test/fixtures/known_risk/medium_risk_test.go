package fixtures

import (
	"context"
	"testing"
)

func TestMediumRiskFunction(t *testing.T) {
	ctx := context.Background()

	result, err := MediumRiskFunction(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
}
