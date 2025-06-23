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

	nrClient, err := notify.NewNewRelicClient()
	if err != nil {
		reportError(slackClient, err, "creating New Relic client")
	}

	c, err := config.NewConfig()
	if err != nil {
		reportError(slackClient, err, "creating config")
	}

	if c.Action == config.ScaleDown {
		err = notify.UpdateCloudwatchAlarms("disable")
		if err != nil {
			reportError(slackClient, err, "disabling Cloudwatch alarms")
		}

		err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleDown)
		if err != nil {
			reportError(slackClient, err, "updating New Relic")
		}
	}

	s, err := service.NewService(c)
	if err != nil {
		reportError(slackClient, err, "creating service")
	}

	if err = s.Run(); err != nil {
		reportError(slackClient, err, "running")
	}

	if c.Action == config.ScaleUp {
		err = notify.UpdateCloudwatchAlarms("enable")
		if err != nil {
			reportError(slackClient, err, "enabling Cloudwatch alarms")
		}

		err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleUp)
		if err != nil {
			reportError(slackClient, err, "updating New Relic")
		}
	}
}

func reportError(slackClient *notify.SlackClient, err error, message string) {
	log.Error(message, "error", err)
	notify.Slack(slackClient, fmt.Sprintf("error whilst %s: %v", message, err))
	os.Exit(1)
}
