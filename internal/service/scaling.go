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

const (
	timeout      = time.Minute * 15
	timeInterval = time.Second * 2
)

func (s *Service) scaleDownGroup(groupNumber int) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var resources []k8sResource
	var found bool
	if resources, found = s.startUpOrder[groupNumber]; !found {
		return fmt.Errorf("scaleDownGroup %d not found in the startUpOrder map", groupNumber)
	}

	for _, resource := range resources {
		if resource.ResourceType == "deployment" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
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
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(resource.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

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
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
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
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(resource.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

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

	if err := s.waitForPodTermination(resources); err != nil {
		return fmt.Errorf("waiting for pods to terminate: %w", err)
	}

	return nil
}

func (s *Service) waitForPodTermination(resources []k8sResource) error {
	interval := time.Tick(timeInterval)
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
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

func podsStillRunning(resources []k8sResource) bool {
	var runningPods bool
	for _, r := range resources {
		if r.podsTerminated == false {
			runningPods = true
		}
	}

	return runningPods
}

func int32Ptr(i int32) *int32 { return &i }

func (s *Service) scaleUpGroup(groupNumber int) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var resources []k8sResource
	var found bool
	if resources, found = s.startUpOrder[groupNumber]; !found {
		return fmt.Errorf("scaleUpGroup %d not found in the startUpOrder map", groupNumber)
	}

	for _, resource := range resources {
		if resource.ResourceType == "deployment" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := s.conf.K8sClient.AppsV1().Deployments(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("getting deployment %s: %w", result.Name, getErr)
				}

				replicasRaw, found := result.Annotations[OriginalReplicasAnnotationKey]
				if !found {
					log.Warn("NumReplicas Annotation key not set. The resource might have been created after the scaledown. Skipping", "key", OriginalReplicasAnnotationKey, "type", resource.ResourceType, "resource", result.Name, "Namespace", result.Namespace)
					return nil
				}

				replica64, err := strconv.ParseInt(replicasRaw, 10, 32)
				if err != nil {
					return fmt.Errorf("parsing an int from %s: %w", replicasRaw, err)
				}
				replicas := int32(replica64)

				result.Spec.Replicas = &replicas
				delete(result.Annotations, OriginalReplicasAnnotationKey)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().Deployments(resource.Namespace).Update(ctx, result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update deployment %s in Namespace %s: %w", resource.Name, resource.Namespace, retryErr)
			}
			log.Debug("Deployment scaled up", "deployment", resource.Name, "Namespace", resource.Namespace)
		}

		if resource.ResourceType == "statefulset" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := s.conf.K8sClient.AppsV1().StatefulSets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("getting statefulset %s: %w", result.Name, getErr)
				}

				replicasRaw, found := result.Annotations[OriginalReplicasAnnotationKey]
				if !found {
					log.Warn("NumReplicas Annotation key not set. The resource might have been created after the scaledown. Skipping", "key", OriginalReplicasAnnotationKey, "type", resource.ResourceType, "resource", result.Name, "Namespace", result.Namespace)
					return nil
				}

				replica64, err := strconv.ParseInt(replicasRaw, 10, 32)
				if err != nil {
					return fmt.Errorf("parsing an int from %s: %w", replicasRaw, err)
				}
				replicas := int32(replica64)

				result.Spec.Replicas = &replicas
				delete(result.Annotations, OriginalReplicasAnnotationKey)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().StatefulSets(resource.Namespace).Update(ctx, result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update %s %s in Namespace %s: %w", resource.ResourceType, resource.Name, resource.Namespace, retryErr)
			}
			log.Debug("Statefulset scaled up", "statefulset", resource.Name, "Namespace", resource.Namespace)
		}
	}

	if err := s.waitForPodsReady(resources); err != nil {
		return fmt.Errorf("waiting for pods to be ready: %w", err)
	}

	return nil
}

func (s *Service) waitForPodsReady(resources []k8sResource) error {
	interval := time.Tick(timeInterval)
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()

	for {
		select {
		case <-interval:
			if podsUpdatedAndReady(resources) {
				return nil
			}

			for i := range resources {
				// If we range over the slice, we are working with copies only, so can't update the podsTerminated correctly
				r := &resources[i]

				if r.ResourceType == "deployment" {
					log.Debug("Checking if pods are updated and ready", "type", r.ResourceType, "resource", r.Name, "Namespace", r.Namespace)
					deployment, err := s.conf.K8sClient.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
					if err != nil {
						return fmt.Errorf("getting deloyment %s in Namespace %s: %w", r.Name, r.Namespace, err)
					}
					if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
						deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
						deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
						deployment.Status.ObservedGeneration >= deployment.Generation {
						r.podsUpdatedAndReady = true
						log.Debug("Deployment ready", "type", r.ResourceType, "resource", r.Name, "Namespace", r.Namespace)
					}
					continue
				}
				if r.ResourceType == "statefulset" {
					log.Debug("Checking if pods are updated and ready", "type", r.ResourceType, "resource", r.Name, "Namespace", r.Namespace)
					statefulset, err := s.conf.K8sClient.AppsV1().StatefulSets(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
					if err != nil {
						return fmt.Errorf("getting statefulset %s in Namespace %s: %w", r.Name, r.Namespace, err)
					}
					if statefulset.Status.AvailableReplicas == *statefulset.Spec.Replicas &&
						statefulset.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
						statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas &&
						statefulset.Status.ObservedGeneration >= statefulset.Generation {
						r.podsUpdatedAndReady = true
						log.Debug("Statefulset ready", "type", r.ResourceType, "resource", r.Name, "Namespace", r.Namespace)
					}
					continue
				}

				return fmt.Errorf("expected 'deployment' or 'statefulset' type, but got '%s'", r.ResourceType)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func podsUpdatedAndReady(resources []k8sResource) bool {
	podsReady := true
	for _, r := range resources {
		if r.podsUpdatedAndReady == false {
			podsReady = false
		}
	}

	return podsReady
}

func (s *Service) terminateStandalonePods() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()

	namespaces, err := s.conf.K8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing namespaces: %w", err)
	}
	for _, ns := range namespaces.Items {
		pods, err := s.conf.K8sClient.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("listing pods in Namespace %s: %w", ns.Name, err)
		}

		for _, pod := range pods.Items {
			log.Info("Terminating remaining pod", "pod", pod.Name, "Namespace", pod.Namespace)
			if err = s.conf.K8sClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("deleting pod %s in Namespace %s: %w", pod.Name, pod.Namespace, err)
			}
		}
	}

	return nil
}
