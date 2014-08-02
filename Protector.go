package main

/*
 * A reverse proxy for the server implemented by SorryServer.go
 * (http://localhost:8181) that protects it by allowing only
 * five requests in and then, depending on the value of environment
 * variable PROTECTOR_OVERCAPACITY, either discards them (when PROTECTOR_OVERCAPACITY
 * is "503") with a 503 error or holding them until a slot opens up (PROTECTOR_OVERCAPACITY=200)
 *
 * A bit more than based upon from https://gist.github.com/JalfResi/6287706
 * Thank you sir!
 */
import (
	"os"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

const maxConcurrentRequests = 5
const maxReqsAllowed = 75
const holdOvercapacityReqs = "200"
const rejectOvercapacityReqs = "503"

var totalReqs func() int // A global, set within makeDefaultHandler, that returns the total
// concurrent requests pending. I have to use a global; making it a return value would break
// the footprint for handler function (and it is defined within a handler)
// from the function that creates it.

// makeTotalConcurrentCounters uses the methods Lock() and Unlock() on a *sync.Mutex to control
// access to a counter that holds the number of concurrent requests the server is holding at
// any time (since handlers are called as goroutines, and could stomp on/undo each other's
// work.
// Example: the handler is invoked when the total concurrent requests is (unambiguously) 30.
// As they dash through their other duties, they reach the point here the counter must be
// incremented. From the perspective of both of these functions, it's 30, so after they
// both finish it's 31 when it should be 32.
func makeTotalConcurrentCounters() (incrementTotal func(), decrementTotal func(), tellMeTotal func() int) {
	var counterAccess sync.Mutex
	totalRequests := 0

	incrementTotal = func() {
		(&counterAccess).Lock()
		totalRequests++
		(&counterAccess).Unlock()
	}

	decrementTotal = func() {
		(&counterAccess).Lock()
		totalRequests--
		(&counterAccess).Unlock()
	}

	tellMeTotal = func() (theTotal int) {
		return totalRequests
	}

	return
}

// This function isn't the default handler but returns the default handler, i.e. has 
// a function closure. This allows me to set up my Mutex secured increment and decrement functions
// and assemble an access queue so that only five of the concurrent requests are being processed
// at any time, and put the function that reports the number of requests in a a global,
// so it may be called from main().
func makeDefaultHandler(p *httputil.ReverseProxy, overcapacityBehavior string) func(http.ResponseWriter, *http.Request) {
	capacityQueue := make(chan int, maxConcurrentRequests)	// the Brass Rings, as it were.
	for i := 0; i < maxConcurrentRequests; i++ {			// grab one, and you're in. :)
		capacityQueue <- i
	}

	incrementConcurrentReqs, decrementConcurrentReqs, tellMeNumReqs := makeTotalConcurrentCounters()
	totalReqs = tellMeNumReqs

	return func(w http.ResponseWriter, r *http.Request) {
		incrementConcurrentReqs() // do this BEFORE pulling off queue; we want to count the reqs that wait
		log.Printf("New Request: now concurrent requests: %03d\n", tellMeNumReqs())
		if overcapacityBehavior == rejectOvercapacityReqs {	// Reject more than five concurrent requests
			if tellMeNumReqs() > maxReqsAllowed {
				log.Printf("Surpassed capacity for %03d concurrent requests: sending back 503 error\n", maxReqsAllowed)
				http.Error(w, "Maximum Capacity Exceeded", http.StatusServiceUnavailable)
				return
			}
		}	// overcapacityBehavior == rejectOvercapacityReqs
		queueSlot := <-capacityQueue
		log.Printf("queueSlot %d, Client requested %s, forwarding to localhost:8181 (totalReqs = %d)\n",
			queueSlot, r.URL, tellMeNumReqs())
		p.ServeHTTP(w, r)
		capacityQueue <- queueSlot
		decrementConcurrentReqs()
		log.Printf("Request completed: now concurrent requests: %03d\n", tellMeNumReqs())
	}
}

func haltsrv(w http.ResponseWriter, r *http.Request) {
	log.Fatal("Stopping proxy server\n")
}

// initMsgs() gets the value of PROTECTOR_OVERCAPACITY from the environment and prints an appropriate
// message for the value. Note that PROTECTOR_OVERCAPACITY doesn't need to be set; the default behavior
// is to hold requests past five concurrent ones and let one in once a slot frees up.
func initMsgs () (overcapacityBehavior string) {
	overcapacityBehavior = os.Getenv("PROTECTOR_OVERCAPACITY")

	switch {
	case overcapacityBehavior == "": 
		overcapacityBehavior = holdOvercapacityReqs // overcapacityBehavior == "200"
		log.Printf("Reverse proxy default behavior: greater than %02d concurrent requests held until they\n", maxReqsAllowed)
		log.Println("can be processed")
    case overcapacityBehavior == holdOvercapacityReqs: // overcapacityBehavior == "200"
		log.Printf("Reverse proxy will hold greater than %02d concurrent requests until they can be\n", maxReqsAllowed)
		log.Println("processed. This is default behavior, i.e., the environment variable PROTECTOR_OVERCAPACITY")
		log.Println("does not need to be set.")
	case overcapacityBehavior == rejectOvercapacityReqs:	// overcapacityBehavior == "503"
		log.Printf("Reverse proxy will discard with status 503 any concurrent requests greater than %02d\n", maxReqsAllowed)
	default:
		log.Printf("Warning: '%s' is not a legitimate value for PROTECTOR_OVERCAPACITY\n", overcapacityBehavior)
		log.Println("PROTECTOR_OVERCAPACITY must either be (1) unset, (2) '200', or '503'")
		log.Println("Default behavior (PROTECTOR_OVERCAPACITY is unset) is the same as PROTECTOR_OVERCAPACITY = '200', i.e.")
		log.Println("if more than five concurrent requests are pending, they just wait until a slot opens up.")
		log.Fatal("Aborting.")
	}
	return overcapacityBehavior
}

func main() {
	remote, err := url.Parse("http://localhost:8181/")
	if err != nil {
		log.Fatalf("url.Parse of http://localhost:8181/ failed: %s\n", err)
	}
	overcapacityBehavior := initMsgs()

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", makeDefaultHandler(proxy, overcapacityBehavior))
	http.HandleFunc("/HALT", haltsrv)
	log.Printf("Reverse proxy to http://localhost:8181 about to accept requests at http://locahost:8080\n")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Total reqs made %d\n%s\n", totalReqs(), err)
	}
}
