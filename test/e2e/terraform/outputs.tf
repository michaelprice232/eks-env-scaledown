output "kubeconfig_path" {
  description = "Path to the kubeconfig written for the kind cluster (current-context is kind-<name>)"
  value       = kind_cluster.this.kubeconfig_path
}


output "cluster_name" {
  description = "Name of the K8s cluster in Kind"
  value       = var.cluster_name
}