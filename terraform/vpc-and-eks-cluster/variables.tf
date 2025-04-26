
variable "environment" {
  description = "The environment we are deploying to"
  type        = string
  default     = "test"
}

variable "region" {
  description = "AWS region we are deploying to"
  type        = string
  default     = "eu-west-1"
}

variable "vpc_cidr_block" {
  description = "CIDR block to assign to VPC"
  type        = string
  default     = "10.10.0.0/16"
}

variable "eks_k8s_version" {
  description = "Version of K8s to run"
  type        = string
  default     = "1.32"
}

variable "cluster_enabled_log_types" {
  description = "A list of the desired control plane logs to enable (audit, api etc). For more information, see Amazon EKS Control Plane Logging documentation (https://docs.aws.amazon.com/eks/latest/userguide/control-plane-logs.html)"
  type        = list(string)
  default     = []
}