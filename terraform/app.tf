resource "google_service_account" "runtime" {
  project = google_project.simple_time_service.project_id

  account_id   = "simple-time-service-runtime"
  display_name = "SimpleTimeService runtime"
  description  = "Least-privilege runtime identity for SimpleTimeService."

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "otel_collector_config" {
  project   = google_project.simple_time_service.project_id
  secret_id = "otel-collector-config"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "otel_collector_config" {
  secret      = google_secret_manager_secret.otel_collector_config.id
  secret_data = <<-EOF
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318

    processors:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 65
        spike_limit_percentage: 20
      batch:
        send_batch_max_size: 200
        send_batch_size: 200
        timeout: 5s
      resourcedetection:
        detectors: [env, gcp]
        timeout: 2s
        override: false
      resource:
        attributes:
          - key: service.name
            value: simple-time-service
            action: upsert

    exporters:
      googlemanagedprometheus:

    extensions:
      health_check:
        endpoint: 0.0.0.0:13133

    service:
      extensions: [health_check]
      pipelines:
        metrics:
          receivers: [otlp]
          processors: [memory_limiter, batch, resourcedetection, resource]
          exporters: [googlemanagedprometheus]
  EOF
}

resource "google_secret_manager_secret_iam_member" "runtime_can_read_otel_config" {
  project   = google_project.simple_time_service.project_id
  secret_id = google_secret_manager_secret.otel_collector_config.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.runtime.member
}

resource "google_project_iam_member" "runtime_can_write_metrics" {
  project = google_project.simple_time_service.project_id
  role    = "roles/monitoring.metricWriter"
  member  = google_service_account.runtime.member
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

    containers {
      name  = "app"
      image = "${var.image_repository}@${var.image_digest}"

      env {
        name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
        value = "http://localhost:4317"
      }

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

    containers {
      name  = "otel-collector"
      image = "us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google:0.156.0"
      args  = ["--config=/etc/otelcol-google/config.yaml"]

      startup_probe {
        timeout_seconds   = 30
        period_seconds    = 30
        failure_threshold = 3

        http_get {
          path = "/"
          port = 13133
        }
      }

      liveness_probe {
        timeout_seconds   = 30
        period_seconds    = 30
        failure_threshold = 3

        http_get {
          path = "/"
          port = 13133
        }
      }

      volume_mounts {
        name       = "otel-collector-config"
        mount_path = "/etc/otelcol-google"
      }

      resources {
        cpu_idle = true
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
      }
    }

    volumes {
      name = "otel-collector-config"

      secret {
        secret = google_secret_manager_secret.otel_collector_config.secret_id

        items {
          version = "latest"
          path    = "config.yaml"
        }
      }
    }
  }

  depends_on = [
    google_project_iam_member.runtime_can_write_metrics,
    google_secret_manager_secret_iam_member.runtime_can_read_otel_config,
  ]
}
