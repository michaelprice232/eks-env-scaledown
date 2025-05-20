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

func (s *Service) scaleDownGroup(groupNumber int) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var resources []*k8sResource
	var found bool
	if resources, found = s.startUpOrder[groupNumber]; !found {
		return fmt.Errorf("scaleDownGroup %d not found in the startUpOrder map", groupNumber)
	}

	for _, resource := range resources {
		if resource.ResourceType == "deployment" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(s.retryBackoff, func() error {
				result, getErr := s.conf.K8sClient.AppsV1().Deployments(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
				if getErr != nil {
					return getErr
				}

				if *result.Spec.Replicas == 0 {
					log.Warn("The workload has already been scaled to zero. Skipping", "type", resource.ResourceType, "resource", result.Name, "Namespace", result.Namespace)
					return nil
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[originalReplicasAnnotationKey] = strconv.FormatInt(int64(resource.ReplicaCount), 10)
				result.Annotations[updatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().Deployments(resource.Namespace).Update(ctx, result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update deployment %s in Namespace %s: %w", resource.Name, resource.Namespace, retryErr)
			}
			log.Debug("Deployment scaled down", "deployment", resource.Name, "Namespace", resource.Namespace)
		}

		if resource.ResourceType == "statefulset" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(s.retryBackoff, func() error {
				result, getErr := s.conf.K8sClient.AppsV1().StatefulSets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
				if getErr != nil {
					return getErr
				}

				if *result.Spec.Replicas == 0 {
					log.Warn("The workload has already been scaled to zero. Skipping", "type", resource.ResourceType, "resource", result.Name, "Namespace", result.Namespace)
					return nil
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[originalReplicasAnnotationKey] = strconv.FormatInt(int64(resource.ReplicaCount), 10)
				result.Annotations[updatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().StatefulSets(resource.Namespace).Update(ctx, result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update %s %s in Namespace %s: %w", resource.ResourceType, resource.Name, resource.Namespace, retryErr)
			}
			log.Debug("Statefulset scaled down", "statefulset", resource.Name, "Namespace", resource.Namespace)
		}
	}

	if !s.skipPodWait {
		if err := s.waitForPodTermination(resources); err != nil {
			return fmt.Errorf("waiting for pods to terminate: %w", err)
		}
	}

	return nil
}

func (s *Service) waitForPodTermination(resources []*k8sResource) error {
	interval := time.Tick(timeInterval)
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()

	for {
		select {
		case <-interval:
			if !podsStillRunning(resources) {
				return nil
			}

			for _, r := range resources {
				log.Debug("Finding non-terminated pods", "type", r.ResourceType, "resource", r.Name, "Namespace", r.Namespace, "selector", r.Selector)

				pods, err := s.conf.K8sClient.CoreV1().Pods(r.Namespace).List(ctx, metav1.ListOptions{LabelSelector: r.Selector})
				if err != nil {
					return fmt.Errorf("listing pods: %w", err)
				}

				if len(pods.Items) == 0 {
					log.Debug("Pods have been terminated", "resource", r.Name, "Namespace", r.Namespace)
					r.podsTerminated = true
					continue
				}

				log.Debug("Pods still running", "resource", r.Name, "Namespace", r.Namespace, "podCount", len(pods.Items))
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func podsStillRunning(resources []*k8sResource) bool {
	var runningPods bool
	for _, r := range resources {
		if r.podsTerminated == false {
			runningPods = true
		}
	}

	return runningPods
}

func (s *Service) terminateStandalonePods() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()

	pods, err := s.conf.K8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}

	for _, pod := range pods.Items {
		log.Debug("Terminating remaining pod", "pod", pod.Name, "Namespace", pod.Namespace)
		if err = s.conf.K8sClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("deleting pod %s in Namespace %s: %w", pod.Name, pod.Namespace, err)
		}
	}

	return nil
}
