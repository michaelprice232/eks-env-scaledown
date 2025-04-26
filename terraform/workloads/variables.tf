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