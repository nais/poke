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
	interval         int
)

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

func init() {
	flag.StringVar(&influxdbEndpoint, "influxdbEndpoint", "", "Which InfluxDB endpoint to post the results to")
	flag.StringVar(&endpointsFile, "file", "", "JSON-file containing your endpoints")
	flag.IntVar(&interval, "interval", 0, "At what interval you want the pokes to be performed (run once if omitted)")
	flag.Parse()
}

func main() {
	flag.Parse()

	if flag.NFlag() < 2 {
		flag.Usage()
		log.Fatal("You did not supply enough arguments (everything must be set)")
	}

	data, err := ioutil.ReadFile(endpointsFile)

	if err != nil {
		log.Fatalf("unable to read file: %s: %s", endpointsFile, err)
	}

	var pokes []Poke

	if err := json.Unmarshal(data, &pokes); err != nil {
		log.Fatalf(fmt.Sprintf("unable to unmarshal endpoint config: %s", err))
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	for {
		timestamp := time.Now().Unix()

		var results []Result

		var payloadElements []string
		for _, poke := range pokes {
			var resultCode = Error
			resp, err := http.Get(poke.Endpoint)

			if err != nil {
				fmt.Println("error: ", err)
			} else {
				if resp.StatusCode == 200 {
					resultCode = Ok
				} else {
				    fmt.Printf("got an unsuccessful statuscode %d for endpoint %s\n", resp.StatusCode, poke.Endpoint)
				}
			}

			elem := fmt.Sprintf("pokes,environment=%s,application=%s,endpoint=%s value=%d %d", poke.Environment, poke.Application, escapeSpecialChars(poke.Endpoint), resultCode, timestamp)
			payloadElements = append(payloadElements, elem)
			results = append(results, Result{resultCode, poke})
		}

		if err := postToInfluxDB(strings.Join(payloadElements, "\n")); err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Successfully posted %d pokes to InfluxDB\n", len(pokes))
		}

		if interval != 0 {
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			return
		}
	}
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
