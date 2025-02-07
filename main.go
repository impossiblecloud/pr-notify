package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Constants
const version = "0.0.1"

// Variables
var myMetrics appMetrics
var registry = prometheus.NewRegistry()

// Decorator for all endpoints which increases total HTTP requests metric
func endpointCounter(endpoint http.HandlerFunc, mymetrics appMetrics) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mymetrics.httpRequestsTotal.WithLabelValues(mymetrics.labelValues...).Inc()
		endpoint(w, r)
	})
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

// Prometheus metrics handler
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	glog.V(10).Info("Got HTTP request for /metrics")

	promhttp.HandlerFor(prometheus.Gatherer(registry), promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

// Main web server
func runMainWebServer(listen string) {
	// Setup http router
	router := mux.NewRouter().StrictSlash(true)

	// Routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/metrics", metricsHandler).Methods("GET")
	router.HandleFunc("/", endpointCounter(rootHandler, myMetrics)).Methods("GET")

	// Run main http router
	glog.Fatal(http.ListenAndServe(listen, router))
}

func main() {

	var listen string

	flag.StringVar(&listen, "listen", ":8765", "Address:port to listen on")
	flag.Parse()

	// Init metric
	myMetrics = initMetrics(registry, []string{}, []string{})
	registry.MustRegister()

	glog.V(4).Infof("Starting application. Version: %s", version)
	runMainWebServer(listen)
}
