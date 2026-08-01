resource "google_service_account" "runtime" {
  project = google_project.simple_time_service.project_id

  account_id   = "simple-time-service-runtime"
  display_name = "SimpleTimeService runtime"
  description  = "Least-privilege runtime identity for SimpleTimeService."

  depends_on = [google_project_service.required]
}

resource "google_cloud_run_v2_service" "simple_time_service" {
  project  = google_project.simple_time_service.project_id
  name     = "simple-time-service"
  location = var.region

  ingress              = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
  default_uri_disabled = true
  invoker_iam_disabled = true
  deletion_protection  = false

  template {
    service_account                  = google_service_account.runtime.email
    max_instance_request_concurrency = 80
    execution_environment            = "EXECUTION_ENVIRONMENT_GEN1"

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = google_compute_network.simple_time_service.name
        subnetwork = google_compute_subnetwork.cloud_run.name
      }
    }

    containers {
      image = "${var.image_repository}:${var.image_tag}"

      ports {
        name           = "http1"
        container_port = 8080
      }

      resources {
        cpu_idle = true
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
      }
    }
  }
}
