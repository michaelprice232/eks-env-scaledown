package service

import (
	"context"
	"fmt"
	log "log/slog"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (s *Service) ScaleDownGroup(groupNumber int) error {
	var resources []K8sResource
	var found bool
	if resources, found = s.StartUpOrder[groupNumber]; !found {
		return fmt.Errorf("ScaleDownGroup %d not found in the StartUpOrder map", groupNumber)
	}

	for _, group := range resources {
		if group.Type == "deployment" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := s.Conf.K8sClient.AppsV1().Deployments(group.Namespace).Get(context.TODO(), group.Name, metav1.GetOptions{})
				if getErr != nil {
					return getErr
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(group.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.Conf.K8sClient.AppsV1().Deployments(group.Namespace).Update(context.TODO(), result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update deployment %s in namespace %s: %w", group.Name, group.Namespace, retryErr)
			}
			log.Debug("Deployment scaled down", "deployment", group.Name, "namespace", group.Namespace)
		}

		if group.Type == "statefulset" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := s.Conf.K8sClient.AppsV1().StatefulSets(group.Namespace).Get(context.TODO(), group.Name, metav1.GetOptions{})
				if getErr != nil {
					return getErr
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(group.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.Conf.K8sClient.AppsV1().StatefulSets(group.Namespace).Update(context.TODO(), result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update %s %s in namespace %s: %w", group.Type, group.Name, group.Namespace, retryErr)
			}
			log.Debug("Statefulset scaled down", "statefulset", group.Name, "namespace", group.Namespace)
		}
	}

	if err := s.waitForPodTermination(resources); err != nil {
		return fmt.Errorf("waiting for pods to terminate: %w", err)
	}

	return nil
}

func (s *Service) waitForPodTermination(resources []K8sResource) error {
	interval := time.Tick(time.Second * 2)
	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancelCtx()

	for {
		select {
		case <-interval:
			if !podsStillRunning(resources) {
				return nil
			}

			for i := range resources {
				// If we range over the slice, we are working with copies only, so can't update the podsTerminated correctly
				r := &resources[i]

				log.Debug("Finding non-terminated pods", "type", r.Type, "resource", r.Name, "namespace", r.Namespace, "selector", r.Selector)

				pods, err := s.Conf.K8sClient.CoreV1().Pods(r.Namespace).List(ctx, metav1.ListOptions{LabelSelector: r.Selector})
				if err != nil {
					return fmt.Errorf("listing pods: %w", err)
				}

				if len(pods.Items) == 0 {
					log.Debug("Pods have been terminated", "resource", r.Name, "namespace", r.Namespace)
					r.podsTerminated = true
					continue
				}

				log.Debug("Pods still running", "resource", r.Name, "namespace", r.Namespace, "podCount", len(pods.Items))
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func podsStillRunning(resources []K8sResource) bool {
	var runningPods bool
	for _, r := range resources {
		if r.podsTerminated == false {
			runningPods = true
		}
	}

	return runningPods
}

func int32Ptr(i int32) *int32 { return &i }
