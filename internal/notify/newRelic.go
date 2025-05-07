package notify

import (
	"fmt"
	log "log/slog"
	"os"
	"strconv"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/alerts"
)

type NewRelicClient struct {
	Client   *alerts.Alerts
	PolicyID int
}

func NewNewRelicClient() (*NewRelicClient, error) {
	policyID := os.Getenv("NEW_RELIC_ALERT_POLICY_ID")
	if policyID == "" {
		log.Warn("NEW_RELIC_ALERT_POLICY_ID envar not set. Will not disable any New Relic alert policies")
		return nil, nil
	}
	policyID64, err := strconv.ParseInt(policyID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse envar NEW_RELIC_ALERT_POLICY_ID %s into an int. Will not disable any New Relic alert policies", policyID)
	}

	apiKey := os.Getenv("NEW_RELIC_API_KEY")
	if apiKey == "" {
		log.Warn("NEW_RELIC_API_KEY not set. Will not disable any New Relic alert policies")
		return nil, nil
	}

	client, err := newrelic.New(newrelic.ConfigPersonalAPIKey(apiKey), newrelic.ConfigRegion("eu"))
	if err != nil {
		return nil, fmt.Errorf("creating New Relic client: %w", err)
	}

	return &NewRelicClient{
		Client:   &client.Alerts,
		PolicyID: int(policyID64),
	}, nil
}

type ScaleAction string

const (
	ScaleUp   ScaleAction = "ScaleUp"
	ScaleDown ScaleAction = "ScaleDown"
)

func UpdateNewRelicAlertPolicy(nrClient *NewRelicClient, action ScaleAction) error {
	alertConditions, err := nrClient.Client.ListNrqlConditions(nrClient.PolicyID)
	if err != nil {
		return fmt.Errorf("listing nrql conditions in policyID %d: %w", nrClient.PolicyID, err)
	}
	log.Debug("Found NRQL alert conditions", "policyID", nrClient.PolicyID, "count", len(alertConditions))

	for _, c := range alertConditions {
		switch action {
		case ScaleUp:
			log.Debug("Enabling alert condition", "name", c.Name)
			c.Enabled = true
		case ScaleDown:
			log.Debug("Suspending alert condition", "name", c.Name)
			c.Enabled = false
		default:
			return fmt.Errorf("action not recogised: %v", action)
		}

		_, err = nrClient.Client.UpdateNrqlCondition(*c)
		if err != nil {
			return fmt.Errorf("updating (%s) New Relic alert condition %d in policy %d: %w", action, c.ID, nrClient.PolicyID, err)
		}
	}

	return nil
}
