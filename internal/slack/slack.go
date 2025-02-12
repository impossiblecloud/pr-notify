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
	glog.V(8).Infof("Getting GitHub login for Slack user: %q", slackUserID)
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
	glog.V(8).Infof("Getting conversation members for channel: %q", channelID)

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
	glog.V(8).Infof("Building Slack to GitHub user map for channels: %v", channelIDs)
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
