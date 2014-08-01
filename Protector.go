package main

/*
 * A reverse proxy for the server implemented by FiveOnlySrv.go
 * (http://localhost:8181) that protects it by allowing only
 * five requests in and then turning others away with a 503
 * status unavilable error.
 *
 * A bit more than based upon from https://gist.github.com/JalfResi/6287706
 * Thank you sir!
 */
import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

const maxConcurrentRequests = 5
const maxReqsAllowed = 75

var totalReqs func() int // An awful global, so I can get the value out of the function
// that calls the function that creates it and into main()

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

// This function actually isn't the handler but returns the handler, i.e. has a function
// closure. This allows me to set up my Mutex secured increment and decrement functions
// assemble an access queue so that only five of the concurrent requests are being processed
// at any time, and put the function that reports the number of requests in a a global,
// so it may be called from main().
func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	capacityQueue := make(chan int, maxConcurrentRequests)
	for i := 0; i < maxConcurrentRequests; i++ {
		capacityQueue <- i
	}

	incrementConcurrentReqs, decrementConcurrentReqs, tellMeNumReqs := makeTotalConcurrentCounters()
	totalReqs = tellMeNumReqs

	return func(w http.ResponseWriter, r *http.Request) {
		incrementConcurrentReqs() // do this BEFORE pulling off queue; we want to count the reqs that wait
		log.Printf("New Request: now concurrent requests: %03d\n", tellMeNumReqs())
		if tellMeNumReqs() >= maxReqsAllowed {
			log.Printf("Surpassed capacity for %03d concurrent requests: sending back 503 error\n", maxReqsAllowed)
			http.Error(w, "Maximum Capacity Exceeded", http.StatusServiceUnavailable)
			return
		}
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

func main() {
	remote, err := url.Parse("http://localhost:8181/")
	if err != nil {
		log.Fatalf("url.Parse of http://localhost:8181/ failed: %s\n", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))
	http.HandleFunc("/HALT", haltsrv)
	log.Printf("Reverse proxy to http://localhost:8181 about to accept requests at http://locahost:8080\n")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Total reqs made %d\n%s\n", totalReqs(), err)
	}
}
