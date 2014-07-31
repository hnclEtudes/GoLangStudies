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
import(
	"log"
	"sync"
	"net/url"
	"net/http"
	"net/http/httputil"
)

const maxConcurrentRequests = 5

var totalReqs	func() (int)

func makeTotalConcurrentCounters ()(incrementTotal func(), decrementTotal func(), tellMeTotal func() (int)) {
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

	
func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	capacityQueue := make (chan int, maxConcurrentRequests)	
	for i:= 0; i < maxConcurrentRequests; i++ {
		capacityQueue <- i
	}

	incrementConcurrentReqs, decrementConcurrentReqs, tellMeNumReqs := makeTotalConcurrentCounters()
	totalReqs = tellMeNumReqs

	return func(w http.ResponseWriter, r *http.Request) {
		// This two item read is the idiom in the language spec to check if the
		// queue is empty (http://golang.org/ref/spec#Receive_operator)
		// Regardless of type, if the queue is empty, it is assigned false

		incrementConcurrentReqs() // do this BEFORE pulling off queue; we want to count the reqs that wait
		queueSlot := <- capacityQueue 
		log.Printf("queueSlot %d, Client requested %s, forwarding to localhost:8181 (totalReqs = %d)\n", 
			queueSlot, r.URL, tellMeNumReqs())
		p.ServeHTTP(w, r)
		capacityQueue <- queueSlot
		decrementConcurrentReqs()
	}
}

func main() {
	remote, err := url.Parse("http://localhost:8181/")
	if err != nil {
		log.Fatalf("url.Parse of http://localhost:8181/ failed: %s\n", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Total reqs made %d\n%s\n", totalReqs(), err)
	}
}

