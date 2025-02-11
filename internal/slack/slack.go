package slack

import (
	"fmt"
	"log"
	"os"
	"strings"

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
	_, _, err := s.Client.PostMessage(prn.Notifications.Slack.ChannelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
		slack.MsgOptionLinkNames(true),
	)
	return err
}
