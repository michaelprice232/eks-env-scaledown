package service

import (
	"context"
	"fmt"

	"github.com/michaelprice232/eks-env-scaledown/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	log "log/slog"
)

func (s *Service) updateKedaScaleObjects(sa config.ScaleAction) error {
	if sa != config.ScaleDown && sa != config.ScaleUp {
		return fmt.Errorf("invalid ScaleAction detected. Must be 'ScaleUp' or 'ScaleDown'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	gvr := schema.GroupVersionResource{
		Group:    "keda.sh",
		Version:  "v1alpha1",
		Resource: "scaledobjects",
	}

	scaledobjects, err := s.conf.K8sDynamicClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing ScaledObjects: %w", err)
	}

	for _, item := range scaledobjects.Items {
		name := item.GetName()
		namespace := item.GetNamespace()

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Get the latest version of the ScaledObject
			latest, getErr := s.conf.K8sDynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("failed to get latest version of ScaledObject %s/%s: %v", namespace, name, getErr)
			}

			annotations, found, err := unstructured.NestedStringMap(latest.Object, "metadata", "annotations")
			if err != nil {
				return fmt.Errorf("failed to get annotations for ScaledObject %s/%s: %v", namespace, name, err)
			}

			if sa == config.ScaleDown {
				if found {
					annotations[kedaPausedKey] = "true"
				} else {
					annotations = map[string]string{kedaPausedKey: "true"}
				}
			}

			if sa == config.ScaleUp && found {
				delete(annotations, kedaPausedKey)
			}

			err = unstructured.SetNestedStringMap(latest.Object, annotations, "metadata", "annotations")
			if err != nil {
				return fmt.Errorf("failed to set annotations for ScaledObject %s/%s: %v", namespace, name, err)
			}

			// Attempt to update and retry on conflict
			_, updateErr := s.conf.K8sDynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, latest, metav1.UpdateOptions{})
			return updateErr
		})
		if retryErr != nil {
			return fmt.Errorf("failed to update ScaledObject %s/%s: %v\n", namespace, name, retryErr)
		}

		log.Debug("Annotated pod", "namespace", namespace, "name", name)
	}

	return nil
}
