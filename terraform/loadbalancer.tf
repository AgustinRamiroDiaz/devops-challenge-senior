resource "google_compute_region_network_endpoint_group" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-neg"
  region  = var.region

  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = google_cloud_run_v2_service.simple_time_service.name
  }
}

resource "google_compute_region_backend_service" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-backend"
  region  = var.region

  protocol              = "HTTP"
  load_balancing_scheme = "EXTERNAL_MANAGED"

  backend {
    group = google_compute_region_network_endpoint_group.simple_time_service.id
  }

  log_config {
    enable      = true
    sample_rate = 1.0
  }
}

resource "google_compute_region_url_map" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-url-map"
  region  = var.region

  default_service = google_compute_region_backend_service.simple_time_service.id
}

resource "google_compute_region_target_http_proxy" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-http-proxy"
  region  = var.region

  url_map = google_compute_region_url_map.simple_time_service.id
}

resource "google_compute_address" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-ip"
  region  = var.region

  address_type = "EXTERNAL"
  network_tier = "STANDARD"
}

resource "google_compute_forwarding_rule" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-http"
  region  = var.region

  load_balancing_scheme = "EXTERNAL_MANAGED"
  network               = google_compute_network.simple_time_service.id
  network_tier          = "STANDARD"
  ip_address            = google_compute_address.simple_time_service.id
  port_range            = "80"
  target                = google_compute_region_target_http_proxy.simple_time_service.id
}
