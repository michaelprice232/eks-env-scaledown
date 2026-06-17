# End-to-end tests

These tests exercise the project against a **real Kubernetes cluster** rather than the fake clients
used by the unit tests. To keep the feedback loop fast (and avoid provisioning EKS), the cluster is a
local [kind](https://kind.sigs.k8s.io/) (Kubernetes-in-Docker) cluster, provisioned with
[Terraform](https://github.com/tehcyx/terraform-provider-kind) and driven by
[Terratest](https://terratest.gruntwork.io/).

This is a **separate Go module** so Terratest's large dependency tree stays out of the production
module — the main `go build ./...`, `golangci-lint` and the Docker image are unaffected.

## What it does

`cluster_test.go` holds the tests. Each one provisions its **own** kind cluster (the app scales the
whole cluster, by design, so the tests must not share one), runs the app in-cluster as a Job, asserts
the outcome and tears the cluster down. The shared setup — build the image, provision the cluster, load
the image into the node — lives in the `provisionCluster` helper.

- **`TestScaleDownEndToEnd`** — deploys a sample Deployment (2 replicas), runs a `ScaleDown` Job, and
  asserts it was scaled to `0` with its original count saved in the `eks-env-scaledown/original-replicas`
  annotation.
- **`TestScaleUpEndToEnd`** — applies a Deployment that is already scaled to zero with the
  `original-replicas` annotation set (`manifests/deployment-prescaled.yaml`), so it does **not** depend on
  the scale-down test. Runs a `ScaleUp` Job and asserts the replica count is restored and the annotation
  is cleared. The Job sets `ALERT_STABILIZATION_DELAY=0s` to skip the production 10-minute settle wait.
- **`TestStartupOrderEndToEnd`** — brings up a statefulset (startup-order `0`) and a deployment
  (startup-order `1`, with a slow preStop hook simulating a graceful web-server shutdown). After a
  `ScaleDown` Job it asserts both reached `0` and that the statefulset's `updated-at` is *after* the
  deployment's — proving the tool shut down the higher-ordered group first and waited for it to drain.
- **`TestStandalonePodCleanupEndToEnd`** — applies a bare pod with no owning controller
  (`manifests/sample-workloads/standalone-pod.yaml`) and asserts the `ScaleDown` Job deletes it.

> Note: the app scales **cluster-wide** (by design, for dedicated environments), so it also scales
> `kube-system` Deployments like CoreDNS to zero inside the throwaway kind cluster. That's fine — the
> cluster is destroyed at the end of the test.

## Prerequisites

- A running **Docker** engine (Docker Desktop on Mac is fine).
- The **Terraform** CLI on your `PATH`.
- The **kind** CLI on your `PATH` — used to load the locally-built image into the cluster
  (`brew install kind`). The Terraform provider embeds kind for *cluster creation*, but image loading
  needs the CLI.

The first run pulls the `kindest/node` image (slow once, then cached).

## Running

From the repo root:

```shell
make test-e2e
```

or directly:

```shell
cd test/e2e && go test -v -tags e2e -timeout 20m ./...
```

The tests are guarded by the `e2e` build tag, so a normal `go test ./...` never runs them.

## Cleanup

The test always runs `terraform destroy` (via `defer`) to remove the cluster. If a run is killed
mid-way, clean up manually:

```shell
terraform -chdir=test/e2e/terraform destroy
```
