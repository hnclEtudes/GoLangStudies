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
	"log"
	"math/rand"
	"net/http"
	"time"
)

// randomSleep() returns a (pseudo) random number of milliseconds, up to 999
// and a function that sleeps for just that amount of time. It's a handy thing
// and I should make a package of them.
func randomSleep() (napTime time.Duration, napFunc func ()) {
	napTime = time.Duration(rand.Intn(999)) * time.Millisecond
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

// Contains the values of totalRequestsProcessed for those lucky enough to make it in.
// Note that it will hold one more than maxConcurrentRequest; it's just to elicit
// the lovely error 503 every so often
var requestQueue = make(chan int, maxConcurrentRequest + 1)	// Maximum requests to let in

// func serveRequest() handles all requests since it handles "/" and there are no 
// other handlers installed. It does very little: it increments the relevant 
// counters, tosses totalRequestProcess into the 'requestQueue' to ensure that no more 
// than five maxConcurrentRequests + 1 (+1 to deliberately trip the error 503 code 
// every so often) make it in. Those requests lucky enough to be the first through 
// fifth in the channel receive a bunch of messages about the state of the server,
// how long it will sleep before completing the request, and after the nap a "Done"
// with the total number of requests the server has fielded since startup.
func serveRequest(w http.ResponseWriter, r *http.Request) {
	requestQueue <- totalRequestsProcessed
	reqSerialNumber := totalRequestsProcessed
	totalRequestsProcessed++; 
	concurrentReqsAtStart := concurrentRequests
	concurrentRequests++
	log.Printf("Starting server request %03d, with concurrentRequests: %d\n", 
		totalRequestsProcessed, concurrentRequests)

	
	/* This shouldn't happen if the requestQueue isn't bigger 
	 * than maxConcurrentRequests, but in this case it is
	 * (although just by 1):
	 */

	if concurrentRequests >= maxConcurrentRequest { 
		http.Error(w, 
			fmt.Sprintf("Maximum Requests reached (Total requests processed: %03d)", totalRequestsProcessed),
			http.StatusServiceUnavailable)
	} else {
		napTime, napFunc := randomSleep()
		fmt.Fprintf(w, "Request Serial #%04d: currently %d requests\n", reqSerialNumber, concurrentRequests)
		fmt.Fprintf(w, "Request Serial #%04d: Milliseconds until completion:\t%04d.\n", 
			reqSerialNumber, napTime/time.Millisecond)
		log.Printf("Request Serial #%04d: currently %d requests\n", reqSerialNumber, concurrentRequests) 
		log.Printf("Request Serial $%04d:\t %04d Milliseconds until completion.\n", 
			reqSerialNumber, napTime/time.Millisecond)
		napFunc()
		fmt.Fprintf(w, "Request Serial #%04d: Done (Total requests processed %03d)\n", 
			reqSerialNumber, totalRequestsProcessed)
		log.Printf("Request Serial #%04d: Done (Total requests processed %03d)\n", 
			reqSerialNumber, totalRequestsProcessed)
	}
	concurrentRequests--
	concurrentReqsAtEnd := concurrentRequests
	// There is nothing *wrong* (at least I think there's nothing wrong) with different
	// numbers of concurrent requests coming off the queue, as up to six of them are
	// running simultaneously and independently. So if there are numbers 1, 2, and 3 in
	// the channel, but one gets unlucky and pulls a long sleep time, but 2 and 3 are relatively
	// brief, request 2 will finish first and pull "1" off the queue, and then 3 will finish and
	// pull "2" off the queue, so that when 1 finishes, it pulls "3" off the queue.

	// I was just overcome by this terrible feeling of a fourteen year old happening on this code
	// on github, noticing a bad practice in my code that shows I'm the sort of person who doesn't
	// secure his machine in all the right ways, and that I will wake up tomorrow to find my bank
	// accounts have been emptied, and that I have been declared legally dead in New Jersey, the
	// object of a manhunt in New York, and a suspected child pornographer in Connecticutt. And when 
	// I wake up Suzanne, she screams and says, "Who are you and what have you done with Dennis?"
	//
	// I should not make jokes. Not when I'm floundering like this.

	if concurrentReqsAtEnd != concurrentReqsAtStart {	
		log.Printf("%d Request outstanding at the start: now %d requests pending\n", 
			concurrentReqsAtStart, concurrentReqsAtEnd)
	} else {
		log.Printf("%d Requests still outstanding\n", concurrentRequests)
	}
	log.Printf ("End of request %04d\n", <-requestQueue)	
					
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

