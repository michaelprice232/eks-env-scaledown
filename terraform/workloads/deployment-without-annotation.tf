resource "kubernetes_deployment_v1" "nginx_without_annotation" {
  metadata {
    name = "nginx-without-annotation"
    labels = {
      app = "nginx-without-annotation"
    }
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "nginx-without-annotation"
      }
    }

    template {
      metadata {
        labels = {
          app = "nginx-without-annotation"
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
                  values   = ["nginx-without-annotation"]
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