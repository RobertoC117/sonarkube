package targets

import (
	"strings"
	"testing"
)

func TestGetParsedTargets(t *testing.T) {
	targets := []string{"taregt1", "target2"}
	targetsReader := strings.NewReader(strings.Join(targets, "\n"))

	parsedTargets, err := ParseTargets(targetsReader)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(parsedTargets) != len(targets) {
		t.Errorf("Expected %v targets but got %v targets", len(targets), len(parsedTargets))
	}
}
