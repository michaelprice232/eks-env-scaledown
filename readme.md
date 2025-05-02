# eks-env-scaledown

> [!NOTE]
> Currently WIP and is being worked on in my spare time

Project to enable a K8s cluster which is backed by a node scheduler such as Karpenter to scale the cluster nodes to zero.
It does this by scaling all the workload replicas to zero and is designed to run in a non-production environment to save costs out of hours.
This will also take into account app dependencies and shutdown/startup in the correct order to avoid log/trace noise which can clutter observability systems.
The plan is also to handle autoscalers such as HPAs and Keda as well as integrations such as Slack and disabling of New Relic alerts.