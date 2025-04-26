data "aws_eks_cluster_auth" "cluster" {
  name = var.environment
}

data "aws_eks_cluster" "cluster" {
  name = var.environment
}