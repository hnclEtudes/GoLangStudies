package main

import (
	"os"
	"log"
	"net/http"
	"strconv"
)

const serverURL  = "http://localhost:8181" // The URL for the server created by SorryServer.go
const proxyURL   = "http://localhost:8080" // The URL for the reverse proxy created by Protector.go
const maxRequest = 100

var requestURL string	// This will be serverURL unless we find a variable in the environment
						// HARNESS_TO_REV_PROXY set to "yes", in which case it will be set
						// to proxyURL, the reverse proxy made by Protector.go


var numReqsFinished = 0
var doneChannel = make(chan bool, 1)
var semaphore = make(chan bool, 1)
// Called as a goroutine. Note the semaphore around just the log.Printf outputs: 
// we can let each routine talk to the server on its own, but only one routine
// at a time can write to the log.
// Also, since these won't finish in the order they're called, we have to count
// internally how many are finished: when we reach maxRequest, we send true down
// the doneChannel

func makeRequestAndGetResponse (requestNumber int)  {
	var myClient http.Client

	myRequest, _ := http.NewRequest("GET", requestURL, nil)
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
	log.Printf("\tResponse Content: %s\n", theContent)
	numReqsFinished++
	<- semaphore
	if numReqsFinished == maxRequest {
		doneChannel <- true
	}
	return
}

func main() {

	requestURL = serverURL
	changeTarget := os.Getenv("HARNESS_TO_REV_PROXY")
	if changeTarget == "yes" {
		requestURL = proxyURL
	}

	log.Printf("Sending requests to %s\n", requestURL)
	log.Printf("Informational messages print out from the routines makeRequest() and\n")
	log.Printf("getResponse(). Both are run as go routines (although the second is gated\n")
	log.Printf("to have but one instance running at a time)\n")

	for i := 0; i < maxRequest; i++ {
		go makeRequestAndGetResponse (i + 1)
	}

	// wait until all routines are done:
	<- doneChannel
}
