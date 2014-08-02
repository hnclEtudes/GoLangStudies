package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

const serverURL = "http://localhost:8181" // The URL for the server created by SorryServer.go
const maxRequest = 100

var requestURL string // This will be serverURL unless we find a variable in the environment
// HARNESS_URL set to another URL, which *SHOULD* be http://localhost:8080, the URL of the
// reverse proxy implemented in Protector.go. No checking is performed on this, which could
// lead to some bizarre errors.

// Called as a goroutine. Note the semaphore around just the log.Printf outputs:
// we can let each routine talk to the server on its own, but only one routine
// at a time can write to the log.
// Also, since these won't finish in the order they're called, we have to count
// internally how many are finished: when we reach maxRequest, that is, this is the
// 100th goroutine that finished, is when we send true down the doneChannel.
// Previously, when I sent the request number in as a parameter,
// I closed the channel when it reached 100. Since these goroutines are finishing
// after random waits that get longer and longer, this only meant the last request
// dispatched was sure to finish; others, say, #60, which had a wait time that
// would have it finish after #100 did would then try to send on a closed channel.
// Oops.

func setupRequestAndReplyFunc() (func(int), chan bool) {
	var numReqsFinished = 0 // Counts the times makeRequestAndGetResponse returns:
	// when it is equal to the number of times its called,
	// we know all the goroutines are done.
	var semaphore = make(chan bool, 1) // Make sure one write to the log doesn't get clobbered by another

	var doneChannel = make(chan bool, 1) // Signals to other routine that all goroutines are done
	makeRequestAndGetResponse := func(requestNumber int) {
		var myClient http.Client

		myRequest, _ := http.NewRequest("GET", requestURL, nil)
		myRequest.Header.Add("Harness-Request-Number", fmt.Sprintf("\"%04d\"", requestNumber))
		myResponse, _ := (&myClient).Do(myRequest)
		lenContent, _ := strconv.Atoi(myResponse.Header.Get("Content-Length"))
		theContent := make([]byte, lenContent)
		actuallyRead, _ := myResponse.Body.Read(theContent)
		myResponse.Body.Close()
		semaphore <- true
		log.Printf("Retrieval #%04d (Compare with server # in Content).\n", requestNumber)
		log.Printf("\tResponse Status Code: %d\n", myResponse.StatusCode)
		log.Printf("\tResponse Status '%s'\n", myResponse.Status)
		log.Printf("\tLength: %d\n", actuallyRead)
		log.Printf("\tResponse Content:\n%s\n", theContent)
		numReqsFinished++
		<-semaphore
		if numReqsFinished == maxRequest {
			doneChannel <- true
		}
	}
	return makeRequestAndGetResponse, doneChannel
}

func main() {

	requestURL = serverURL
	changeTarget := os.Getenv("HARNESS_URL")
	if changeTarget != "" {
		requestURL = changeTarget
	}

	log.Printf("Sending requests to %s\n", requestURL)
	log.Printf("Informational messages come from the goroutine makeRequestAndGetResponse().\n")
	log.Printf("Although up to %d instances are running as goroutines and interacting with\n", maxRequest)
	log.Printf("the server, the output is gated so that only one instance prints messages at\n")
	log.Printf("a time.\n")

	makeRequestAndGetResponse, doneChannel := setupRequestAndReplyFunc()

	for i := 0; i < maxRequest; i++ {
		go makeRequestAndGetResponse(i + 1)
	}
	log.Printf("Fired all %d goroutines...\n", maxRequest)

	// wait until all routines are done:
	<-doneChannel
	log.Printf("...and now all have finished.\n")
}
