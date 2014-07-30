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
	"net/url"
	"net/http"
	"net/http/httputil"
)

const maxConcurrentRequests = 5

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	capacityQueue := make (chan int, maxConcurrentRequests)	
	for i:= 0; i < maxConcurrentRequests; i++ {
		capacityQueue <- i
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// This two item read is the idiom in the language spec to check if the
		// queue is empty (http://golang.org/ref/spec#Receive_operator)
		// Regardless of type, if the queue is empty, it is assigned false
		queueSlot, ok := <- capacityQueue 
		if ok == false {
			log.Printf("Client request for %s exceeds capacity of localhost:8181, returning 503 error\n", r.URL)
			http.Error(w, "Service Unavailble (error 503): server capacity exceeded", http.StatusServiceUnavailable)
		}
		log.Printf("queueSlot %d, Client requested %s, forwarding to localhost:8181\n", queueSlot, r.URL)
		p.ServeHTTP(w, r)
		capacityQueue <- queueSlot
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
		panic(err)
	}
}

