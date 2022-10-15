package timer

import (
	"errors"
	"math/rand"
	"time"
)

var (
	errNegativeValue = errors.New("timer: nagative min value")
	errMinMax        = errors.New("timer: min not lower lower max")
)

func randomDuration(min, max time.Duration) time.Duration {
	if min < 0 {
		panic(errNegativeValue)
	}
	if min >= max {
		panic(errMinMax)
	}

	d := time.Duration(rand.Int63n(int64(max - min)))
	return d + min
}

func RandomTimer(min, max time.Duration) <-chan time.Time {
	return time.After(randomDuration(min, max))
}
