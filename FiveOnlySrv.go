package main

/*
 * Implements a simple webserver that handles up to five
 * concurrent requests that each take a random amount
 * of time to complete. Once a sixth concurrent request
 * comes in, it dies, calling log.fatal
 *
 */

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// randomSleep() returns a (pseudo) random number of milliseconds, up to maxSleep
// and a function that sleeps for just that amount of time. 
func randomSleep(maxSleep int) (napTime time.Duration, napFunc func()) {
	napTime = time.Duration(rand.Intn(maxSleep)) * time.Millisecond
	napFunc = func() {
		time.Sleep(napTime)
	}
	return
}

// As many requests as serverRequest() will handle WITHOUT
// tossing back error 503 (Service Unavailable)
const maxConcurrentRequest = 5

// This *SHOULD* be the number of concurrentRequests at any time, but
// because of concurrency it gets munged up. Wish there were a way to
// examine the items on a channel without pulling'em off. :(
var concurrentRequests int = 0

// A running count of requests served.
var totalRequestsProcessed int = 0

var inProgress = make(map[int]bool) // key: the serial # of jobs in progress/value: true if in progress
var semaphore = make(chan bool, 1)

// dieOvercapacity() prints out the jobs still running (there should be five, always)
// and then calls log.Fatal to print the reason in why
func dieOvercapacity (why string) {
	semaphore <- true	// Let no jobs start, stop, be added
	log.Printf("Reached capacity\nJobs still in progress are:\n")
	for jobnum, _ := range inProgress {
		log.Printf("Request #%04d\n", jobnum)
	}
	log.Fatalf ("%s\n", why)
}
// func serveRequest() handles all requests since it handles "/" and there are no
// other handlers installed. It does very little: it increments concurentRequests
// and if concurrentRequests < maxConcurrentRequest, shuts down the process with
// a log.fatal() call.
func serveRequest(w http.ResponseWriter, r *http.Request) {
	concurrentRequests++
	if concurrentRequests > maxConcurrentRequest {
		dieOvercapacity(fmt.Sprintf("More than %02d concurrentRequests: dying!\n", maxConcurrentRequest))
	}

	semaphore <- true
	totalRequestsProcessed++
	reqSerialNumber := totalRequestsProcessed
	inProgress[reqSerialNumber] = true
	<-semaphore
	napTime, napFunc := randomSleep(999)
	fmt.Fprintf(w, "INFO:\n====\n")
	fmt.Fprintf(w, "Request #%04d for '%+v'. Currently %02d requests\n", reqSerialNumber,
		r.URL, concurrentRequests)
	fmt.Fprintf(w, "#%04d: Milliseconds until completion:\t%04d.\n", reqSerialNumber, napTime/time.Millisecond)
	log.Printf("INFO:\n====\n")
	log.Printf("Request #%04d for '%+v'. Currently %02d requests\n", reqSerialNumber,
		r.URL, concurrentRequests)
	log.Printf("#%04d: Milliseconds until completion:\t%04d.\n", reqSerialNumber, napTime/time.Millisecond)
	napFunc()
	semaphore <- true
	concurrentRequests--
	delete (inProgress, reqSerialNumber)
	<-semaphore
	fmt.Fprintf(w, "Request #%04d: Done. Currently %02d requests\n\n",
		reqSerialNumber, concurrentRequests)
	log.Printf("Request #%04d: Done. Currently %02d requests\n\n",
		reqSerialNumber, concurrentRequests)
	return
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
