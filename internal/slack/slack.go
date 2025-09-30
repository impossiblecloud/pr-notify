package slack

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/impossiblecloud/pr-notify/internal/cfg"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Slack struct {
	Client *socketmode.Client
}

// Init initializes Slack client
func (s *Slack) Init(debug bool) error {
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		return fmt.Errorf("SLACK_APP_TOKEN must be set")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		return fmt.Errorf("SLACK_APP_TOKEN must have the prefix \"xapp-\"")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("SLACK_BOT_TOKEN must be set")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		return fmt.Errorf("SLACK_BOT_TOKEN must have the prefix \"xoxb-\"")
	}

	api := slack.New(
		botToken,
		slack.OptionDebug(debug),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	s.Client = socketmode.New(
		api,
		socketmode.OptionDebug(debug),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	info, err := api.AuthTest()
	if err != nil {
		glog.Fatalf("Got error while running api.AuthTest(): %s", err.Error())
	}

	if debug {
		glog.Infof("Slack Debug: %+v", info)
	}

	return nil
}

// SendMessage sends a slack message based on PR notification config
func (s *Slack) SendMessage(prn cfg.PrNotification, message string) error {
	glog.V(8).Infof("SLACK Sending message to slack channel %q", prn.Notifications.Slack.ChannelID)
	_, _, err := s.Client.PostMessage(prn.Notifications.Slack.ChannelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionLinkNames(true),
	)
	return err
}

// DM sends a direct message to a user
func (s *Slack) DM(userID string, message string) error {
	glog.V(8).Infof("SLACK Sending DM to slack user %q", userID)
	_, _, err := s.Client.PostMessage(userID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionLinkNames(true),
	)
	return err
}

// MakeGithubToSlackUserMap builds a map of ALL slack users with their github logins
// based on the info found in Slack custom profile fields
func (s *Slack) MakeGithubToSlackUserMap() (map[string]string, error) {
	glog.V(8).Infof("Building GitHub to Slack user map for all users in Slack organization")
	userMap := make(map[string]string)
	users, err := s.Client.GetUsers()
	if err != nil {
		glog.Errorf("Failed to get users: %s", err.Error())
		return nil, err
	}
	for _, user := range users {
		if user.IsBot || user.Deleted {
			glog.V(10).Infof("Skipping bot or deleted user: %q", user.ID)
			continue
		}
		glog.V(10).Infof("User custom fields info for %q: %+v", user.ID, user.Profile.Fields)
		githubLogin, err := s.GetUserGithubLogin(user.ID)
		if err != nil {
			glog.Errorf("Failed to get GitHub login for user %q: %s", user.ID, err.Error())
			continue
		}
		if githubLogin != "" {
			userMap[user.ID] = githubLogin
		}
	}
	return userMap, nil
}

// GetUserGithubLogin returns a github login for a Slack user based on custom fields
func (s *Slack) GetUserGithubLogin(slackUserID string) (string, error) {
	glog.V(9).Infof("Getting GitHub login for Slack user: %q", slackUserID)
	profile, err := s.Client.GetUserProfile(&slack.GetUserProfileParameters{
		UserID: slackUserID,
	})
	if err != nil {
		glog.Errorf("Failed to get user profile for %q: %s", slackUserID, err.Error())
		return "", err
	}
	for k, v := range profile.Fields.ToMap() {
		glog.V(10).Infof("User profile field for %q: %q: %+v", slackUserID, k, v)
		if v.Value != "" && strings.Contains(strings.ToLower(v.Value), "github") {
			return v.Alt, nil
		}
	}
	return "", nil
}

// GetConversationMembers returns a list of slack users in a channel
func (s *Slack) GetConversationMembers(channelID string) ([]string, error) {
	glog.V(10).Infof("Getting conversation members for channel: %q", channelID)

	users, _, err := s.Client.GetUsersInConversation(&slack.GetUsersInConversationParameters{
		ChannelID: channelID,
	})
	if err != nil {
		glog.Errorf("Failed to get conversation members for channel %q: %s", channelID, err.Error())
		return nil, err
	}

	return users, nil
}

// MakeGithubtoSlackUserMapInChannels builds a map of slack users found in the channel
// with their github logins based on the info found in Slack custom profile fields
func (s *Slack) MakeGithubToSlackUserMapInChannels(channelIDs []string) (map[string]string, error) {
	glog.V(10).Infof("Building Slack to GitHub user map for channels: %v", channelIDs)
	userMap := make(map[string]string)
	users := []string{}
	for _, chID := range channelIDs {
		channelUsers, err := s.GetConversationMembers(chID)
		if err != nil {
			glog.Errorf("Failed to get conversation members for channel %q: %s", chID, err.Error())
			return nil, err
		}
		users = append(users, channelUsers...)
	}

	for _, user := range users {
		if _, exists := userMap[user]; exists {
			continue
		}
		githubLogin, err := s.GetUserGithubLogin(user)
		if err != nil {
			glog.Errorf("Failed to get GitHub login for user %q: %s", user, err.Error())
			continue
		}
		if githubLogin != "" {
			userMap[githubLogin] = user
		}
	}
	return userMap, nil
}

// SlackToGithubUpdateLoop runs a loop forever and updates the Slack to GitHub user mapping
func (s *Slack) SlackToGithubUpdateLoop(conf *cfg.AppConfig) {
	channels := []string{}
	for _, slackChannel := range conf.SlackConfig.GithubUsersChannels {
		channels = append(channels, slackChannel.ID)
	}

	for {
		glog.V(8).Infof("Updating Slack to GitHub user mapping for channels: %v", channels)
		userMap, err := s.MakeGithubToSlackUserMapInChannels(channels)
		if err != nil {
			glog.Errorf("Failed to update Slack to GitHub user mapping: %s", err.Error())
		} else {
			glog.V(8).Infof("Updated Slack to GitHub user mapping: %+v", userMap)
		}
		conf.SlackToGithubUserMap = userMap
		time.Sleep(time.Duration(conf.SlackConfig.SlackUsersPullIntervalSeconds) * time.Second)
	}
}

// SlackSocketModeHandler handles Slack events in Socket Mode
func (s *Slack) SlackSocketModeHandler(conf *cfg.AppConfig) {
	glog.Infof("SlackSocketModeHandler started")
	defer glog.Infof("Exiting SlackSocketModeHandler")

	for evt := range s.Client.Events {
		glog.V(12).Infof("Received Events API event type: %+v", evt.Type)
		switch evt.Type {
		case socketmode.EventTypeConnecting:
			glog.Info("Connecting to Slack with Socket Mode...")
		case socketmode.EventTypeConnectionError:
			glog.Error("Connection failed. Retrying later...")
		case socketmode.EventTypeConnected:
			glog.Info("Connected to Slack with Socket Mode.")
		case socketmode.EventTypeEventsAPI:
			glog.V(10).Infof("Received Events API event type: %+v", evt.Type)
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				glog.Errorf("Could not type cast the event to the EventsAPIEvent: %v\n", evt)
				continue
			}
			s.Client.Ack(*evt.Request)
			switch eventsAPIEvent.Type {
			case slackevents.CallbackEvent:
				innerEvent := eventsAPIEvent.InnerEvent
				switch ev := innerEvent.Data.(type) {
				case *slackevents.MessageEvent:
					glog.V(10).Infof("Received message event: %+v", ev)
					if ev.BotID != "" {
						glog.V(10).Infof("Skipping bot message: %+v", ev)
						continue
					}

					// Skip messages that are in Slack threads
					if ev.ThreadTimeStamp != "" {
						glog.V(10).Infof("Skipping message in thread: %+v", ev)
						continue
					}

					// Find response for the channel and message
					response, replyInThread := conf.GetSlackMessageReply(ev.Channel, ev.Text)
					if response == "" {
						glog.V(10).Infof("No response found for channel %q and message", ev.Channel)
						continue
					}
					glog.V(10).Infof("Found response for channel %q and message: %q", ev.Channel, response)

					// Check cache
					if githubLogin, exists := conf.GetGithubLogin(ev.User); exists && githubLogin != "" {
						glog.V(8).Infof("User %q from channel %q has GitHub login %q in profile, not replying", ev.User, ev.Channel, githubLogin)
						continue
					}

					// Fetch user profile and check again
					githubLogin, err := s.GetUserGithubLogin(ev.User)
					if err != nil {
						glog.Errorf("Failed to get GitHub login for user %q: %s", ev.User, err.Error())
						continue
					}

					// No github login found, reply to the user
					if githubLogin == "" {
						glog.V(8).Infof("User %q from channel %q does not have GitHub login in profile, replying...", ev.User, ev.Channel)
						replyOptions := []slack.MsgOption{
							slack.MsgOptionText(response, false),
							slack.MsgOptionAsUser(true),
							slack.MsgOptionLinkNames(true),
						}
						if replyInThread {
							replyOptions = append(replyOptions, slack.MsgOptionTS(ev.TimeStamp))
						}
						// Send reply in Slack
						glog.V(10).Infof("SLACK Replying in channel %q to user %q with text %q", ev.Channel, ev.User, response)
						_, _, err := s.Client.PostMessage(ev.Channel, replyOptions...)
						if err != nil {
							glog.Errorf("Failed to post message to channel %q: %s", ev.Channel, err.Error())
							continue
						}
					} else {
						// Add user to cache
						conf.SlackToGithubUserMap[ev.User] = githubLogin
						glog.V(8).Infof("User %q from channel %q has GitHub login %q in profile, not replying", ev.User, ev.Channel, githubLogin)
					}
				}
			default:
				glog.V(12).Infof("Unsupported inner event type: %T", eventsAPIEvent.InnerEvent.Data)
			}
		default:
			glog.V(12).Infof("Unsupported Events API event type: %v", evt.Type)
		}
	}
}

