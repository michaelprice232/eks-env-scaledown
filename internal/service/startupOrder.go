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

func (s *Service) buildStartUpOrder() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	orders := make(startUpOrder)

	// Deployments
	deployments, err := s.conf.K8sClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s deployments: %w", err)
	}

	for _, d := range deployments.Items {
		selector, err := convertLabelSelectorToString(d.Spec.Selector)
		if err != nil {
			return err
		}

		res := k8sResource{
			Name:         d.Name,
			ResourceType: "deployment",
			Namespace:    d.Namespace,
			ReplicaCount: *d.Spec.Replicas,
			Selector:     selector,
		}

		if orderKey, found := d.Annotations[startupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				log.Warn("Unable to parse the int from the startup order key. Assigning to default group", "deployment", d.Name, "Namespace", d.Namespace, "originalOrder", orderKey, "key", startupOrderAnnotationKey)
				orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
				continue
			}

			if so < 0 || so >= defaultStartUpGroup {
				log.Warn("startUpOrder number can only be from 0 to 99. Assigning to default group", "deployment", d.Name, "Namespace", d.Namespace, "originalOrder", so, "defaultGroup", defaultStartUpGroup)
				orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
				continue
			}

			orders[so] = append(orders[so], res)
			continue
		}
		// Assign to the default group which starts up last if not set
		orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
	}

	// Statefulsets
	statefulset, err := s.conf.K8sClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing K8s statefulsets: %w", err)
	}
	for _, ss := range statefulset.Items {
		selector, err := convertLabelSelectorToString(ss.Spec.Selector)
		if err != nil {
			return err
		}

		res := k8sResource{
			Name:         ss.Name,
			ResourceType: "statefulset",
			Namespace:    ss.Namespace,
			ReplicaCount: *ss.Spec.Replicas,
			Selector:     selector,
		}

		if orderKey, found := ss.Annotations[startupOrderAnnotationKey]; found {
			so, err := strconv.Atoi(orderKey)
			if err != nil {
				log.Warn("Unable to parse the int from the startup order key. Assigning to default group", "statefulset", ss.Name, "Namespace", ss.Namespace, "originalOrder", orderKey, "key", startupOrderAnnotationKey)
				orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
				continue
			}

			if so < 0 || so >= defaultStartUpGroup {
				log.Warn("startUpOrder number can only be from 0 to 99. Assigning to default group", "statefulset", ss.Name, "Namespace", ss.Namespace, "originalOrder", so, "defaultGroup", defaultStartUpGroup)
				orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
				continue
			}

			orders[so] = append(orders[so], res)
			continue
		}
		// Assign to the default group which starts up last if not set
		orders[defaultStartUpGroup] = append(orders[defaultStartUpGroup], res)
	}

	log.Debug("Completed building startUpOrder", "orders", orders)
	s.startUpOrder = orders

	return nil
}
