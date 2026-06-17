variable "cluster_name" {
  description = "Name of the kind cluster to create for the E2E tests"
  type        = string
  default     = "eks-env-scaledown-e2e"
}
