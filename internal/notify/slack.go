package notify

import (
	log "log/slog"
	"os"

	"github.com/slack-go/slack"
)

// SlackClient holds the Slack client and the context used when sending notifications.
type SlackClient struct {
	Client      *slack.Client
	ChannelID   string
	Environment string
	ScaleAction string
}

// NewSlackClient returns SlackClient, which can be used for sending messages to Slack channels.
func NewSlackClient() *SlackClient {
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")
	slackChannelID := os.Getenv("SLACK_CHANNEL_ID")
	environment := os.Getenv("ENVIRONMENT")
	scaleAction := os.Getenv("SCALE_ACTION")

	if slackAPIToken != "" && slackChannelID != "" && environment != "" {
		return &SlackClient{
			Client:      slack.New(slackAPIToken),
			ChannelID:   slackChannelID,
			Environment: environment,
			ScaleAction: scaleAction,
		}
	}

	log.Warn("SLACK_API_TOKEN and/or SLACK_CHANNEL_ID and/or ENVIRONMENT envar(s) not set. Disabling Slack notifications")
	return nil
}

// PostMessage sends a formatted error notification to the configured Slack channel.
func PostMessage(slackClient *SlackClient, message string) error {
	attachment := slack.Attachment{
		Text: "Details",
		Fields: []slack.AttachmentField{
			{
				Title: "Environment",
				Value: slackClient.Environment,
			},
			{
				Title: "Scaling Type",
				Value: slackClient.ScaleAction,
			},
			{
				Title: "Error",
				Value: message,
			},
		},
	}

	_, _, err := slackClient.Client.PostMessage(
		slackClient.ChannelID,
		slack.MsgOptionText("A problem has occurred whilst scaling the environment cloud infrastructure", false),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		return err
	}

	return nil
}

// Slack sends msg to Slack if a SlackClient is configured, logging any send failure.
func Slack(slackClient *SlackClient, msg string) {
	if slackClient == nil {
		return
	}

	slackErr := PostMessage(slackClient, msg)
	if slackErr != nil {
		log.Error("sending Slack message", "error", slackErr)
	}
}
