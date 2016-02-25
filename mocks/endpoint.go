package main

import (
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "hello")
}

func main() {
    http.HandleFunc("/", handler)
	fmt.Println("listening to http traffic @ http://localhost:5000")
    http.ListenAndServe(":5000", nil)
}
