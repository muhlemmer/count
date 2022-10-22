package tester

import (
	"os"
	"testing"
	"time"
)

func TestRunWithData(t *testing.T) {
	if ci := os.Getenv("CI"); ci == "true" {
		t.Skipf("skipping tester.RunWithData() because CI = %s", ci)
	}
	RunWithData(10*time.Minute, func(*Resources) int {
		return 0
	})
}
