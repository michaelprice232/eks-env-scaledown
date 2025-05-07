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
		log.Error("creating New Relic client", "error", err)
		notify.Slack(slackClient, fmt.Sprintf("error whilst creating New Relic client: %v", err))
		os.Exit(1)
	}

	c, err := config.NewConfig()
	if err != nil {
		log.Error("creating config", "error", err)
		notify.Slack(slackClient, fmt.Sprintf("error whilst creating config: %v", err))
		os.Exit(2)
	}

	// Disable the New Relic alert policy if config has been set
	if nrClient != nil && c.Action == config.ScaleDown {
		err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleDown)
		if err != nil {
			log.Error("updating New Relic", "error", err)
			notify.Slack(slackClient, fmt.Sprintf("error whilst updating New Relic: %v", err))
			os.Exit(3)
		}
	}

	s, err := service.NewService(c)
	if err != nil {
		log.Error("creating service", "error", err)
		notify.Slack(slackClient, fmt.Sprintf("error whilst creating service: %v", err))
		os.Exit(4)
	}

	if err = s.Run(); err != nil {
		log.Error("running", "error", err)
		notify.Slack(slackClient, fmt.Sprintf("error whilst running: %v", err))
		os.Exit(5)
	}

	// Re-enable the New Relic alert policy if config has been set
	if nrClient != nil && c.Action == config.ScaleUp {
		err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleUp)
		if err != nil {
			log.Error("updating New Relic", "error", err)
			notify.Slack(slackClient, fmt.Sprintf("error whilst updating New Relic: %v", err))
			os.Exit(6)
		}
	}
}