// ------------------------ DEBUG STUFF BELOW ------------------------

// LogUserInfo logs info about a Slack user by their user ID
func (s *Slack) LogUserInfo(slackUserID string) {
	glog.Infof("Logging info for Slack user: %q", slackUserID)
	user, err := s.Client.GetUserInfo(slackUserID)
	if err != nil {
		glog.Errorf("Failed to get user info for %q: %s", slackUserID, err.Error())
		return
	}
	glog.Infof("User info for %q: %+v", slackUserID, user)
	profile, err := s.Client.GetUserProfile(&slack.GetUserProfileParameters{
		UserID: slackUserID,
	})
	if err != nil {
		glog.Errorf("Failed to get user profile for %q: %s", slackUserID, err.Error())
		return
	}
	glog.Infof("User profile for %q: %+v", slackUserID, profile)
	for k, v := range profile.Fields.ToMap() {
		glog.Infof("User profile field for %q: %q: %+v", slackUserID, k, v)
	}

	githubLogin, err := s.GetUserGithubLogin(slackUserID)
	if err != nil {
		glog.Errorf("Failed to get GitHub login for user %q: %s", slackUserID, err.Error())
		return
	}
	glog.Infof("GitHub login for user %q: %q", slackUserID, githubLogin)
}

// LogGithubSlackUserMappings logs mappings of slack to GH for debug
func (s *Slack) LogGithubSlackUserMappings() {
	userMap, err := s.MakeGithubToSlackUserMap()
	if err != nil {
		glog.Errorf("Failed to make Slack to GitHub user map: %s", err.Error())
		return
	}
	for slackID, ghLogin := range userMap {
		glog.Infof("Slack user %q is mapped to GitHub user %q", slackID, ghLogin)
	}
}

// LogsUsersInChannel logs users in the channel
func (s *Slack) LogsUsersInChannel(channelID string) {
	users, err := s.GetConversationMembers(channelID)
	if err != nil {
		glog.Errorf("Failed to get conversation members for channel %q: %s", channelID, err.Error())
		return
	}
	glog.Infof("Users in channel %q: %v", channelID, users)

	userMap, err := s.MakeGithubToSlackUserMapInChannels([]string{channelID})
	if err != nil {
		glog.Errorf("Failed to make Slack to GitHub user map for channel %q: %s", channelID, err.Error())
		return
	}
	for ghLogin, slackID := range userMap {
		glog.Infof("GitHub user %q is mapped to Slack user %q", ghLogin, slackID)
	}
}
