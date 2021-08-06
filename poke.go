package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	Ok    = 0
	Error = 1
)

var (
	influxdbEndpoint string
	endpointsFile    string
	measurementName  string
	timeout          int64
	interval         int64
	counter          int64
)

type Poke struct {
	Endpoint string            `json:"endpoint"`
	Tags     map[string]string `json:"tags"`
}

type Result struct {
	latency int64
	status  int
	poke    Poke
}

func init() {
	flag.StringVar(&influxdbEndpoint, "influxdbEndpoint", "", "Which InfluxDB endpoint to post the results to (required)")
	flag.StringVar(&endpointsFile, "endpoints", "", "JSON-file containing your endpoints (required)")
	flag.StringVar(&measurementName, "measurement-name", "pokes", "Name of InfluxDB measurement to write data to")
	flag.Int64Var(&timeout, "timeout", 2, "default request timeout (seconds)")
	flag.Int64Var(&interval, "interval", 0, "At what interval you want the pokes to be performed (run once if omitted)")
	flag.Parse()
}

func main() {
	flag.Parse()

	if len(influxdbEndpoint) == 0 || len(endpointsFile) == 0 {
		flag.Usage()
		log.Fatal("missing required configuration")
	}

	pokes, err := pokes(endpointsFile)

	if err != nil {
		log.Fatalf("unable to extract endpoints to poke from file: %s: %s", endpointsFile, err)
	}

	client := http.Client{
		Timeout: time.Second * time.Duration(timeout),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}}

	for {
		timestamp := time.Now().Unix()

		var results []Result

		var payloadElements []string
		for _, poke := range pokes {
			resultCode := Error
			errorMsg := ""
			var elapsed int64
			start := time.Now()
			resp, err := client.Get(withCounter(poke.Endpoint))
			if err != nil {
				errorMsg = fmt.Sprintf("unable to perform request: %s", err)
				log.Printf("error: unable to perform request to endpoint %s: %s", poke.Endpoint, err)
			} else {
				elapsed = time.Since(start).Milliseconds()
				if resp.StatusCode == 200 {
					resultCode = Ok
				} else {
					log.Printf("got an unsuccessful statuscode %d for endpoint %s\n", resp.StatusCode, poke.Endpoint)
					errorMsg = fmt.Sprintf("got an unsuccessful statuscode: %d", resp.StatusCode)
					b, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						errorMsg += fmt.Sprintf(". Unable to read body: %s", err)
					} else {
						errorMsg += fmt.Sprintf(". Response body: %s", string(b))
						_ = resp.Body.Close()
					}
				}
			}

			elem := fmt.Sprintf("%s,%s value=%d,counter=%d,err=\"%s\" %d", measurementName, tags(poke), resultCode, counter, errorMsg, timestamp)
			payloadElements = append(payloadElements, elem)
			results = append(results, Result{elapsed, resultCode, poke})
		}

		if err := postToInfluxDB(strings.Join(payloadElements, "\n")); err != nil {
			log.Print(err)
		} else {
			log.Printf("Successfully posted %d pokes to InfluxDB\n", len(pokes))
		}

		// if no interval is provided, we only run once
		if interval == 0 {
			return
		}

		counter++
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func withCounter(endpoint string) string {
	return strings.ReplaceAll(fmt.Sprintf("%s/?counter=%d", endpoint, counter), "//?", "/?")
}

func tags(poke Poke) string {
	pairs := []string{fmt.Sprintf("endpoint=%s", escapeSpecialChars(poke.Endpoint))}

	for key, value := range poke.Tags {
		pairs = append(pairs, fmt.Sprintf("%s=%s", escapeSpecialChars(key), escapeSpecialChars(value)))
	}

	return strings.Join(pairs, ",")
}

func pokes(endpointsFile string) (pokes []Poke, err error) {
	data, err := ioutil.ReadFile(endpointsFile)

	if err != nil {
		return nil, fmt.Errorf("unable to read endpoints file: %s: %s", endpointsFile, err)
	}

	if err := json.Unmarshal(data, &pokes); err != nil {
		return nil, fmt.Errorf("unable to unmarshal endpoint config: %s", err)
	}

	return
}

func postToInfluxDB(payload string) error {
	log.Printf("Posting the following payload to InfluxDB (%s)\n%s", influxdbEndpoint, payload)
	resp, err := http.Post(influxdbEndpoint, "text/plain", strings.NewReader(payload))

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

// escapeSpecialChars escapes '=' and ','
func escapeSpecialChars(string string) string {
	return strings.Replace(strings.Replace(string, "=", "\\=", -1), ",", "\\,", -1)
}
