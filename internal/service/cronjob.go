package service

import (
	"context"
	"fmt"
	log "log/slog"
	"time"

	"github.com/michaelprice232/eks-env-scaledown/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (s *Service) updateCronJobs() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()

	cjs, err := s.conf.K8sClient.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing CronJobs: %w", err)
	}

	for _, cj := range cjs.Items {
		retryErr := retry.RetryOnConflict(s.retryBackoff, func() error {
			result, getErr := s.conf.K8sClient.BatchV1().CronJobs(cj.Namespace).Get(ctx, cj.Name, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			if appNameLabel, found := result.Labels["app"]; found && appNameLabel == cronJobAppName {
				log.Debug("Skipping CronJob as it matches the app label which manages this app", "CronJob", cj.Name, "namespace", cj.Namespace)
				return nil
			}

			if result.Annotations == nil {
				result.Annotations = make(map[string]string)
			}

			// Do not enable anything that was previously suspended
			if s.conf.Action == config.ScaleUp {
				if value, found := result.Annotations[cronJobWasDisabledAnnotationKey]; found && value == "yes" {
					log.Warn("CronJob was previously disabled. Skipping", "CronJob", cj.Name, "namespace", cj.Namespace)
					return nil
				}
				result.Spec.Suspend = boolPtr(false)
			}

			if s.conf.Action == config.ScaleDown {
				if *result.Spec.Suspend == true {
					log.Warn("CronJob is already suspended. Setting annotation for scaleup run so it isn't enabled at scaleup", "CronJob", cj.Name, "namespace", cj.Namespace)
					result.Annotations[cronJobWasDisabledAnnotationKey] = "yes"
				}
				result.Spec.Suspend = boolPtr(true)
			}

			result.Annotations[updatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

			_, updateErr := s.conf.K8sClient.BatchV1().CronJobs(cj.Namespace).Update(ctx, result, metav1.UpdateOptions{})
			return updateErr
		})
		if retryErr != nil {
			return fmt.Errorf("updating CronJob %s in namespace %s: %w", cj.Name, cj.Namespace, err)
		}
	}

	return nil
}
