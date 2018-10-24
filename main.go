package main

import (
	"fmt"
	"github.com/fatih/stopwatch"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := stopwatch.Start(0)

		w.Header().Set("Content-Type", "application/json")

		// simulated execution time
		time.Sleep(500 * time.Millisecond)

		s.Stop()

		friendlyElapsed := s.ElapsedTime().String()

		log.Printf("[%s] - %s %s", friendlyElapsed, r.Method, r.URL)

		fmt.Fprintf(w, "{ \"status\": \"success\", \"data\": \"Let it go!\", \"runtime\":\""+friendlyElapsed+"\"}")
	})

	fmt.Println("Web server is started at \":80\"...")
	http.ListenAndServe(":80", nil)
}
