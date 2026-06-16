package service

import (
	"context"
	"testing"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func TestUpdateKedaScaleObjects_ScaleDown(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects"}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvr.GroupVersion().WithKind("ScaledObject"))
	obj.SetName("test-scaledobject")
	obj.SetNamespace("default")

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, obj)

	svc := &Service{
		conf: config.Config{
			K8sDynamicClient: fakeClient,
			SuspendKeda:      true,
		},
	}

	err := svc.updateKedaScaleObjects(config.ScaleDown)
	assert.NoError(t, err)

	// Check annotation was set during scale down
	res, err := fakeClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-scaledobject", metav1.GetOptions{})
	assert.NoError(t, err)
	annotations := res.GetAnnotations()
	assert.Equal(t, "true", annotations[kedaPausedKey], "expected keda.sh/paused annotation to be set to true")
}

func TestUpdateKedaScaleObjects_ScaleUp(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects"}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvr.GroupVersion().WithKind("ScaledObject"))
	obj.SetName("test-scaledobject")
	obj.SetNamespace("default")
	obj.SetAnnotations(map[string]string{kedaPausedKey: "true"})

	scheme := runtime.NewScheme()
	fakeClient := fake.NewSimpleDynamicClient(scheme, obj)

	svc := &Service{
		conf: config.Config{
			K8sDynamicClient: fakeClient,
			SuspendKeda:      true,
		},
	}

	err := svc.updateKedaScaleObjects(config.ScaleUp)
	assert.NoError(t, err)

	// Re-fetch the object to get the updated state
	res, err := fakeClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-scaledobject", metav1.GetOptions{})
	assert.NoError(t, err)
	annotations := res.GetAnnotations()
	_, found := annotations[kedaPausedKey]
	assert.False(t, found, "expected keda.sh/paused annotation to be removed")
}

func TestUpdateKedaScaleObjects_InvalidAction(t *testing.T) {
	svc := &Service{}
	err := svc.updateKedaScaleObjects("invalid")
	assert.Error(t, err, "expected error for invalid action")
}
