package service

import (
	"context"
	"errors"
	"io"
	log "log/slog"
	"testing"
	"time"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func Test_scaleDownGroup_deployments(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name              string
		wantErr           bool
		resourceName      string
		resourceNamespace string
		group             int
		replicaCount      int32
	}{
		{name: "normal scaledown", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 3},
		{name: "deployment has zero replicas: skip", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 0},
		{name: "group not present", wantErr: true},
		{name: "K8s Server Error", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

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
								Name:      tc.resourceName,
								Namespace: tc.resourceNamespace,
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

				// Do not wait for pods to be terminated during tests
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

			err := s.scaleDownGroup(tc.group)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				result, err := s.conf.K8sClient.AppsV1().Deployments(tc.resourceNamespace).Get(context.Background(), tc.resourceName, metav1.GetOptions{})
				assert.NoError(t, err)

				if tc.replicaCount == 0 {
					assert.NotContains(t, result.Annotations, updatedAtAnnotationKey)
					assert.NotContains(t, result.Annotations, originalReplicasAnnotationKey)

				} else {
					assert.Contains(t, result.Annotations, updatedAtAnnotationKey)
					assert.Contains(t, result.Annotations, originalReplicasAnnotationKey)

					var value string
					var found bool
					if value, found = result.Annotations[updatedAtAnnotationKey]; found {
						parsedTime, err := time.Parse(time.RFC3339, value)
						assert.NoError(t, err)
						assert.NotNil(t, parsedTime)
					}
				}
			}
		})
	}
}

func Test_scaleDownGroup_statefulsets(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name              string
		wantErr           bool
		resourceName      string
		resourceNamespace string
		group             int
		replicaCount      int32
	}{
		{name: "normal scaledown", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 3},
		{name: "deployment has zero replicas: skip", wantErr: false, resourceName: "nginx", resourceNamespace: "web", group: 2, replicaCount: 0},
		{name: "group not present", wantErr: true},
		{name: "K8s Server Error", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

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
								Name:      tc.resourceName,
								Namespace: tc.resourceNamespace,
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

				// Do not wait for pods to be terminated during tests
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

			err := s.scaleDownGroup(tc.group)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				result, err := s.conf.K8sClient.AppsV1().StatefulSets(tc.resourceNamespace).Get(context.Background(), tc.resourceName, metav1.GetOptions{})
				assert.NoError(t, err)

				if tc.replicaCount == 0 {
					assert.NotContains(t, result.Annotations, updatedAtAnnotationKey)
					assert.NotContains(t, result.Annotations, originalReplicasAnnotationKey)

				} else {
					assert.Contains(t, result.Annotations, updatedAtAnnotationKey)
					assert.Contains(t, result.Annotations, originalReplicasAnnotationKey)

					var value string
					var found bool
					if value, found = result.Annotations[updatedAtAnnotationKey]; found {
						parsedTime, err := time.Parse(time.RFC3339, value)
						assert.NoError(t, err)
						assert.NotNil(t, parsedTime)
					}
				}
			}
		})
	}
}

func Test_terminateStandalonePods(t *testing.T) {
	s := &Service{
		conf: config.Config{
			K8sClient: fake.NewClientset(
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "web",
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: "database",
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-3",
						Namespace: "database",
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
	}

	result, err := s.conf.K8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Items), "Expected 3 pods before the termination")

	err = s.terminateStandalonePods()
	assert.NoError(t, err)

	result, err = s.conf.K8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result.Items), "Expected 0 pods after the termination")
}

func Test_terminateStandalonePods_with_ignore_pod(t *testing.T) {
	s := &Service{
		conf: config.Config{
			K8sClient: fake.NewClientset(
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ignore-pod",
						Namespace: "app1",
						Labels: map[string]string{
							"app": cronJobAppName,
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: "database",
					},
				},
			),
		},
	}

	result, err := s.conf.K8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Items), "Expected 2 pods before the termination")

	err = s.terminateStandalonePods()
	assert.NoError(t, err)

	result, err = s.conf.K8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Items), "Expected 1 pod after the termination as 1 pod is ignored due to the app label")
}

func Test_podsStillRunning(t *testing.T) {
	tests := []struct {
		name            string
		resources       []*k8sResource
		expectedRunning bool
	}{
		{name: "none running", resources: []*k8sResource{}, expectedRunning: false},
		{name: "some running", resources: []*k8sResource{
			{Name: "pod-1", Namespace: "web"},
			{Name: "pod-2", Namespace: "database"},
		}, expectedRunning: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := podsStillRunning(tc.resources)
			assert.Equal(t, tc.expectedRunning, result)
		})
	}
}

func Test_waitForPodTermination(t *testing.T) {
	timeInterval = 100 * time.Millisecond
	timeout = 2 * time.Second

	s := &Service{
		conf: config.Config{
			K8sClient: fake.NewClientset(
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "web",
						Labels: map[string]string{
							"app": "nginx",
						},
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
	}

	// Fake K8s client doesn't display pod deletions by default e.g., list after delete. Adding a reactor to simulate this
	podDeleted := false
	k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
	k8sFakeClient.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if podDeleted {
			return true, &v1.PodList{Items: []v1.Pod{}}, nil
		}
		return true, &v1.PodList{Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "web",
					Labels:    map[string]string{"app": "nginx"},
				},
			},
		}}, nil
	})

	done := make(chan error)

	// Start in a separate go routine to allow us to dynamically terminate pods mid-test
	go func() {
		err := s.waitForPodTermination([]*k8sResource{{Name: "pod-1", Namespace: "web", ResourceType: "deployment", Selector: "app=nginx"}})
		done <- err
	}()

	time.Sleep(1 * time.Second)

	podDeleted = true

	// Wait for waitForPodTermination to run
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for waitForPodTermination to finish")
	}
}
