resource "kubernetes_stateful_set_v1" "apache_httpd" {
  metadata {
    name = "httpd"

    labels = {
      app = "httpd"
    }

    annotations = {
      # Startup BEFORE the nginx deployment set
      "eks-env-scaledown/startup-order" = "0"
    }
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "httpd"
      }
    }

    service_name = "httpd"

    template {
      metadata {
        labels = {
          app = "httpd"
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
                  values   = ["httpd"]
                }
              }
            }
          }
        }

        container {
          name              = "app"
          image             = "httpd:2.4"

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