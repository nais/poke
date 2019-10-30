package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	influxdbEndpoint = flag.String("influxdbEndpoint", "", "")
	endpointsFile    = flag.String("file", "", "")
	interval         = flag.Int("interval", 0, "")
	debug            = flag.Bool("debug", false, "")
)

var usage = `Usage: poker [options...]

Options:

  -influxdbEndpoint	Which InfluxDB endpoint to post the results to
  -file  		JSON-file containing your endpoints
  -interval      	At what interval you want the pokes to be performed (run once if omitted)
  -debug		Prints payload
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

		if err != nil {
			log.Fatal("unable to read file, ", *endpointsFile)
			panic(err)
		}

		if !*debug {
			log.SetOutput(ioutil.Discard)
		}

		var pokes []Poke

		err = json.Unmarshal(data, &pokes)

		timestamp := time.Now().Unix()

		var results []Result

		for _, poke := range pokes {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			resp, err := http.Get(poke.Endpoint)

			var resultCode = Error
			if err != nil {
				log.Println(err)
			} else {
				if resp.StatusCode == 200 {
					resultCode = Ok
				}
			}

			results = append(results, Result{resultCode, poke})
		}

		var payloadElements []string
		for _, result := range results {
			poke := result.poke
			elem := fmt.Sprintf("pokes,environment=%s,application=%s,endpoint=%s value=%d %d", poke.Environment, poke.Application, escapeSpecialChars(poke.Endpoint), result.status, timestamp)
			payloadElements = append(payloadElements, elem)
		}

		if err := postToInfluxDB(strings.Join(payloadElements, "\n")); err != nil {
			log.Println(err)
		} else {
			log.Printf("Successfully posted %d pokes to InfluxDB", len(pokes))
		}

		if *interval != 0 {
			time.Sleep(time.Duration(*interval) * time.Second)
		} else {
			return
		}
	}
}

func postToInfluxDB(payload string) error {
	log.Printf("Posting the following payload to InfluxDB (%s)\n%s", *influxdbEndpoint, payload)

	resp, err := http.Post(*influxdbEndpoint, "text/plain", strings.NewReader(payload))

	if err != nil {
		return fmt.Errorf("unable to post pokes to InfluxDB: %s", err)
	}

	if resp != nil && resp.StatusCode != 204 {
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("unable to post pokes to InfluxDB, got HTTP status code %d and body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func escapeSpecialChars(string string) string {
	equallessString := strings.Replace(string, "=", "\\=", -1)
	return strings.Replace(equallessString, ",", "\\,", -1)
}

type Poke struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Environment string `json:"environment"`
	Application string `json:"application"`
}

type Result struct {
	status int
	poke   Poke
}

const (
	Ok    = 0
	Error = 1
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
