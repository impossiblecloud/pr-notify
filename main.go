package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/impossiblecloud/pr-notify/internal/cfg"
	"github.com/impossiblecloud/pr-notify/internal/metrics"
)

// Constants
const version = "0.0.1"

// Prometheus metrics handler
func handleMetrics(config cfg.AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		glog.V(10).Info("Got HTTP request for /metrics")

		promhttp.HandlerFor(prometheus.Gatherer(config.Metrics.Registry), promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

// Root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	glog.V(10).Info("Got HTTP request for /")

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Up and running. Version: %s", version)
}

// Health handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	glog.V(10).Info("Got HTTP request for /health")

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Health is OK")
}

// Main web server
func runMainWebServer(config cfg.AppConfig, listen string) {
	// Setup http router
	router := mux.NewRouter().StrictSlash(true)

	// Routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/metrics", handleMetrics(config)).Methods("GET")
	router.HandleFunc("/", rootHandler).Methods("GET")

	// Run main http router
	glog.Fatal(http.ListenAndServe(listen, router))
}

func main() {
	var listen string
	var showVersion bool

	// Init config
	config := cfg.AppConfig{}

	flag.StringVar(&listen, "listen", ":8765", "Address:port to listen on")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.Parse()

	// Show and exit functions
	if showVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	// Init metric
	config.Metrics = metrics.InitMetrics(version)

	glog.V(4).Infof("Starting application. Version: %s", version)
	runMainWebServer(config, listen)
}
