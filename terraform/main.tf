locals {
  project_id         = "${var.project_id_prefix}-${random_id.project_suffix.hex}"
  billing_account_id = var.billing_account_id != "" ? var.billing_account_id : data.external.gcloud_billing_account[0].result.billing_account_id

  required_services = toset([
    "compute.googleapis.com",
    "iam.googleapis.com",
    "run.googleapis.com",
    "serviceusage.googleapis.com",
  ])
}

resource "random_id" "project_suffix" {
  byte_length = 3
}

resource "google_project" "simple_time_service" {
  name            = "SimpleTimeService"
  project_id      = local.project_id
  billing_account = local.billing_account_id

  auto_create_network = false
  deletion_policy     = "DELETE"

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_project_service" "required" {
  for_each = local.required_services

  project = google_project.simple_time_service.project_id
  service = each.value

  disable_on_destroy         = false
  disable_dependent_services = false
}
