package main

import "fmt"
import "net/http"
import "time"

func main() {

	for {
		someCheck := Check{"someEndpoint", "http://localhost:5000", "production", "myapp"}
		checks := []Check{someCheck}

		checksChannel := make(chan Check)
		resultsChannel := make(chan Result)

		go performCheck(checksChannel, resultsChannel)

		for _, check := range checks {
			checksChannel <- check
		}

		close(checksChannel)

		payloadElements := []string{}

		// loops until we have all the results back from performCheck
		for {
			result := <-resultsChannel
			check := *result.check
			payloadElements = append(payloadElements, fmt.Sprintf("%d %s %s %s", result.status, check.endpoint, check.environment, check.application))
			payloadElementsSize := len(payloadElements)
			checksSize := len(checks)

			if payloadElementsSize == checksSize {
				fmt.Println("we've added the same amount of payloadelements as checks to perform, which means we're done...")
				close(resultsChannel)
				break
			}
		}

		fmt.Println("apparently its done, lets post payload")

		for _, elem := range payloadElements {
			fmt.Println("", res)
		}

		time.Sleep(5 * time.Second)
	}
}

func performCheck(checks chan Check, results chan Result) {

	check := <-checks

	fmt.Println("Got a check!", check)
	resp, err := http.Get(check.endpoint)

	var resultCode int

	if err != nil {
		resultCode = Unknown
	}

	if resp.StatusCode == 200 {
		resultCode = Ok
	} else {
		resultCode = Error
	}

	result := Result{resultCode, &check}
	fmt.Println("got result", result)

	results <- result
}

type Check struct {
	name        string
	endpoint    string
	environment string
	application string
}

type Result struct {
	status int
	check  *Check
}

const (
	Ok      = 0
	Error   = 1
	Unknown = -1
)
