package service

import (
	"context"
	"errors"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	log "log/slog"
	"testing"
	"time"

	"github.com/michaelprice232/eks-env-scaledown/config"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func Test_updateCronJobs(t *testing.T) {
	log.SetDefault(log.New(log.NewJSONHandler(io.Discard, &log.HandlerOptions{Level: log.LevelError})))

	tests := []struct {
		name               string
		wantErr            bool
		cronjobName        string
		cronjobNamespace   string
		action             config.ScaleAction
		appLabel           string
		disabledAnnotation string
		alreadySuspended   bool
	}{
		{name: "normal scaleup", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleUp},
		{name: "normal scaledown", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleDown},
		{name: "Correct app label set: skip", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleDown, appLabel: cronJobAppName},
		{name: "Incorrect app label set: do not skip", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleDown, appLabel: "incorrect-app-label"},
		{name: "Previously disabled: skip", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleUp, disabledAnnotation: "yes"},
		{name: "Incorrect previously disabled annotation: do not skip", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleUp, disabledAnnotation: "invalid"},
		{name: "Already scaled down: set additional annotation", wantErr: false, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleDown, alreadySuspended: true},
		{name: "K8s Server Error", wantErr: true, cronjobName: "my-batch-job", cronjobNamespace: "my-ns", action: config.ScaleDown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var labels map[string]string
			if tc.appLabel != "" {
				labels = map[string]string{
					"app": tc.appLabel,
				}
			}

			var annotations map[string]string
			if tc.disabledAnnotation != "" {
				annotations = map[string]string{
					cronJobWasDisabledAnnotationKey: tc.disabledAnnotation,
				}
			}

			s := &Service{
				conf: config.Config{
					Action: tc.action,

					K8sClient: fake.NewClientset(
						&batchv1.CronJob{
							ObjectMeta: metav1.ObjectMeta{
								Name:        tc.cronjobName,
								Namespace:   tc.cronjobNamespace,
								Labels:      labels,
								Annotations: annotations,
							},
							Spec: batchv1.CronJobSpec{
								Suspend: boolPtr(tc.alreadySuspended),
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

			// Simulate server side error
			if tc.name == "K8s Server Error" {
				k8sFakeClient := s.conf.K8sClient.(*fake.Clientset)
				k8sFakeClient.Fake.PrependReactor("update", "cronjobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("server side error")
				})
			}

			err := s.updateCronJobs()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				result, err := s.conf.K8sClient.BatchV1().CronJobs(tc.cronjobNamespace).Get(context.Background(), tc.cronjobName, metav1.GetOptions{})
				assert.NoError(t, err)

				var expectedToSkip bool
				if tc.appLabel == cronJobAppName {
					expectedToSkip = true
				}

				if tc.action == config.ScaleUp && tc.disabledAnnotation == "yes" {
					expectedToSkip = true
				}

				if expectedToSkip {
					assert.NotContains(t, result.Annotations, updatedAtAnnotationKey)
				} else {
					if tc.action == config.ScaleUp {
						assert.Falsef(t, *result.Spec.Suspend, "Expected CronJob %s in namespace %s to be not suspended following a scaleup", tc.cronjobName, tc.cronjobNamespace)
					}

					if tc.action == config.ScaleDown {
						assert.Truef(t, *result.Spec.Suspend, "Expected CronJob %s in namespace %s to be suspended following a scaledown", tc.cronjobName, tc.cronjobNamespace)

						if tc.alreadySuspended {
							assert.Containsf(t, result.Annotations, cronJobWasDisabledAnnotationKey, "Expected annotation %s to be set as the CronJob is already suspended prior to a scale down", cronJobWasDisabledAnnotationKey)
						}
					}

					assert.Contains(t, result.Annotations, updatedAtAnnotationKey)
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
