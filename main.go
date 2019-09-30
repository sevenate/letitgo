//+build !test
//go:generate esc -o static.go -prefix static static
//go:generate goversioninfo

package main // import "github.com/sevenate/letitgo"

import (
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"github.com/fatih/stopwatch"
	"github.com/justinas/alice"
	"github.com/logrusorgru/aurora"
	"github.com/vharitonsky/iniflags"
	"github.com/xi2/httpgzip"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	host     = flag.String("host", "localhost", "Host name for web service")
	port     = flag.Int("port", 443, "HTTPS server port to listen")
	portHTTP = flag.Int("portHTTP", 80, "HTTP server port to listen (will be redirected to HTTPS port)")
	debug    = flag.Bool("debug", false, "Use local files in static/ subfolder instead of embedded")
	about    = flag.Bool("version", false, "Get the application version")

	version = "dev"
	date    = "unknown"
	commit  = "none"

	numberOfHTTPCalls = expvar.NewMap("app-stats")
)

type info struct {
	Version string
	Commit  string
	Date    string
}

func getInfo() interface{} {
	return info{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

type nodeInfo struct {
	Version         string `json:"version"`
	CurrentBlock    string `json:"current_block"`
	UncheckedBlocks string `json:"unchecked_blocks"`
	Peers           string `json:"peers"`
	SyncStatus      string `json:"sync_status"` // is this usefull info?
	Uptime          string `json:"uptime"`
	Load            string `json:"load"`
	MemoryUsed      string `json:"memory_used"`
	LedgerFileSize  string `json:"ledger_file_size"`
}

type networkInfo struct {
	OnlineRepresentatives string `json:"online_representatives"`
	OnlineVotingWeight    string `json:"online_voting_weight"`
	PeersV17              string `json:"peers_v17"`
	PeersV16              string `json:"peers_v16"`
	PeersV15              string `json:"peers_v15"`
	PeersV14              string `json:"peers_v14"`
	PeersV13              string `json:"peers_v13"`
}

type representativeInfo struct {
	AccountPart0 string `json:"account_part_0"` // "nano_" prefix
	AccountPart1 string `json:"account_part_1"` // highlighted 7 leading characters
	AccountPart2 string `json:"account_part_2"` // remaining 47 characters in the middle
	AccountPart3 string `json:"account_part_3"` // highlighted 6 trailing characters
	VotingWeight string `json:"voting_weight"`
	Delegators   string `json:"delegators"`
	Balance      string `json:"balance"` // is this usefull info for rep account?
	Pending      string `json:"pending"` // is this usefull info for rep account?
	Location     string `json:"location"`
}

type nodeStatusSnapshot struct {
	NodeInfo           nodeInfo           `json:"node"`
	NetworkInfo        networkInfo        `json:"network"`
	RepresentativeInfo representativeInfo `json:"representative"`
}

// DEMO DATA
var statusSnapshot = nodeStatusSnapshot{
	NodeInfo: nodeInfo{
		Version:         "Nano 19.0",
		CurrentBlock:    "31,966,872",
		UncheckedBlocks: "48",
		Peers:           "308",
		SyncStatus:      "100 %",
		Uptime:          "97.286 %",
		Load:            "2.02",
		MemoryUsed:      "2,749 / 7,976 MB",
		LedgerFileSize:  "19.459 GB",
	},
	NetworkInfo: networkInfo{
		OnlineRepresentatives: "118",
		OnlineVotingWeight:    "114,559,823 Nano (85.97 %)",
		PeersV17:              "213 (75.53%)",
		PeersV16:              "49 (17.38%)",
		PeersV15:              "8 (2.84%)",
		PeersV14:              "10 (3.55%)",
		PeersV13:              "2 (0.71%)",
	},
	RepresentativeInfo: representativeInfo{
		AccountPart0: "nano_",
		AccountPart1: "1fnx59b",
		AccountPart2: "qpx11s1yn7i5hba3ot5no4ypy971zbkp5wtium3yyafpwhh",
		AccountPart3: "wkq8fc",
		VotingWeight: "274,725 NANO",
		Delegators:   "722",
		Balance:      "324 NANO",
		Pending:      "0 NANO",
		Location:     "Clifton, US",
	},
}

func redirectTLSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostAddress := strings.Split(r.Host, ":")[0]

		target := "https://" + hostAddress + ":" + strconv.Itoa(*port) + r.URL.Path

		if len(r.URL.RawQuery) > 0 {
			target += "?" + r.URL.RawQuery
		}

		log.Printf("Redirect to: %s", aurora.BrightGreen(target))

		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}

func format(n int64) string {
	in := strconv.FormatInt(n, 10)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ' '
		}
	}
}

func loggerHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := stopwatch.Start(0)
		next.ServeHTTP(w, r)
		s.Stop()
		friendlyElapsed := s.ElapsedTime().Nanoseconds()

		// as an option to determine if this is HTTP or HTTPS request
		// check the field TLS *tls.ConnectionState on http.Request for nil
		log.Printf("[%13s µs] - %s %s %s", aurora.Cyan(format(friendlyElapsed)), r.Proto, r.Method, r.URL)
	})
}

func urlFilterHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			next.ServeHTTP(w, r)
		case "/api":
			next.ServeHTTP(w, r)
		case "/debug":
			next.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func statsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numberOfHTTPCalls.Add(r.URL.Path, 1)
		next.ServeHTTP(w, r)
	})
}

func gzipHandler(next http.Handler) http.Handler {
	return httpgzip.NewHandler(next, nil)
}

func apiHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")

			// simulated execution time
			//time.Sleep(500 * time.Millisecond)

			json.NewEncoder(w).Encode(statusSnapshot)

		case "POST":
			d := json.NewDecoder(r.Body)
			d.DisallowUnknownFields() // catch unwanted fields

			var incomingData nodeStatusSnapshot

			err := d.Decode(&incomingData)
			if err != nil {
				// bad JSON or unrecognized json field
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// optional extra check
			if d.More() {
				http.Error(w, "extraneous data after JSON object", http.StatusBadRequest)
				return
			}

			statusSnapshot = incomingData

		default:
		}
	})
}

func main() {
	iniflags.Parse() // instead of flag.Parse()

	var s string

	t, err := time.Parse(time.RFC3339, date)

	if err == nil {
		s = fmt.Sprintf("%s.%d%02d%02d-%02d%02d%02d.%s", version, t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second(), commit)
	} else {
		s = fmt.Sprintf("%s.%s.%s", version, date, commit)
	}

	if *about {
		fmt.Println(s)
		return
	}

	expvar.Publish("app-info", expvar.Func(getInfo))

	mux := http.NewServeMux()

	defaultChain := alice.New(loggerHandler, urlFilterHandler, statsHandler, gzipHandler)
	redirectChain := alice.New(loggerHandler, urlFilterHandler)

	mux.Handle("/", defaultChain.Then(http.FileServer(FS(*debug))))
	mux.Handle("/api", defaultChain.Then(apiHandler()))
	mux.Handle("/debug", defaultChain.Then(expvar.Handler()))

	srv := &http.Server{
		Addr:           "localhost:" + strconv.Itoa(*port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	pid := os.Getpid()
	startedText := aurora.Sprintf(aurora.BrightWhite("Web server %s [PID - %d] is listening at %s ..."), s, aurora.BrightYellow(pid), aurora.BrightGreen("https://"+srv.Addr))
	fmt.Println(startedText)

	// redirect all HTTP -> HTTPS
	go http.ListenAndServe(*host+":"+strconv.Itoa(*portHTTP), redirectChain.Then(redirectTLSHandler()))

	// www.selfsignedcertificate.com
	log.Fatal(srv.ListenAndServeTLS("localhost.cert", "localhost.key"))
}
