# EKS Environment Scaler

A Go-based utility to automate the scale-up and scale-down of services in an AWS EKS (Elastic Kubernetes Service) environment. 
This tool is useful for managing workloads efficiently across environments like staging or development, reducing cloud costs during idle periods.
When used alongside a tool such as Karpenter, this will enable all the compute/workers to be scaled to zero out of hours.
Designed to be run as K8s CronJobs to avoid the need for a long-lived controller (and associated compute cost) like with [cluster-turndown](https://github.com/kubecost/cluster-turndown)

## 🛠 Features

- Scale up/down Kubernetes Deployments, CronJobs, and StatefulSets
- Maintain startup ordering for dependencies to avoid log/tracing noise
- Slack and New Relic integration for notifications


## Running Locally

The following environment variables are available:

| Environment Variable        | Purpose                                                                                                                        |
|----------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| `SCALE_ACTION`             | Defines whether to scale resources up or down (can be `ScaleUp` or `ScaleDown`).                                               |
| `KUBE_CONTEXT`             | (optional) If running locally this specifies the Kubernetes context to operate in (e.g., `docker-desktop`).                    |
| `LOG_LEVEL`                | (optional) Sets the logging verbosity level (e.g., `info`, `debug`). Defaults to info.                                         |
| `SLACK_API_TOKEN`          | (optional) API token used to send scaling failure messages to Slack. Disabled if not set.                                      |
| `SLACK_CHANNEL_ID`         | (optional) Target Slack channel ID for notifications. Disabled if not set.                                                     |
| `ENVIRONMENT`              | (optional) The environment name the script operates against (e.g., `staging`). Only used when Slack notificaitons are enabled. |
| `NEW_RELIC_ALERT_POLICIES` | (optional) Comma-separated list of New Relic alert policy IDs to disable during environment scale downs. Disabled if not set.  |
| `NEW_RELIC_API_KEY`        | (optionall) API key to use when managing New Relic alerts during scaling. Disabled if not set.                                 |

```shell
# Scale cluster down using the "docker-desktop" k8s context
make scale-down

# Scale cluster up using the "docker-desktop" k8s context
make scale-up

# Override the k8s context for either command
make scale-down KUBE_CONTEXT=my-context
```

Sample K8s manifests are available in the [manifests directory](./manifests) for applying locally.

## Running tests

```shell
# Unit tests
make test

# Unit tests with HTML test coverage report
make cover
```

## How Scaling Works

<details>
<summary>During scale down:</summary>

1. New Relic alert policies are suspended (if this functionality is enabled via envars)
2. All CronJobs are suspended
    - If the CronJob is already suspended then an `eks-env-scaledown/cronjob-was-disabled` annotation is added so it isn't re-enabled at scaleup
    - If any have an `app` label equal to `eks-env-scaledown` they are skipped (meant for managing this process)
3. For all K8s Deployments and Statefulsets each is placed in a map group number based on the `eks-env-scaledown/startup-order` annotation (if set) e.g. "3". This must be a number from `0` -> `99`.
4. For any which do not have the annotation set they default to group `100` which is scaled down first
5. Iterates through the groups one at a time (highest to lowest):
   - If the replica count is already 0 then skips the resource
   - Sets the replica count to 0
   - Sets an annotation `eks-env-scaledown/original-replicas` containing the original number of replicas, used for scale up
   - Sets an annotation `eks-env-scaledown/updated-at` detailing the current date/time
   - Waits for all the pods to terminate before moving onto the next group
6. Terminate any remaining pods, including ones which are not managed by a controller
7. Any errors are alerted into Slack (if this functionality is enabled via envars)


</details>

<details>
<summary>During scale up:</summary>

1. For all K8s Deployments and Statefulsets each is placed in a map group number based on the `eks-env-scaledown/startup-order` annotation (if set) e.g. "3". This must be a number from `0` -> `99`.
2. For any which do not have the annotation set they default to group `100` which is scaled up last
3. Iterates through the groups one at a time (lowest to highest):
   - If the annotation `eks-env-scaledown/original-replicas` is not set skips the resource as it was either created after the scaledown or was already at zero replicas 
   - Reads the annotation `eks-env-scaledown/original-replicas` and sets the desired replica count to match
   - Removes the `eks-env-scaledown/original-replicas` annotation
   - Sets an annotation `eks-env-scaledown/updated-at` detailing the current date/time
   - Waits for all the pods to pass their readiness probes before moving onto the next group
4. All CronJobs are re-enabled
    - If the CronJob has an `eks-env-scaledown/cronjob-was-disabled` annotation it is skipped as it was disabled prior to scale down
    - If any have an `app` label equal to `eks-env-scaledown` they are skipped (meant for managing this process)
5. New Relic alert policies are re-enabled (if this functionality is enabled via envars)
6. Any errors are alerted into Slack (if this functionality is enabled via envars)

</details>