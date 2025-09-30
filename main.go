package main

import (
	"context"
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
var Version string

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
	fmt.Fprintf(w, "Up and running.")
}

// Health handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	glog.V(10).Info("Got HTTP request for /health")

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Health is OK")
}

// Main web server
func runMainWebServer(config cfg.AppConfig, listen string) {
	glog.Infof("Starting main web server on %s", listen)

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
func prNotificationsCall(config *cfg.AppConfig, g *gh.Github, s *slack.Slack, prn cfg.PrNotification) {
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

	// Prepare direct messages map
	directMessages := make(map[string]string)

	// Loop over all discovered PRs and compile messages
	for _, pr := range prs {
		glog.V(10).Infof("Checking PR-%d: %s", *pr.Number, *pr.Title)
		additionalInfo := ""

		// Skip the PR is it does not match additional conditions
		if !g.MatchesConditions(pr, prn) {
			glog.V(10).Infof("PR-%d does not match conditions", *pr.Number)
			continue
		}

		// Add PR to the assignee DM queue
		if pr.Assignee != nil {
			additionalInfo += fmt.Sprintf(" (assignee: %s)", *pr.Assignee.Login)
			directMessages[*pr.Assignee.Login] += fmt.Sprintf("- You have a PR assigned: %s - %s\n", *pr.HTMLURL, *pr.Title)
			glog.V(8).Infof("Added PR %s for assignee %q", *pr.HTMLURL, *pr.Assignee.Login)
		}

		// Add PR to the main message
		message += fmt.Sprintf("- %s - %s%s\n", *pr.HTMLURL, *pr.Title, additionalInfo)
	}

	// Send individual messages to assignees if configured
	if prn.Notifications.Slack.NotifyAssignee {
		for ghLogin, dm := range directMessages {
			glog.V(8).Infof("Checking PR slack notifications created for %q", ghLogin)
			if slackUID, ok := config.GetSlackUID(ghLogin); ok {
				glog.V(6).Infof("Sending DM to %s (slack UID: %s)", ghLogin, slackUID)
				message := ""
				if prn.Notifications.Slack.Header != "" {
					message += prn.Notifications.Slack.Header + "\n"
				}
				if err := s.DM(slackUID, message+dm); err != nil {
					glog.Errorf("Failed to send DM to %s (slack UID: %s): %s", ghLogin, slackUID, err.Error())
				}
			} else {
				glog.Warning("No Slack UID mapping found for GitHub user: ", ghLogin)
			}
		}
	}

	// Send summary message to the specified channel
	if prn.Notifications.Slack.ChannelID != "" {
		glog.V(6).Infof("Sending message to slack channel %q", prn.Notifications.Slack.ChannelID)
		if err := s.SendMessage(prn, message); err != nil {
			glog.Errorf("Failed to send message to slack: %s", err.Error())
		}
	}
}

func main() {
	var listen, configFile, ghUser, slackUser, slackChannel string
	var showVersion, slackDebug, debugSlackToGithubUserMapping bool

	if Version == "" {
		Version = "unknown"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init config
	config := cfg.AppConfig{}

	flag.StringVar(&configFile, "config", "/etc/pr-notify.yaml", "Config file in YAML format")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&slackDebug, "slack-debug", false, "Slack API debug mode")
	flag.StringVar(&listen, "listen", ":8765", "Address:port to listen on")
	flag.StringVar(&ghUser, "debug-gh-user", "", "GitHub user login to debug and pull info about and exit")
	flag.StringVar(&slackUser, "debug-slack-user", "", "Slack user ID to debug and pull info about and exit")
	flag.StringVar(&slackChannel, "debug-slack-channel", "", "Debug Slack users in channel and exit")
	flag.BoolVar(&debugSlackToGithubUserMapping, "debug-slack-gh-users", false, "Debug Slack to GitHub user mapping and exit")
	flag.Parse()

	// Show and exit functions
	if showVersion {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}
	glog.V(4).Infof("Starting application. Version: %s", Version)

	err := config.LoadConfig(configFile)
	if err != nil {
		glog.Fatalf("Failed to load config file %q: %s", configFile, err.Error())
	}

	// Some quick debug info
	glog.V(6).Infof("Loaded PR notifications: %+v", config.PrNotifications)
	glog.V(6).Infof("Loaded Slack config: %+v", config.SlackConfig)

	// Init metric and cron
	config.Metrics = metrics.InitMetrics(Version)
	cronJob := cron.New()
	defer cronJob.Stop()

	// Init clients
	ghClient := gh.Github{}
	glog.Info("Initializing Github client")
	err = ghClient.Init()
	if err != nil {
		glog.Fatalf("Failed to initialize Github Client: %s", err.Error())
	}

	glog.Info("Initializing Slack client")
	slackClient := slack.Slack{}
	err = slackClient.Init(slackDebug)
	if err != nil {
		glog.Fatalf("Failed to initialize Slack Client: %s", err.Error())
	}

	// Log debug info for a specific GitHub user and exit
	if ghUser != "" {
		ghClient.LogUserInfo(ghUser)
		return
	}

	// Log debug info for a specific Slack user and exit
	if slackUser != "" {
		slackClient.LogUserInfo(slackUser)
		return
	}

	// Log debug info about slack to GH user mappings
	if debugSlackToGithubUserMapping {
		slackClient.LogGithubSlackUserMappings()
		return
	}

	// Log debug info about slack channel and exit
	if slackChannel != "" {
		slackClient.LogsUsersInChannel(slackChannel)
		return
	}

	// Add cron job schedulers for all PR notification configs
	glog.Info("Starting cron job schedulers")
	for id, prn := range config.PrNotifications {
		cronJob.AddFunc(prn.Schedule, func() { prNotificationsCall(&config, &ghClient, &slackClient, prn) })
		glog.Infof("Added cronjob scheduler %d for %s/%s", id, prn.Owner, prn.Repo)
	}
	cronJob.Start()

	// Start Slack to GitHub user mapping update loop
	go slackClient.SlackToGithubUpdateLoop(&config)

	// Start main web server in a separate goroutine
	go runMainWebServer(config, listen)

	// Start Slack channel auto-reply loop and run the Slack SocketMode client blocking call
	go slackClient.SlackSocketModeHandler(&config)
	if err := slackClient.Client.RunContext(ctx); err != nil {
		glog.Fatalf("Error running Slack socketmode: %v", err)
	}
}
