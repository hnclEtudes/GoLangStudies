package main

import (
	"os"
	"log"
	"net/http"
	"strconv"
	"time"
	"math/rand"
)

const serverURL  = "http://localhost:8181" // The URL for the server created by SorryServer.go
const proxyURL   = "http://localhost:8080" // The URL for the reverse proxy created by Protector.go
const maxRequest = 100

var requestURL string	// This will be serverURL unless we find a variable in the environment
						// HARNESS_TO_REV_PROXY set to "yes", in which case it will be set
						// to proxyURL, the reverse proxy made by Protector.go

// randomSleep() returns a (pseudo) random number of milliseconds, up to maxSleep
// and a function that sleeps for just that amount of time. 
func randomSleep(maxSleep int) (napTime time.Duration, napFunc func ()) {
	napTime = time.Duration(rand.Intn(maxSleep)) * time.Millisecond
	napFunc = func() {
		time.Sleep(napTime)
	}
	return
}

// Fires the request at the server, gets back a response, sends it back via a channel
// and returns. Runs as a goRoutine so we can make  multiple requests (assuming the 
// randomSleep function() in main gives us favorable numbers)
func makeRequest(requestNumber int, respChannel chan *http.Response)  {
	var myClient http.Client
	myRequest, _ := http.NewRequest("GET", requestURL, nil)
	log.Printf("\tmakeRequest(): Sending request (issuing Do()) #%d\n", requestNumber)
	myResponse, _ := (&myClient).Do(myRequest)
	respChannel <- myResponse
	if requestNumber == maxRequest { // Channel has hit capacity anyway, so close it
		close(respChannel)
	}
	return
}


func getResponse(responseNumber int, myResponse http.Response) {
	log.Printf("Retrieving request #%04d.\n", responseNumber)
	log.Printf("(NB:  the request # in the body of the may not match)\n")
	if myResponse.StatusCode != http.StatusOK {
		log.Printf("Response Status NOT 200: %d\n[%s]", myResponse.StatusCode, myResponse.Status)
	} else {
		lenContent, _ := strconv.Atoi(myResponse.Header.Get("Content-Length"))
		theContent := make([]byte, lenContent)
		actuallyRead, _ := myResponse.Body.Read(theContent)
		myResponse.Body.Close()
		log.Printf(
			"Response Info:\n[Response Status Code %d]\n[Response Status '%s']\n[Length: %d]\n\nResponse Content:\n[%s]\n",
			myResponse.StatusCode, myResponse.Status, actuallyRead, theContent)
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

	respChannel := make(chan *http.Response)
	for i := 0; i < maxRequest; i++ {
		go makeRequest (i + 1, respChannel)
	}

	// Now get the responses:
	i := 0
	for thisResponse := range respChannel {
		go getResponse(i + 1, *thisResponse)
		i++
	}
}
