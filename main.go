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
	"github.com/impossiblecloud/pr-notify/internal/gh"
	"github.com/impossiblecloud/pr-notify/internal/metrics"
	"github.com/impossiblecloud/pr-notify/internal/slack"
	"github.com/robfig/cron/v3"
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

// prNotificationsCall is basically our main loop call
func prNotificationsCall(g *gh.Github, s *slack.Slack, prn cfg.PrNotification) {
	glog.V(8).Infof("PR notification call start for %s/%s", prn.Owner, prn.Repo)

	prs, err := g.GetPullRequests(prn)
	if err != nil {
		glog.Fatalf("Failed to pull PRs: %s", err.Error())
	}

	if len(prs) < 1 {
		glog.V(6).Infof("No PRs found for repo %s/%s", prn.Owner, prn.Repo)
		return
	}

	message := ""
	if prn.Notifications.Slack.Header != "" {
		message += prn.Notifications.Slack.Header + "\n"
	}
	message += fmt.Sprintf("Pull requests from %s/%s repository:\n", prn.Owner, prn.Repo)

	for _, pr := range prs {
		glog.Infof("PR-%d: %s", *pr.Number, *pr.Title)
		message += fmt.Sprintf("- %s - %s\n", *pr.HTMLURL, *pr.Title)
	}

	if err := s.SendMessage(prn, message); err != nil {
		glog.Errorf("Failed to send message to slack: %s", err.Error())
	}
}

func main() {
	var listen, configFile string
	var showVersion, slackDebug bool

	// Init config
	config := cfg.AppConfig{}

	flag.StringVar(&configFile, "config", "/etc/pr-notify.yaml", "Config file in YAML format")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&slackDebug, "slack-debug", false, "Slack API debug mode")
	flag.StringVar(&listen, "listen", ":8765", "Address:port to listen on")
	flag.Parse()

	// Show and exit functions
	if showVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}
	glog.V(4).Infof("Starting application. Version: %s", version)

	err := config.LoadConfig(configFile)
	if err != nil {
		glog.Fatalf("Failed to load config file %q: %s", configFile, err.Error())
	}
	glog.V(6).Infof("Loaded PR notifications: %+v", config.PrNotifications)

	// Init metric and cron
	config.Metrics = metrics.InitMetrics(version)
	cronJob := cron.New()
	defer cronJob.Stop()

	// Init clients
	ghClient := gh.Github{}
	err = ghClient.Init()
	if err != nil {
		glog.Fatalf("Failed to initialize Github Client: %s", err.Error())
	}
	slackClient := slack.Slack{}
	slackClient.Init(slackDebug)

	// Add cron job schedulers for all PR notification configs
	for _, prn := range config.PrNotifications {
		cronJob.AddFunc(prn.Schedule, func() { prNotificationsCall(&ghClient, &slackClient, prn) })
		glog.V(4).Infof("Added cronjob scheduler for %s/%s", prn.Owner, prn.Repo)
	}

	cronJob.Start()
	runMainWebServer(config, listen)
}
