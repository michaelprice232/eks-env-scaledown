package notify

import (
	log "log/slog"
	"os"

	"github.com/slack-go/slack"
)

type SlackClient struct {
	Client      *slack.Client
	ChannelID   string
	Environment string
	ScaleAction string
}

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
	} else {
		log.Warn("SLACK_API_TOKEN and/or SLACK_CHANNEL_ID and/or ENVIRONMENT envar(s) not set. Disabling Slack notifications")
		return nil
	}
}

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

func Slack(slackClient *SlackClient, msg string) {
	if slackClient != nil {
		slackErr := PostMessage(slackClient, msg)
		if slackErr != nil {
			log.Error("sending Slack message", "error", slackErr)
		}
	}
}
