package service

import (
	"context"
	"fmt"
	log "log/slog"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func convertLabelSelectorToString(ls *metav1.LabelSelector) (string, error) {
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return "", fmt.Errorf("convert label selector to a string format: %w", err)
	}

	return selector.String(), nil
}

func (s *Service) BuildStartUpOrder() error {
	orders := make(StartUpOrder)

	// Deployments
	deployments, err := s.Conf.K8sClient.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s deployments: %w", err)
	}

	for _, d := range deployments.Items {
		selector, err := convertLabelSelectorToString(d.Spec.Selector)
		if err != nil {
			return fmt.Errorf("waiting for pod termination: %w", err)
		}

		res := K8sResource{
			Name:         d.Name,
			Type:         "deployment",
			Namespace:    d.Namespace,
			ReplicaCount: *d.Spec.Replicas,
			Selector:     selector,
		}

		if orderKey, found := d.Annotations[StartupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				return fmt.Errorf("parsing int from string in startup order annotation '%s': %w", orderKey, err)
			}

			if so > 99 {
				log.Warn("StartUpOrder number can only be from 0 to 99. Assigning to default group", "deployment", d.Name, "namespace", d.Namespace, "originalOrder", so)
				orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
				continue
			}

			orders[so] = append(orders[so], res)
			continue
		}
		// Assign to the default group which starts up last if not set
		orders[DefaultStartUpGroup] = append(orders[DefaultStartUpGroup], res)
	}

	// Statefulsets
	statefulset, err := s.Conf.K8sClient.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s statefulsets: %w", err)
	}
	for _, ss := range statefulset.Items {
		selector, err := convertLabelSelectorToString(ss.Spec.Selector)
		if err != nil {
			return fmt.Errorf("waiting for pod termination: %w", err)
		}

		res := K8sResource{
			Name:         ss.Name,
			Type:         "statefulset",
			Namespace:    ss.Namespace,
			ReplicaCount: *ss.Spec.Replicas,
			Selector:     selector,
		}

		if orderKey, found := ss.Annotations[StartupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				return fmt.Errorf("parsing int from string in startup order annotation '%s': %w", orderKey, err)
			}

			if so > 99 {
				log.Warn("StartUpOrder number can only be from 0 to 99. Assigning to default group", "statefulset", ss.Name, "namespace", ss.Namespace, "originalOrder", so)
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
