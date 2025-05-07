package service

import (
	"testing"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_BuildStartUpOrder(t *testing.T) {
	type fields struct {
		Conf         config.Config
		StartUpOrder startUpOrder
	}
	tests := []struct {
		name                string
		fields              fields
		expectedNumOfGroups int
		wantErr             bool
	}{
		{
			name:                "MongoDB before nginx",
			wantErr:             false,
			expectedNumOfGroups: 2,
			fields: fields{
				Conf: config.Config{
					K8sClient: fake.NewClientset(
						&v1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nginx",
								Namespace: "web",
								Annotations: map[string]string{
									StartupOrderAnnotationKey: "1",
								},
							},
							Spec: v1.DeploymentSpec{
								Replicas: int32Ptr(2),
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "nginx"},
								},
							},
						},
						&v1.StatefulSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "mongodb",
								Namespace: "database",
								Annotations: map[string]string{
									StartupOrderAnnotationKey: "0",
								},
							},
							Spec: v1.StatefulSetSpec{
								Replicas: int32Ptr(2),
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "mongodb"},
								},
							},
						},
					),
				},
			},
		},
		{
			name:                "Nginx uses default group",
			wantErr:             false,
			expectedNumOfGroups: 2,
			fields: fields{
				Conf: config.Config{
					K8sClient: fake.NewClientset(
						&v1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "nginx",
								Namespace: "web",
							},
							Spec: v1.DeploymentSpec{
								Replicas: int32Ptr(2),
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "nginx"},
								},
							},
						},
						&v1.StatefulSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "mongodb",
								Namespace: "database",
								Annotations: map[string]string{
									StartupOrderAnnotationKey: "0",
								},
							},
							Spec: v1.StatefulSetSpec{
								Replicas: int32Ptr(2),
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "mongodb"},
								},
							},
						},
					),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				conf:         tt.fields.Conf,
				startUpOrder: tt.fields.StartUpOrder,
			}
			err := s.buildStartUpOrder()
			if tt.wantErr {
				assert.Error(t, err, "Expected error in test case: %s", tt.name)
			} else {
				assert.NoError(t, err, "Unexpected error in test case: %s", tt.name)
			}

			assert.Equal(t, tt.expectedNumOfGroups, len(s.startUpOrder), "Unexpected number of startup groups in test case: %s", tt.name)

		})
	}
}
