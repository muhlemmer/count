//go:build local

// building and running this file will just use up time in the CI environment.
// the build tag prevents mis-reporting of missed lines in code coverage.
package tester

import (
	"testing"
	"time"
)

func TestRunWithData(t *testing.T) {
	RunWithData(10*time.Minute, func(*Resources) int {
		return 0
	})
}
