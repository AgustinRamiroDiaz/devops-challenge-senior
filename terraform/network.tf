resource "google_compute_network" "simple_time_service" {
  project = google_project.simple_time_service.project_id
  name    = "simple-time-service-vpc"

  auto_create_subnetworks         = false
  routing_mode                    = "REGIONAL"
  delete_default_routes_on_create = false

  depends_on = [google_project_service.required]
}

resource "google_compute_subnetwork" "load_balancer_proxy" {
  project = google_project.simple_time_service.project_id
  name    = "lb-proxy-subnet"
  region  = var.region
  network = google_compute_network.simple_time_service.id

  ip_cidr_range = var.proxy_subnet_cidr
  purpose       = "REGIONAL_MANAGED_PROXY"
  role          = "ACTIVE"
  stack_type    = "IPV4_ONLY"
}
