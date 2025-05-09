package service

import (
	"errors"
	"fmt"
	"io"
	log "log/slog"
	"testing"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func Test_BuildStartUpOrder_Deployments(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name              string
		wantErr           bool
		resourceName      string
		resourceNamespace string
		replicaCount      int32
		orderAnnotation   string
		expectedOrder     int
		appSelector       string
	}{
		{
			name:              "Normal assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "5",
			expectedOrder:     5,
			appSelector:       "nginx",
		},
		{
			name:              "No order annotation set so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Order too high so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "999999",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Order too low so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "-10",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Invalid order so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "BAD_INPUT",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:    "K8s Server Error",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var annotations map[string]string
			if tc.orderAnnotation != "" {
				annotations = map[string]string{
					startupOrderAnnotationKey: tc.orderAnnotation,
				}
			}

			s := &Service{
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
								Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": tc.appSelector}},
							},
						},
					),
				},
			}

			// Simulate server side error
			if tc.name == "K8s Server Error" {
				k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
				k8sFakeClient.Fake.PrependReactor("list", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("server side error")
				})
			}

			err := s.buildStartUpOrder()
			if tc.wantErr {
				assert.Error(t, err, "Expected error in test case: %s", tc.name)
			} else {
				assert.NoError(t, err, "Unexpected error in test case: %s", tc.name)
				assert.NotNil(t, s.startUpOrder, "Expected the startup order to be initialised")

				if s.startUpOrder != nil {
					assert.Contains(t, s.startUpOrder, tc.expectedOrder, "Expected the order number to be a key in the startupOrder map")

					assert.Equal(t, "deployment", s.startUpOrder[tc.expectedOrder][0].ResourceType)
					assert.Equal(t, tc.resourceName, s.startUpOrder[tc.expectedOrder][0].Name)
					assert.Equal(t, tc.resourceNamespace, s.startUpOrder[tc.expectedOrder][0].Namespace)
					assert.Equal(t, tc.replicaCount, s.startUpOrder[tc.expectedOrder][0].ReplicaCount)
					assert.Equal(t, fmt.Sprintf("app=%s", tc.appSelector), s.startUpOrder[tc.expectedOrder][0].Selector)
				}
			}

		})
	}
}

func Test_BuildStartUpOrder_Statefulsets(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name              string
		wantErr           bool
		resourceName      string
		resourceNamespace string
		replicaCount      int32
		orderAnnotation   string
		expectedOrder     int
		appSelector       string
	}{
		{
			name:              "Normal assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "5",
			expectedOrder:     5,
			appSelector:       "nginx",
		},
		{
			name:              "No order annotation set so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Order too high so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "999999",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Order too low so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "-10",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:              "Invalid order so default assignment",
			wantErr:           false,
			resourceName:      "nginx",
			resourceNamespace: "web",
			replicaCount:      2,
			orderAnnotation:   "BAD_INPUT",
			expectedOrder:     defaultStartUpGroup,
			appSelector:       "nginx",
		},
		{
			name:    "K8s Server Error",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var annotations map[string]string
			if tc.orderAnnotation != "" {
				annotations = map[string]string{
					startupOrderAnnotationKey: tc.orderAnnotation,
				}
			}

			s := &Service{
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
								Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": tc.appSelector}},
							},
						},
					),
				},
			}

			// Simulate server side error
			if tc.name == "K8s Server Error" {
				k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
				k8sFakeClient.Fake.PrependReactor("list", "statefulsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("server side error")
				})
			}

			err := s.buildStartUpOrder()
			if tc.wantErr {
				assert.Error(t, err, "Expected error in test case: %s", tc.name)
			} else {
				assert.NoError(t, err, "Unexpected error in test case: %s", tc.name)
				assert.NotNil(t, s.startUpOrder, "Expected the startup order to be initialised")

				if s.startUpOrder != nil {
					assert.Contains(t, s.startUpOrder, tc.expectedOrder, "Expected the order number to be a key in the startupOrder map")

					assert.Equal(t, "statefulset", s.startUpOrder[tc.expectedOrder][0].ResourceType)
					assert.Equal(t, tc.resourceName, s.startUpOrder[tc.expectedOrder][0].Name)
					assert.Equal(t, tc.resourceNamespace, s.startUpOrder[tc.expectedOrder][0].Namespace)
					assert.Equal(t, tc.replicaCount, s.startUpOrder[tc.expectedOrder][0].ReplicaCount)
					assert.Equal(t, fmt.Sprintf("app=%s", tc.appSelector), s.startUpOrder[tc.expectedOrder][0].Selector)
				}
			}

		})
	}
}

func Test_BuildStartUpOrder_MultipleGroups(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	s := &Service{
		conf: config.Config{
			K8sClient: fake.NewClientset(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "web",
						Annotations: map[string]string{
							startupOrderAnnotationKey: "2",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(2),
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
					},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "postgres",
						Namespace: "db",
						Annotations: map[string]string{
							startupOrderAnnotationKey: "0",
						},
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: int32Ptr(3),
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "postgres"}},
					},
				},
			),
		},
	}

	err := s.buildStartUpOrder()

	assert.NoError(t, err)
	assert.NotNil(t, s.startUpOrder, "Expected the startup order to be initialised")

	if s.startUpOrder != nil {
		assert.Contains(t, s.startUpOrder, 0, "Expected the order number to be a key in the startupOrder map")
		assert.Contains(t, s.startUpOrder, 2, "Expected the order number to be a key in the startupOrder map")
		assert.Equal(t, len(s.startUpOrder), 2, "Expected there to be two startup groups")
	}
}
