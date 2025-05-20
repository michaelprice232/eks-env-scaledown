package service

import (
	"context"
	"errors"
	"io"
	log "log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func Test_scaleUpGroup_deployments(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name                  string
		wantErr               bool
		resourceName          string
		resourceNamespace     string
		group                 int
		replicaCount          int32
		replicasAnnotationKey string
	}{
		{name: "normal scaleup as annotation is set", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 3, replicasAnnotationKey: "3"},
		{name: "annotation not set: skip", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 0},
		{name: "group not present", wantErr: true},
		{name: "K8s Server Error", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var annotations map[string]string
			if tc.replicasAnnotationKey != "" {
				annotations = map[string]string{
					originalReplicasAnnotationKey: tc.replicasAnnotationKey,
				}
			}

			s := &Service{
				startUpOrder: startUpOrder{
					tc.group: []*k8sResource{
						{Name: tc.resourceName, Namespace: tc.resourceNamespace, ResourceType: "deployment", ReplicaCount: tc.replicaCount},
					},
				},

				conf: config.Config{
					K8sClient: fake.NewClientset(
						&appsv1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:        tc.resourceName,
								Namespace:   tc.resourceNamespace,
								Annotations: annotations,
							},
							Spec: appsv1.DeploymentSpec{
								Replicas: int32Ptr(tc.replicaCount),
							},
						},
					),
				},

				// Reduce backoff and retries for simulated failures in unit tests
				retryBackoff: wait.Backoff{
					Duration: 10 * time.Millisecond,
					Factor:   1.0,
					Jitter:   0.1,
					Steps:    1,
				},

				// Do not wait for pods to be ready during tests
				skipPodWait: true,
			}

			// Simulate server side error
			if tc.name == "K8s Server Error" {
				k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
				k8sFakeClient.Fake.PrependReactor("get", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("server side error")
				})
			}

			if tc.name == "group not present" {
				s.startUpOrder = nil
			}

			err := s.scaleUpGroup(tc.group)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				result, err := s.conf.K8sClient.AppsV1().Deployments(tc.resourceNamespace).Get(context.Background(), tc.resourceName, metav1.GetOptions{})
				require.NoError(t, err)

				if tc.replicasAnnotationKey == "" {
					assert.NotContains(t, result.Annotations, updatedAtAnnotationKey, "Expected the resource to be skipped as the original replicas annotation key was not set")
				} else {
					assert.Contains(t, result.Annotations, updatedAtAnnotationKey, "Expected the resource to have been updated")
					assert.NotContains(t, result.Annotations, originalReplicasAnnotationKey, "Expected the original replicas annotation to be removed following a successful update")

					var value string
					var found bool
					if value, found = result.Annotations[updatedAtAnnotationKey]; found {
						parsedTime, err := time.Parse(time.RFC3339, value)
						assert.NoError(t, err)
						assert.NotNil(t, parsedTime)
					}

					replica64, err := strconv.ParseInt(tc.replicasAnnotationKey, 10, 32)
					require.NoError(t, err)
					assert.Equal(t, int32(replica64), *result.Spec.Replicas, "Expected the number of replicas to be set based on the the original replicas annotation key")
				}
			}
		})
	}
}

func Test_scaleUpGroup_statefulsets(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name                  string
		wantErr               bool
		resourceName          string
		resourceNamespace     string
		group                 int
		replicaCount          int32
		replicasAnnotationKey string
	}{
		{name: "normal scaleup as annotation is set", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 3, replicasAnnotationKey: "3"},
		{name: "annotation not set: skip", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 0},
		{name: "group not present", wantErr: true},
		{name: "K8s Server Error", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var annotations map[string]string
			if tc.replicasAnnotationKey != "" {
				annotations = map[string]string{
					originalReplicasAnnotationKey: tc.replicasAnnotationKey,
				}
			}

			s := &Service{
				startUpOrder: startUpOrder{
					tc.group: []*k8sResource{
						{Name: tc.resourceName, Namespace: tc.resourceNamespace, ResourceType: "statefulset", ReplicaCount: tc.replicaCount},
					},
				},

				conf: config.Config{
					K8sClient: fake.NewClientset(
						&appsv1.StatefulSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:        tc.resourceName,
								Namespace:   tc.resourceNamespace,
								Annotations: annotations,
							},
							Spec: appsv1.StatefulSetSpec{
								Replicas: int32Ptr(tc.replicaCount),
							},
						},
					),
				},

				// Reduce backoff and retries for simulated failures in unit tests
				retryBackoff: wait.Backoff{
					Duration: 10 * time.Millisecond,
					Factor:   1.0,
					Jitter:   0.1,
					Steps:    1,
				},

				// Do not wait for pods to be ready during tests
				skipPodWait: true,
			}

			// Simulate server side error
			if tc.name == "K8s Server Error" {
				k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
				k8sFakeClient.Fake.PrependReactor("get", "statefulsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("server side error")
				})
			}

			if tc.name == "group not present" {
				s.startUpOrder = nil
			}

			err := s.scaleUpGroup(tc.group)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				result, err := s.conf.K8sClient.AppsV1().StatefulSets(tc.resourceNamespace).Get(context.Background(), tc.resourceName, metav1.GetOptions{})
				require.NoError(t, err)

				if tc.replicasAnnotationKey == "" {
					assert.NotContains(t, result.Annotations, updatedAtAnnotationKey, "Expected the resource to be skipped as the original replicas annotation key was not set")
				} else {
					assert.Contains(t, result.Annotations, updatedAtAnnotationKey, "Expected the resource to have been updated")
					assert.NotContains(t, result.Annotations, originalReplicasAnnotationKey, "Expected the original replicas annotation to be removed following a successful update")

					var value string
					var found bool
					if value, found = result.Annotations[updatedAtAnnotationKey]; found {
						parsedTime, err := time.Parse(time.RFC3339, value)
						assert.NoError(t, err)
						assert.NotNil(t, parsedTime)
					}

					replica64, err := strconv.ParseInt(tc.replicasAnnotationKey, 10, 32)
					require.NoError(t, err)
					assert.Equal(t, int32(replica64), *result.Spec.Replicas, "Expected the number of replicas to be set based on the the original replicas annotation key")
				}
			}
		})
	}
}
