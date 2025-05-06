package main

import (
	"fmt"
	log "log/slog"
	"os"

	"github.com/michaelprice232/eks-env-scaledown/config"
	"github.com/michaelprice232/eks-env-scaledown/internal/notify"
	"github.com/michaelprice232/eks-env-scaledown/internal/service"
)

func main() {
	config.SetupLogging()

	slackClient := notify.NewSlackClient()

	c, err := config.NewConfig()
	if err != nil {
		log.Error("creating config", "error", err)

		if slackClient != nil {
			slackErr := notify.PostMessage(slackClient, fmt.Sprintf("error whilst creating config: %v", err))
			if slackErr != nil {
				log.Error("sending Slack message", "error", slackErr)
			}
		}

		os.Exit(1)
	}

	s, err := service.NewService(c)
	if err != nil {
		log.Error("creating service", "error", err)

		if slackClient != nil {
			slackErr := notify.PostMessage(slackClient, fmt.Sprintf("error whilst creating service: %v", err))
			if slackErr != nil {
				log.Error("sending Slack message", "error", slackErr)
			}

			os.Exit(2)
		}
	}

	if err = s.Run(); err != nil {
		log.Error("running", "error", err)

		if slackClient != nil {
			slackErr := notify.PostMessage(slackClient, fmt.Sprintf("error whilst running: %v", err))
			if slackErr != nil {
				log.Error("sending Slack message", "error", slackErr)
			}

			os.Exit(3)
		}
	}
}
