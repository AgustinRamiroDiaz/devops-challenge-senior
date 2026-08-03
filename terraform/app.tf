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
    # We use gen 1 since it allows for smaller than 512Mi memory, and also it has faster cold starts.
    execution_environment = "EXECUTION_ENVIRONMENT_GEN1"

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

      liveness_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 1
        period_seconds        = 10
        failure_threshold     = 3

        http_get {
          path = "/healthz"
          port = 8080
        }
      }

      resources {
        cpu_idle = true
        limits = {
          # Lower CPU would make sense, but it requires setting max_instance_request_concurrency=1 (as per https://docs.cloud.google.com/run/docs/configuring/services/cpu#cpu-min)
          # I think that for this simlpe service, we are better off with it handling a ton of requests concurrently instead of spinning up more replicas
          cpu    = "1"
          memory = "256Mi"
        }
      }
    }
  }
}
