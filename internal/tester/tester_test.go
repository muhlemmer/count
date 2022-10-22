package tester

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	Run(10*time.Minute, func(*Resources) int {
		return 0
	})
}
