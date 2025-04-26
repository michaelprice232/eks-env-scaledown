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
	StartupOrderAnnotationKey         = "eks-env-scaledown/startup-order"
	OriginalReplicasAnnotationKey     = "eks-env-scaledown/original-replicas"
	UpdatedAtAnnotationKey            = "eks-env-scaledown/updated-at"
	DefaultStartUpGroup           int = 100
)

func (s *Service) BuildStartUpOrder() error {
	orders := make(StartUpOrder)

	// Deployments
	deployments, err := s.conf.K8sClient.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s deployments: %w", err)
	}
	for _, d := range deployments.Items {
		res := K8sResource{
			Name:         d.Name,
			Type:         "deployment",
			Namespace:    d.Namespace,
			ReplicaCount: *d.Spec.Replicas,
		}

		if orderKey, found := d.Annotations[StartupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				return fmt.Errorf("parsing int from string in startup order annotation '%s': %w", orderKey, err)
			}

			if so > 99 {
				log.Warn("StartUpOrder number can only be from 0 to 99. Assigning to default group", "deployment", d.Name, "namespace", d.Namespace, "Order", so)
				orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
				continue
			}

			orders[so] = append(orders[so], res)
			continue
		}
		// Assign to the default group which starts up last if not set
		orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
	}

	// Stateful-sets
	stateful, err := s.conf.K8sClient.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s statefulsets: %w", err)
	}
	for _, ss := range stateful.Items {
		res := K8sResource{
			Name:         ss.Name,
			Type:         "statefulset",
			Namespace:    ss.Namespace,
			ReplicaCount: *ss.Spec.Replicas,
		}

		if orderKey, found := ss.Annotations[StartupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				return fmt.Errorf("parsing int from string in startup order annotation '%s': %w", orderKey, err)
			}

			if so > 99 {
				log.Warn("StartUpOrder number can only be from 0 to 99. Assigning to default group", "statefulset", ss.Name, "namespace", ss.Namespace, "Order", so)
				orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
				continue
			}

			orders[so] = append(orders[so], res)
			continue
		}
		// Assign to the default group which starts up last if not set
		orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
	}

	log.Debug("Completed building StartUpOrder", "orders", orders)
	s.StartUpOrder = orders

	return nil
}

func (s *Service) ScaleDownGroup(groupNumber int) error {
	var groups []K8sResource
	var found bool
	if groups, found = s.StartUpOrder[groupNumber]; !found {
		return fmt.Errorf("ScaleDownGroup %d not found in the StartUpOrder map", groupNumber)
	}

	for _, group := range groups {
		if group.Type == "deployment" {
			// Use a retry function to handle conflicts on updates from concurrent changes
			// https://github.com/kubernetes/client-go/tree/master/examples/create-update-delete-deployment
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				result, getErr := s.conf.K8sClient.AppsV1().Deployments(group.Namespace).Get(context.TODO(), group.Name, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("failed to get %s %s in namespace %s: %w", group.Type, group.Name, group.Namespace, getErr)
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(group.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().Deployments(group.Namespace).Update(context.TODO(), result, metav1.UpdateOptions{})
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
				result, getErr := s.conf.K8sClient.AppsV1().StatefulSets(group.Namespace).Get(context.TODO(), group.Name, metav1.GetOptions{})
				if getErr != nil {
					return fmt.Errorf("failed to get %s %s in namespace %s: %w", group.Type, group.Name, group.Namespace, getErr)
				}

				if result.Annotations == nil {
					result.Annotations = make(map[string]string)
				}

				result.Spec.Replicas = int32Ptr(0)
				result.Annotations[OriginalReplicasAnnotationKey] = strconv.FormatInt(int64(group.ReplicaCount), 10)
				result.Annotations[UpdatedAtAnnotationKey] = time.Now().Format(time.RFC3339)

				// RetryOnConflict expects the error to be returned unwrapped
				// https://pkg.go.dev/k8s.io/client-go/util/retry@v0.33.0#RetryOnConflict
				_, updateErr := s.conf.K8sClient.AppsV1().StatefulSets(group.Namespace).Update(context.TODO(), result, metav1.UpdateOptions{})
				return updateErr
			})
			if retryErr != nil {
				return fmt.Errorf("failed to update %s %s in namespace %s: %w", group.Type, group.Name, group.Namespace, retryErr)
			}
			log.Debug("Statefulset scaled down", "statefulset", group.Name, "namespace", group.Namespace)
		}
	}

	// todo: wait for all the pods to be terminated in each group (with a deadline) before returning

	return nil
}

func int32Ptr(i int32) *int32 { return &i }
