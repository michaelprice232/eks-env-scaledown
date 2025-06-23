package notify

import (
	"context"
	"fmt"
	log "log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// UpdateCloudwatchAlarms either enables or disables all the actions for all the Cloudwatch alerts in the target AWS account.
// This includes both metric and composite alarms.
func UpdateCloudwatchAlarms(action string) error {
	if action != "enable" && action != "disable" {
		return fmt.Errorf("invalid action: must be 'enable' or 'disable'")
	}

	manageCloudwatchAlarms := os.Getenv("MANAGE_CLOUDWATCH_ALARMS")
	if manageCloudwatchAlarms == "" {
		log.Warn("MANAGE_CLOUDWATCH_ALARMS envar not set. Alarms will not be disable/or enabled")
		return nil
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("loading aws config: %w", err)
	}

	cwClient := cloudwatch.NewFromConfig(cfg)

	// Only metric alarms are returned by default
	alarmsPaginator := cloudwatch.NewDescribeAlarmsPaginator(cwClient, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{
			types.AlarmTypeCompositeAlarm,
			types.AlarmTypeMetricAlarm,
		},
	})

	for alarmsPaginator.HasMorePages() {
		alarmResults, err := alarmsPaginator.NextPage(context.Background())
		if err != nil {
			return fmt.Errorf("describing cloudwatch alarms: %w", err)
		}

		alarms := make([]string, 0, len(alarmResults.MetricAlarms)+len(alarmResults.CompositeAlarms))

		for _, metricAlarm := range alarmResults.MetricAlarms {
			alarms = append(alarms, *metricAlarm.AlarmName)
		}
		for _, compositeAlarm := range alarmResults.CompositeAlarms {
			alarms = append(alarms, *compositeAlarm.AlarmName)
		}

		log.Debug("Found alarms", "alarms", alarms)

		if action == "disable" {
			if _, err = cwClient.DisableAlarmActions(context.Background(), &cloudwatch.DisableAlarmActionsInput{AlarmNames: alarms}); err != nil {
				return fmt.Errorf("disabling alarm actions: %w", err)
			}
			log.Info("Disabled Cloudwatch alarms")
		} else {
			if _, err = cwClient.EnableAlarmActions(context.Background(), &cloudwatch.EnableAlarmActionsInput{AlarmNames: alarms}); err != nil {
				return fmt.Errorf("enabling alarm actions: %w", err)
			}
			log.Info("Enabled Cloudwatch alarms")
		}
	}

	return nil
}
