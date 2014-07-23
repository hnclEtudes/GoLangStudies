package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"math/rand"
)

const requestURL = "http://localhost:8181" // The URL for the server created by aServer.LaunchServer()
const maxRequest = 25

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

// Fires the request at the server, gets back a response, sends it back via a channel
// and returns. Runs as a goRoutine so we can make  multiple requests (assuming the 
// randomSleep function() in main gives us favorable numbers)
func makeRequest(requestNumber int, respChannel chan *http.Response)  {
	var myClient http.Client
	fmt.Printf("\tmakeRequest(): creating new request #%d\n", requestNumber)
	myRequest, _ := http.NewRequest("GET", requestURL, nil)
	fmt.Printf("\tmakeRequest(): Sending for response (issuing Do()) #%d\n", requestNumber)
	myResponse, _ := (&myClient).Do(myRequest)
	respChannel <- myResponse
	if requestNumber == maxRequest { // Channel has hit capacity anyway, so close it
		close(respChannel)
	}
	return
}

var semaphore = make(chan bool, 1)

func getResponse(responseNumber int, myResponse http.Response) {
	var busy bool = true
	semaphore <- busy	// Since this channel can only hold one int at a time,
								// it should block multiple concurrent calls

	fmt.Printf("Retrieving request #%04d.\n", responseNumber)
	fmt.Printf("(NB:  the request # in the body of the probably won't match\n")
	if myResponse.StatusCode != http.StatusOK {
		fmt.Printf("Response Status NOT 200: %d\n[%s]", myResponse.StatusCode, myResponse.Status)
	} else {
		lenContent, _ := strconv.Atoi(myResponse.Header.Get("Content-Length"))
		theContent := make([]byte, lenContent)
		actuallyRead, _ := myResponse.Body.Read(theContent)
		myResponse.Body.Close()
		fmt.Printf(
			"Response Info:\n[Response Status Code %d]\n[Response Status '%s']\n[Length: %d]\n\nResponse Content:\n[%s]\n",
			myResponse.StatusCode, myResponse.Status, actuallyRead, theContent)
	}
	<- semaphore
	return
}

func main() {

	fmt.Printf("Informational messages print out from the routines makeRequest() and\n")
	fmt.Printf("getResponse(). Both are run as go routines (although the second is gated\n")
	fmt.Printf("to have but one instance running at a time)\n")

	respChannel := make(chan *http.Response)
	for i := 0; i < maxRequest; i++ {
		fmt.Printf("Making Request Number #%d:\n\n", i + 1)
		
		go makeRequest (i + 1, respChannel)
		napTime, napFunc := randomSleep()
		fmt.Printf("About to sleep %04d  milliseconds (compensate for random time in server reples)...", 
			napTime/time.Millisecond)
		napFunc()
		fmt.Println("Done\n")
	}

	// Now get the responses:
	i := 0
	for thisResponse := range respChannel {
		go getResponse(i + 1, *thisResponse)
		i++
	}
}
