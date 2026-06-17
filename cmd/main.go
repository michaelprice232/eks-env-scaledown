// Command eks-env-scaledown scales a Kubernetes environment up or down and
// manages the associated alerting integrations.
package main

import (
	"fmt"
	log "log/slog"
	"os"
	"time"

	"github.com/michaelprice232/eks-env-scaledown/config"
	"github.com/michaelprice232/eks-env-scaledown/internal/notify"
	"github.com/michaelprice232/eks-env-scaledown/internal/service"
)

func main() {
	config.SetupLogging()

	slackClient := notify.NewSlackClient()

	if err := run(); err != nil {
		reportError(slackClient, err)
	}
}

// run performs the full scale up/down workflow, returning a wrapped error on the
// first failure. Keeping the logic out of main() makes it testable and confines
// the os.Exit to a single place.
func run() error {
	nrClient, err := notify.NewNewRelicClient()
	if err != nil {
		return fmt.Errorf("creating New Relic client: %w", err)
	}

	c, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("creating config: %w", err)
	}

	if c.Action == config.ScaleDown {
		if err = notify.UpdateCloudwatchAlarms("disable"); err != nil {
			return fmt.Errorf("disabling Cloudwatch alarms: %w", err)
		}

		if err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleDown); err != nil {
			return fmt.Errorf("updating New Relic: %w", err)
		}
	}

	s, err := service.NewService(c)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	if err = s.Run(); err != nil {
		return fmt.Errorf("running: %w", err)
	}

	if c.Action == config.ScaleUp {
		// Delay re-enabling alerts to allow the services to stabilize first
		log.Info("Waiting for services to stabilize before enabling alerts", "delay", c.AlertStabilizationDelay)
		time.Sleep(c.AlertStabilizationDelay)

		if err = notify.UpdateCloudwatchAlarms("enable"); err != nil {
			return fmt.Errorf("enabling Cloudwatch alarms: %w", err)
		}

		if err = notify.UpdateNewRelicAlertPolicy(nrClient, notify.ScaleUp); err != nil {
			return fmt.Errorf("updating New Relic: %w", err)
		}
	}

	return nil
}

func reportError(slackClient *notify.SlackClient, err error) {
	log.Error("scaling the environment failed", "error", err)
	notify.Slack(slackClient, fmt.Sprintf("error whilst scaling the environment: %v", err))
	os.Exit(1)
}
