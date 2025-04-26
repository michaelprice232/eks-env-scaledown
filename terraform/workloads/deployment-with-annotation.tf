resource "kubernetes_deployment_v1" "nginx_with_annotation" {
  metadata {
    name = "nginx-with-annotation"
    labels = {
      app = "nginx-with-annotation"
    }

    annotations = {
      "eks-env-scaledown/startup-order" = "1"
    }
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "nginx-with-annotation"
      }
    }

    template {
      metadata {
        labels = {
          app = "nginx-with-annotation"
        }
      }

      spec {

        # Spread across hosts
        affinity {
          pod_anti_affinity {
            required_during_scheduling_ignored_during_execution {
              topology_key = "kubernetes.io/hostname"
              label_selector {
                match_expressions {
                  key      = "app"
                  operator = "In"
                  values   = ["nginx-with-annotation"]
                }
              }
            }
          }
        }

        container {
          image = "nginx:1.27-alpine"
          name  = "app"


          port {
            container_port = 80
            name           = "http"
          }

          resources {
            limits = {
              memory = "50Mi"
            }
            requests = {
              cpu    = "50m"
              memory = "10Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/"
              port = 80
            }

            period_seconds = 30
          }

          startup_probe {
            http_get {
              path = "/"
              port = 80
            }

            period_seconds = 3
          }
        }
      }
    }
  }
}