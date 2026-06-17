//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	imageTag     = "eks-env-scaledown:e2e"
	repoRoot     = "../.."
	appNamespace = "eks-env-scaledown"

	startupOrderAnno     = "eks-env-scaledown/startup-order"
	originalReplicasAnno = "eks-env-scaledown/original-replicas"
	updatedAtAnno        = "eks-env-scaledown/updated-at"
)

// provisionCluster builds the app image, provisions a fresh kind cluster via Terraform, loads the
// locally-built image into it, and returns the kubeconfig path. The cluster is torn down when the
// test ends. Each test gets its own cluster: the app scales the whole cluster (by design), so tests
// must not share one.
func provisionCluster(t *testing.T) string {
	// Build the app image on the host Docker daemon (a no-op cache hit after the first test).
	docker.Build(t, repoRoot, &docker.BuildOptions{Tags: []string{imageTag}})

	tfOpts := &terraform.Options{TerraformDir: "terraform"}
	t.Cleanup(func() { terraform.Destroy(t, tfOpts) })
	terraform.InitAndApply(t, tfOpts)

	kubeconfig := terraform.Output(t, tfOpts, "kubeconfig_path")
	clusterName := terraform.Output(t, tfOpts, "cluster_name")

	// kind nodes run their own containerd, separate from the host Docker daemon, so a host image must
	// be loaded before pods can use it. This is inherent to Docker-based clusters (k3d/minikube-docker
	// are identical).
	shell.RunCommand(t, shell.Command{
		Command: "kind",
		Args:    []string{"load", "docker-image", imageTag, "--name", clusterName},
	})

	return kubeconfig
}

// runAppJob installs the app's namespace + RBAC, applies the given Job manifest (which runs the app
// in-cluster), and waits for the Job to complete.
func runAppJob(t *testing.T, appNS *k8s.KubectlOptions, jobManifest, jobName string) {
	k8s.KubectlApply(t, appNS, repoRoot+"/manifests/controller/namespace.yaml")
	k8s.KubectlApply(t, appNS, repoRoot+"/manifests/controller/rbac.yaml")
	k8s.KubectlApply(t, appNS, jobManifest)
	k8s.WaitUntilJobSucceedContext(t, context.Background(), appNS, jobName, 90, 3*time.Second)
}

// TestScaleDownEndToEnd exercises the core scale-down logic: it deploys a sample Deployment, runs the
// app in-cluster as a ScaleDown Job, and asserts the Deployment was scaled to zero with its original
// replica count preserved in an annotation.
func TestScaleDownEndToEnd(t *testing.T) {
	const (
		deploymentName = "nginx-with-annotation"
		deploymentFile = repoRoot + "/manifests/sample-workloads/deployment-1.yaml"
	)

	ctx := context.Background()
	kubeconfig := provisionCluster(t)
	defaultNS := k8s.NewKubectlOptions("", kubeconfig, "default")
	appNS := k8s.NewKubectlOptions("", kubeconfig, appNamespace)

	// Deploy a sample workload (2 replicas) and wait for it to be ready.
	k8s.KubectlApply(t, defaultNS, deploymentFile)
	k8s.WaitUntilDeploymentAvailableContext(t, ctx, defaultNS, deploymentName, 60, 2*time.Second)

	runAppJob(t, appNS, "manifests/scaledown-job.yaml", "eks-env-scaledown-down")

	deployment := k8s.GetDeployment(t, defaultNS, deploymentName)
	require.NotNil(t, deployment.Spec.Replicas, "deployment spec.replicas should be set")
	require.Equal(t, int32(0), *deployment.Spec.Replicas, "expected the deployment to be scaled to zero")
	require.Equal(t, "2", deployment.Annotations[originalReplicasAnno],
		"expected the original replica count saved in an annotation")
}

// TestScaleUpEndToEnd exercises the restore path. The Deployment is pre-scaled to zero with the
// original-replicas annotation already set (as if a previous scale-down had run), so this test does
// not depend on the scale-down test. The app should restore the replica count and clear the annotation.
func TestScaleUpEndToEnd(t *testing.T) {
	const (
		deploymentName = "nginx-prescaled"
		deploymentFile = "manifests/deployment-prescaled.yaml"
	)

	ctx := context.Background()
	kubeconfig := provisionCluster(t)
	defaultNS := k8s.NewKubectlOptions("", kubeconfig, "default")
	appNS := k8s.NewKubectlOptions("", kubeconfig, appNamespace)

	k8s.KubectlApply(t, defaultNS, deploymentFile)

	runAppJob(t, appNS, "manifests/scaleup-job.yaml", "eks-env-scaledown-up")

	// The Job already waits for the restored pods to be ready before completing; re-assert here.
	k8s.WaitUntilDeploymentAvailableContext(t, ctx, defaultNS, deploymentName, 60, 2*time.Second)

	deployment := k8s.GetDeployment(t, defaultNS, deploymentName)
	require.NotNil(t, deployment.Spec.Replicas, "deployment spec.replicas should be set")
	require.Equal(t, int32(2), *deployment.Spec.Replicas, "expected the deployment restored to its original replica count")
	require.NotContains(t, deployment.Annotations, originalReplicasAnno,
		"expected the original-replicas annotation removed after scale-up")
}

