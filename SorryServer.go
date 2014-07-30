package main

/*
 * Implements a webserver that slows down and speeds up
 * with the number of concurrent requests it has to handle.
 * While this is natural behavior, this exaggerates it by
 * counting concurrent requests and using a random sleep function
 * with greater sleep times the greater the number of concurrent
 * requests. When it reaches five times the highest threshold,
 * the server just dies (see the comment to dieOvercapacity())
 *
 */

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// randomSleep() returns a pseudo-random number of milliseconds, up to maxSleep
// and a function that sleeps for just that amount of time. 
func randomSleep(maxSleep int) (napTime time.Duration, napFunc func()) {
	napTime = time.Duration(rand.Intn(maxSleep)) * time.Millisecond
	napFunc = func() {
		time.Sleep(napTime)
	}
	return
}

// The thresholds for levels of service degredation:
const concurrentReqsLvl0 = 5	// Zippy, randomSleep(50)
const concurrentReqsLvl1 = 10	// OK, randomSleep(100)
const concurrentReqsLvl2 = 25	// Getting pokey, randomSleep(250)
const concurrentReqsLvl3 = 50	// Bad, randomSleep(500)
const concurrentReqsLvl4 = 100	// Awful, randomSleep(750)

// The threshold for total server capacity:
const maxConcurrentReqs = concurrentReqsLvl4 * 5

// The number of concurrent requests at any time.
// Incrementing/decrementing protected by a single element
// channel called semaphore, below.
var concurrentRequests int = 0

// A running count of requests served.
var totalRequestsProcessed int = 0

var semaphore = make(chan bool, 1)

// dieOvercapacity() prints out the number of  jobs still running 
// and then calls log.Fatal to print the reason in why
func dieOvercapacity (why string) {
	semaphore <- true	// Let no jobs start, stop, be added
	log.Printf("Reached capacity\nStill %d jobs in progress.\n", concurrentRequests)
	log.Fatalf ("%s\n", why)
}
// func serveRequest() handles all requests since it handles "/" and there are no
// other handlers installed. It does very little: it increments concurentRequests
// and if concurrentRequests < maxConcurrentReqs, shuts down the process with
// a log.fatal() call.
var startSemaphore, releaseSemaphore, priorStart time.Time

func serveRequest(w http.ResponseWriter, r *http.Request) {
	concurrentRequests++
	if concurrentRequests > maxConcurrentReqs {
		dieOvercapacity(fmt.Sprintf("More than %04d concurrentRequests: dying!\n", maxConcurrentReqs))
	}


	if ! startSemaphore.IsZero()  {
		priorStart = startSemaphore
	}
	startSemaphore = time.Now()
	log.Printf("\tApplying entry semaphore at %s\n", startSemaphore.String())
	semaphore <- true
	totalRequestsProcessed++
	reqSerialNumber := totalRequestsProcessed
	<-semaphore
	releaseSemaphore = time.Now()
	entrySemaphoreHeldFor := releaseSemaphore.Sub(startSemaphore)
	log.Printf("\tReleasing entry semaphore: duration was %010d ns\n", entrySemaphoreHeldFor)
	if ! priorStart.IsZero() {
		log.Printf("\tPrevious request arrived at %s\n", priorStart.String())
		log.Printf("\tCurrent request arrived at %s\n", startSemaphore.String())
		log.Printf("\tElapsed time: %04d ms, %05d ns\n", 
			time.Duration(startSemaphore.Nanosecond() - priorStart.Nanosecond())/time.Millisecond,
			time.Duration(startSemaphore.Nanosecond() - priorStart.Nanosecond()) % 10000)
	}
		
	var maxNap int
	switch {
		case concurrentRequests < concurrentReqsLvl0:
			maxNap = 50
		case concurrentRequests < concurrentReqsLvl1:
			maxNap = 100
		case concurrentRequests < concurrentReqsLvl2:
			maxNap = 250
		case concurrentRequests < concurrentReqsLvl3:
			maxNap = 500
		case concurrentRequests < concurrentReqsLvl4:
			maxNap = 750
	}
	napTime, napFunc := randomSleep(maxNap)
	fmt.Fprintf(w, "INFO:\n====\n")
	fmt.Fprintf(w, "Request #%04d for '%+v'. Currently %02d requests\n", reqSerialNumber,
		r.URL, concurrentRequests)
	fmt.Fprintf(w, "#%04d: Milliseconds until completion:\t%04d.\n", reqSerialNumber, napTime/time.Millisecond)
	log.Printf("Request #%04d for '%+v'. Currently %04d requests: Sleeping for %04d ms \n", reqSerialNumber,
		r.URL, concurrentRequests, napTime/time.Millisecond)
	napFunc()
	semaphore <- true
	concurrentRequests--
	<-semaphore
	fmt.Fprintf(w, "Request #%04d: Done. Currently %02d requests\n\n",
		reqSerialNumber, concurrentRequests)
	return
}

// Stops the server. I want to prevent the port from getting stuck open,
// which seems to happen after several times starting and then killing
// the server
func haltsrv (w http.ResponseWriter, r *http.Request) {
	log.Fatal("Stopping server\n")
}

func main() {

	// Initialize the random number generator with something that
	// should be nicely messy.
	rand.Seed(int64(time.Duration(time.Now().Nanosecond()) / time.Millisecond))

	// with no other handlers installed, this will catch all requests
	http.HandleFunc("/", serveRequest)

	// a convenience handler to shut down the server: sometimes, after numerous
	// startup/shutdowns the port gets "stuck" and I have to reboot. I hope
	// this will stop that
	http.HandleFunc("/HALT", haltsrv)


	// Run the server on port 8181. Prints the reason
	// why the server stops (if it's not by a ^C on
	// the controlling terminal
	whyCrashed := http.ListenAndServe(":8181", nil)
	fmt.Printf("ListenAndServe died: %s\n", whyCrashed)
}
