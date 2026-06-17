# https://registry.terraform.io/modules/terraform-aws-modules/eks/aws
# Uses EKS mode to offload a number of controllers to AWS responsibility: https://docs.aws.amazon.com/eks/latest/userguide/automode.html
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "21.23.0"

  name               = var.environment
  kubernetes_version = var.eks_k8s_version
  vpc_id             = module.vpc.vpc_id
  subnet_ids         = module.vpc.private_subnets

  compute_config = {
    enabled    = true
    node_pools = ["general-purpose"]
  }

  # K8s API endpoint open to the world (locked down with just AuthN)
  endpoint_public_access = true

  enabled_log_types   = var.cluster_enabled_log_types
  authentication_mode = "API"

  enable_cluster_creator_admin_permissions = true
}