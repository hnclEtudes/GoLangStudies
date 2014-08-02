package waiting

// As of 2014-08-05, a package that contains a function that given a maximum
// number of milliseconds, returns a function that will sleep a random
// number of milliseconds and the number of milliseconds that function will
// sleep.  Also, but not essential, is a function that initializes the 
// random number generator using a pretty pseudo-random number, namely the
// millisecond part of the current time.

import (
	"time"
	"math/rand"
)

// func Init() initializes the random number generator with something that
// should be nicely messy, the current millisecond part of the current time.
func Init() {
	rand.Seed(int64(time.Duration(time.Now().Nanosecond()) / time.Millisecond))
}

// RandomSleep() returns a pseudo-random number of milliseconds, up to maxSleep
// and a function that sleeps for just that amount of time. 
func RandomSleep(maxSleep int) (napTime time.Duration, napFunc func()) {
	napTime = time.Duration(rand.Intn(maxSleep)) * time.Millisecond
	napFunc = func() {
		time.Sleep(napTime)
	}
	return
}
