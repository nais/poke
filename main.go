package main

import "fmt"
import "net/http"
import "time"
import "strings"

func main() {

	someCheck := Check{"someEndpoint", "http://localhost:5000", "production", "myapp"}
	someOtherCheck := Check{"someOtherEndpoint", "http://localhost:5000", "production", "someApp"}
	someThirdCheck := Check{"someThirdEndpoint", "http://localhost:5000", "production", "appapp"}
	checks := []Check{someCheck, someOtherCheck, someThirdCheck}

	checkCount := len(checks)
	fmt.Printf("We have %d checks\n", checkCount)

	checksChannel := make(chan Check)
	resultsChannel := make(chan Result)
	done := make(chan bool)
	payloadElements := []string{}

	go check(checksChannel, resultsChannel)
	go func() {
		timestamp := time.Now().Unix()
		for i := 0; i < checkCount; i++ {
			result := <-resultsChannel
			check := *result.check
			fmt.Println("Got result", result.status)
			payloadElements = append(payloadElements, fmt.Sprintf("checker,environment=%s,application=%s,endpoint=%s value=%d %d", check.environment, check.application, check.endpoint, result.status, timestamp))
		}
		done <- true
	}()

	for _, check := range checks {
		checksChannel <- check
	}

	<-done
	fmt.Printf("Apparently we're done, posting payload\n%s\n", strings.Join(payloadElements, "\n"))

	return
}

//func transformResults()

func check(checks <-chan Check, results chan<- Result) {
	for check := range checks {
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
		fmt.Println("done checking, posting result to channel")
		results <- result
	}
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
