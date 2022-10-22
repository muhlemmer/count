package main

import (
	"os"
	"time"

	"github.com/muhlemmer/count/internal/tester"
)

func main() {
	os.Exit(tester.Run(5*time.Minute, func(r *tester.Resources) int {
		return 0
	}))
}