// TestStartupOrderEndToEnd verifies the tool honours the startup-order annotation across more than one
// workload. The statefulset (order 0) is shut down last, the deployment (order 1) first. The deployment
// has a slow preStop hook simulating a graceful web-server shutdown, so the tool must wait for it to
// fully terminate before scaling the lower-ordered statefulset. Ordering is asserted via the updated-at
// timestamps the tool writes as it scales each group.
func TestStartupOrderEndToEnd(t *testing.T) {
	const (
		deploymentName  = "slow-shutdown"
		statefulSetName = "httpd"
		deploymentFile  = "manifests/deployment-slow-shutdown.yaml"
		statefulSetFile = repoRoot + "/manifests/sample-workloads/statefulset.yaml"
	)

	ctx := context.Background()
	kubeconfig := provisionCluster(t)
	defaultNS := k8s.NewKubectlOptions("", kubeconfig, "default")
	appNS := k8s.NewKubectlOptions("", kubeconfig, appNamespace)

	// Bring both workloads up. The deployment must be running so its preStop hook actually fires.
	k8s.KubectlApply(t, defaultNS, statefulSetFile)
	k8s.KubectlApply(t, defaultNS, deploymentFile)
	k8s.WaitUntilDeploymentAvailableContext(t, ctx, defaultNS, deploymentName, 60, 2*time.Second)
	k8s.WaitUntilNumPodsCreatedContext(t, ctx, defaultNS,
		metav1.ListOptions{LabelSelector: "app=" + statefulSetName}, 2, 60, 2*time.Second)

	runAppJob(t, appNS, "manifests/scaledown-job.yaml", "eks-env-scaledown-down")

	// Both workloads scaled to zero with their original counts preserved.
	deployment := k8s.GetDeployment(t, defaultNS, deploymentName)
	require.NotNil(t, deployment.Spec.Replicas, "deployment spec.replicas should be set")
	require.Equal(t, int32(0), *deployment.Spec.Replicas, "expected the deployment to be scaled to zero")
	require.Equal(t, "1", deployment.Annotations[originalReplicasAnno], "expected the deployment original replicas preserved")

	clientset, err := k8s.GetKubernetesClientFromOptionsE(t, defaultNS)
	require.NoError(t, err)
	statefulset, err := clientset.AppsV1().StatefulSets("default").Get(ctx, statefulSetName, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, statefulset.Spec.Replicas, "statefulset spec.replicas should be set")
	require.Equal(t, int32(0), *statefulset.Spec.Replicas, "expected the statefulset to be scaled to zero")
	require.Equal(t, "2", statefulset.Annotations[originalReplicasAnno], "expected the statefulset original replicas preserved")

	// The deployment (order 1) is scaled before the statefulset (order 0), and the tool waits for the
	// deployment's slow shutdown in between, so the statefulset's updated-at must be strictly later.
	deployUpdatedAt, err := time.Parse(time.RFC3339, deployment.Annotations[updatedAtAnno])
	require.NoError(t, err, "deployment updated-at should be a valid RFC3339 timestamp")
	stsUpdatedAt, err := time.Parse(time.RFC3339, statefulset.Annotations[updatedAtAnno])
	require.NoError(t, err, "statefulset updated-at should be a valid RFC3339 timestamp")
	require.True(t, stsUpdatedAt.After(deployUpdatedAt),
		"expected the statefulset (order 0) to be scaled down after the deployment (order 1): deployment=%s statefulset=%s",
		deployUpdatedAt, stsUpdatedAt)
}

// TestStandalonePodCleanupEndToEnd verifies that a pod with no owning controller (and without the
// app's own label) is deleted during scale-down. With no controller to recreate it, the pod stays gone.
func TestStandalonePodCleanupEndToEnd(t *testing.T) {
	const (
		podName = "standalone-pod"
		podFile = repoRoot + "/manifests/sample-workloads/standalone-pod.yaml"
	)

	ctx := context.Background()
	kubeconfig := provisionCluster(t)
	defaultNS := k8s.NewKubectlOptions("", kubeconfig, "default")
	appNS := k8s.NewKubectlOptions("", kubeconfig, appNamespace)

	k8s.KubectlApply(t, defaultNS, podFile)
	k8s.WaitUntilPodAvailableContext(t, ctx, defaultNS, podName, 60, 2*time.Second)

	runAppJob(t, appNS, "manifests/scaledown-job.yaml", "eks-env-scaledown-down")

	// Deletion is graceful, so the pod may still be terminating when the Job completes — poll until gone.
	require.Eventually(t, func() bool {
		_, err := k8s.GetPodE(t, defaultNS, podName)
		return err != nil
	}, 60*time.Second, 2*time.Second, "expected the standalone pod to be deleted")
}
