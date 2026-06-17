provider "kind" {}

# A bare-bones single-node Kubernetes-in-Docker cluster used by the E2E tests.
# The provider embeds kind as a library, so only the Docker engine is required.
resource "kind_cluster" "this" {
  name           = var.cluster_name
  wait_for_ready = true
}
