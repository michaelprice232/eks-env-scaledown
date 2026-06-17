# terraform

Terraform resources for provisioning an AWS VPC, EKS cluster and some sample workloads.

These are intended for **local testing only** — provision a real environment by hand when you want to
exercise the tool against a live EKS cluster. They are no longer used by the automated test suite: the
end-to-end (E2E) tests now run against a local Kubernetes cluster using Docker/Kind.