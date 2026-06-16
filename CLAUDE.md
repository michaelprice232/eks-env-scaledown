# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

`eks-env-scaledown` is a run-once Go CLI (deployed as a Kubernetes CronJob/Job) that scales a whole
EKS environment down to zero and back up, so Karpenter can remove the worker nodes and save cost in
non-production clusters. A single invocation performs one action — `ScaleUp` or `ScaleDown` — then exits.

On scale **down** it: optionally pauses Keda ScaledObjects, suspends CronJobs, scales
Deployments/StatefulSets to zero in a defined order, terminates any leftover pods, and disables
CloudWatch + New Relic alerts. Scale **up** reverses this (restoring replica counts saved in
annotations) and re-enables alerts after a stabilisation delay. Errors are optionally reported to Slack.

It operates **cluster-wide across all namespaces** — by design, as it targets dedicated/throwaway
environments.

## Module / toolchain

- Module: `github.com/michaelprice232/eks-env-scaledown`
- Go 1.26.4

## Layout

- `cmd/main.go` — entrypoint; `run()` orchestrates the workflow and is the single place that exits.
- `config/` — env-driven `Config` plus the Kubernetes clients (typed + dynamic).
- `internal/service/` — core scaling logic: `service.go` (dispatch), `scaledown.go`, `scaleup.go`,
  `startup_order.go`, `cronjob.go`, `keda.go`.
- `internal/notify/` — integrations: `aws.go` (CloudWatch alarms), `new_relic.go` (alert policies),
  `slack.go` (error notifications).
- `manifests/` — Kubernetes manifests (controller + sample workloads). `terraform/` — VPC + EKS cluster.

## Common commands

- `make test` — unit tests (`go test ./...`)
- `make cover` — tests + HTML coverage report
- `make lint` — `golangci-lint run ./...` (config in `.golangci.yml`)
- `make scale-down` / `make scale-up` — run locally against `KUBE_CONTEXT` (default `docker-desktop`)
- `make build-docker` — multi-arch image build

## Configuration

Behaviour is driven entirely by environment variables — the full table is in `readme.md`. Most
important: `SCALE_ACTION` (`ScaleUp`/`ScaleDown`, required), `KUBE_CONTEXT`, `SUSPEND_CRONJOB`
(default true), `SUSPEND_KEDA_SCALED_OBJECTS` (default false), `MANAGE_CLOUDWATCH_ALARMS`,
`NEW_RELIC_*`, `SLACK_*`.

Per-workload behaviour is controlled by annotations: `eks-env-scaledown/startup-order` (0–99, default
group 100, started last / shut down first), and the tool-managed `original-replicas`, `updated-at`
and `cronjob-was-disabled`.

## Conventions (please follow)

- **Conventional Commits** for commit messages (`feat:`, `fix:`, `chore:`, …). The repo uses
  **rebase merging**, so your individual commits land on `main` and drive the semver bump — keep them
  conventional (PR titles do not matter for versioning).
- Go source files use lowercase `snake_case` names (e.g. `startup_order.go`).
- Tests are table-driven using `testify` and the client-go **fake** clients (typed + dynamic). Add
  tests alongside new logic.
- Keep it simple and readable — this is a focused tool; avoid unnecessary abstractions.
- Run `make lint` and `make test` before committing; both must be clean.

## CI / releases (`.github/workflows/ci.yml`)

Runs on every branch push:

1. **test-and-lint** — `go test` + golangci-lint.
2. **version** — on `main`, computes the next semver from the conventional commits since the last tag
   and pushes the git tag; on other branches the long git SHA is used.
3. **build-and-push-image** — builds the multi-arch (amd64/arm64) image and pushes it to ECR
   (`eu-west-2`) via an OIDC IAM role, tagged with the semver (main) or git SHA (branches).
