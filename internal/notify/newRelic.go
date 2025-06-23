package notify

import (
	"fmt"
	log "log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/alerts"
)

type NewRelicClient struct {
	Client    *alerts.Alerts
	PolicyIDs []int
}

// NewNewRelicClient returns NewRelicClient, which can be used for enabling and disabling New Relic alert policies.
func NewNewRelicClient() (*NewRelicClient, error) {
	apiKey := os.Getenv("NEW_RELIC_API_KEY")
	if apiKey == "" {
		log.Warn("NEW_RELIC_API_KEY not set. Will not disable any New Relic alert policies")
		return nil, nil
	}

	newRelicRegion := os.Getenv("NEW_RELIC_REGION")
	if newRelicRegion == "" {
		newRelicRegion = "eu"
	}

	ids := os.Getenv("NEW_RELIC_ALERT_POLICIES")
	if ids == "" {
		log.Warn("NEW_RELIC_ALERT_POLICIES envar not set. Will not disable any New Relic alert policies")
		return nil, nil
	}

	idsSplit := strings.Split(ids, ",")
	policyIds := make([]int, 0, len(idsSplit))

	for _, policy := range idsSplit {
		policy64, err := strconv.ParseInt(policy, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse New Relic alert policy ID %s into an int", policy)
		}
		policyIds = append(policyIds, int(policy64))
	}

	client, err := newrelic.New(newrelic.ConfigPersonalAPIKey(apiKey), newrelic.ConfigRegion(newRelicRegion))
	if err != nil {
		return nil, fmt.Errorf("creating New Relic client: %w", err)
	}

	return &NewRelicClient{
		Client:    &client.Alerts,
		PolicyIDs: policyIds,
	}, nil
}

type ScaleAction string

const (
	ScaleUp   ScaleAction = "ScaleUp"
	ScaleDown ScaleAction = "ScaleDown"
)

func (sa ScaleAction) validateAction() error {
	switch sa {
	case ScaleUp, ScaleDown:
		return nil
	default:
		return fmt.Errorf("invalid Action: must be 'ScaleUp' or 'ScaleDown'")
	}
}

// UpdateNewRelicAlertPolicy disables or enables New Relic alert policies.
func UpdateNewRelicAlertPolicy(nrClient *NewRelicClient, action ScaleAction) error {
	if nrClient == nil {
		return nil
	}

	if err := action.validateAction(); err != nil {
		return err
	}

	for _, policyID := range nrClient.PolicyIDs {
		if action == ScaleUp {
			log.Info("Enabling New Relic alert policy", "action", action, "policyID", policyID)
		}
		if action == ScaleDown {
			log.Info("Suspending New Relic alert policy", "action", action, "policyID", policyID)
		}

		alertConditions, err := nrClient.Client.ListNrqlConditions(policyID)
		if err != nil {
			return fmt.Errorf("listing nrql conditions in policyID %d: %w", policyID, err)
		}
		log.Debug("Found NRQL alert conditions", "policyID", policyID, "count", len(alertConditions))

		for _, c := range alertConditions {
			switch action {
			case ScaleUp:
				log.Debug("Enabling alert condition", "name", c.Name)
				c.Enabled = true
			case ScaleDown:
				log.Debug("Suspending alert condition", "name", c.Name)
				c.Enabled = false
			}

			_, err = nrClient.Client.UpdateNrqlCondition(*c)
			if err != nil {
				return fmt.Errorf("updating (%s) New Relic alert condition %d in policy %d: %w", action, c.ID, policyID, err)
			}
		}
	}

	return nil
}
