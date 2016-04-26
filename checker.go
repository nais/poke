package main

import (
	"strings"
	"time"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"log"
	"flag"
	"os"
)

var (
	influxdbEndpoint = flag.String("influxdbEndpoint", "", "")
	endpointsFile = flag.String("file", "", "")
	interval = flag.Int("interval", 0, "")
)

var usage = `Usage: checker [options...]

Options:

  -influxdbEndpoint	Which InfluxDB endpoint to post the results to
  -file  		JSON-file containing your endpoints
  -interval      	At what interval you want the checks to be performed (run once if omitted)
`

func main() {
	for {
		flag.Parse()

		flag.Usage = func() {
			fmt.Fprint(os.Stderr, usage)
		}

		if flag.NFlag() < 2 {
			usageAndExit("You did not supply enough arguments (everything must be set)")
		}

		data, err := ioutil.ReadFile(*endpointsFile)

		if (err != nil) {
			panic(err)
		}

		checks := []Check{}

		err = json.Unmarshal(data, &checks)

		checksChannel := make(chan Check)
		resultsChannel := make(chan Result)
		done := make(chan bool)
		payloadElements := []string{}

		go check(checksChannel, resultsChannel)
		go func() {
			timestamp := time.Now().Unix()

			for result := range resultsChannel {
				check := result.check
				elem := fmt.Sprintf("checker,environment=%s,application=%s,endpoint=%s value=%d %d", check.Environment, check.Application, check.Endpoint, result.status, timestamp)
				payloadElements = append(payloadElements, elem)
			}

			done <- true
		}()

		for _, check := range checks {
			checksChannel <- check
		}

		close(checksChannel)

		<-done

		postToInfluxDB(strings.Join(payloadElements, "\n"))

		log.Printf("Successfully posted %d check results to InfluxDB", len(checks))

		if (*interval != 0){
			time.Sleep(time.Duration(*interval) * time.Second)
		} else {
			return
		}
	}
}

func postToInfluxDB(payload string) {
	_, err := http.Post(*influxdbEndpoint, "text/plain", strings.NewReader(payload))
	if (err != nil) {
		panic(err)
	}
}

func check(checks <-chan Check, results chan <- Result) {
	for check := range checks {
		resp, err := http.Get(check.Endpoint)

		var resultCode int

		if err != nil {
			resultCode = Unknown
		} else if resp.StatusCode == 200 {
			resultCode = Ok
		} else {
			resultCode = Error
		}

		result := Result{resultCode, check}
		results <- result
	}
	close(results)
}

type Check struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Environment string `json:"environment"`
	Application string `json:"application"`
}

type Result struct {
	status int
	check  Check
}

const (
	Ok = 0
	Error = 1
	Unknown = -1
)

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}