// Test web app
// Wraps a standard HTTP handler with AppOptics instrumentation

package main

import (
	"log"
	"net/http"

	"github.com/solarwindscloud/swo-golang/v1/ao"
)

func bobHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)
	w.Write([]byte(`{"result":"hello from bob"}`))
}

func main() {
	http.HandleFunc("/bob", ao.HTTPHandler(bobHandler))
	http.ListenAndServe(":8081", nil)
}
