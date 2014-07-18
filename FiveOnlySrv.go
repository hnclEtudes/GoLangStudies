package main

/*
 * Implements a simple webserver that handles up to five
 * concurrent requests that each take a random amount
 * of time to complete; otherwise returns 503, Service
 * Unavailable, to the client.
 *
 */

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// Returns a time.Duration between 0 and 999. The random number
// generator is reseeded in main with time.Now().Nanoseconds(),
// which should be a big messy weird number
func randomSleep() (napTime time.Duration) {
	napTime = time.Duration(rand.Intn(999)) * time.Millisecond
	return
}

var concurrentRequests int = 0

// func serveRequest() handles all requests since it handles "/" and
// there are no other handlers installed. It does very little:
// checks the concurrentRequests counter to see if more than
// maxRequests (now 5) are being handled. If so, it returns
// Status Service Unavailable error (503). If not, it gets
// a random amount of time to sleep, outputs that info,
// calls time.Sleep() for that duration, then says "Done" and 
// decrements the concurrentRequests counter
func serveRequest(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "concurrentRequests: %d\n", concurrentRequests)
	if concurrentRequests >= 5 {
		w.WriteHeader(http.StatusServiceUnavailable)
		http.Error(w, "Maximum Requests reached", http.StatusServiceUnavailable)
		concurrentRequests--
		return
	}

	concurrentRequests++
	serveDuration := randomSleep()
	fmt.Fprintf(w, "Currently there are: %d requests\n", concurrentRequests)
	fmt.Fprintf(w, "Milliseconds until completion:\t%04d.\n", serveDuration/time.Millisecond)
	fmt.Printf("Currently there are: %d requests\n", concurrentRequests)
	fmt.Printf("Milliseconds until completion:\t%04d.\n", serveDuration/time.Millisecond)
	time.Sleep(serveDuration)
	fmt.Fprintf(w, "Done\n")

	concurrentRequests--
}

func main() {

	// Initialize the random number generator with something that
	// should be nicely messy.
	rand.Seed(int64(time.Duration(time.Now().Nanosecond()) / time.Millisecond))

	// with no other handlers installed, this will catch all requests
	http.HandleFunc("/", serveRequest)

	// Run the server on port 8181. Prints the reason
	// why the server stops (if it's not by a ^C on
	// the controlling terminal
	whyCrashed := http.ListenAndServe(":8181", nil)
	fmt.Printf("ListenAndServe died: %s\n", whyCrashed)
}
