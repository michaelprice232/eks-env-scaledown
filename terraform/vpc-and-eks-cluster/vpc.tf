module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.19.0"

  name               = var.environment
  cidr               = var.vpc_cidr_block
  azs                = data.aws_availability_zones.available.zone_ids
  private_subnets    = [for i in range(3) : cidrsubnet(var.vpc_cidr_block, 4, i)]
  public_subnets     = [for i in range(3, 6) : cidrsubnet(var.vpc_cidr_block, 4, i)]
  enable_nat_gateway = true
  single_nat_gateway = true

  # Required by the load balancer controller: https://docs.aws.amazon.com/eks/latest/userguide/tag-subnets-auto.html
  public_subnet_tags = {
    "kubernetes.io/role/elb" : 1
  }
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" : 1
  }
}