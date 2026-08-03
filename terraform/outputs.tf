output "project_id" {
  description = "ID of the Terraform-created GCP project."
  value       = google_project.simple_time_service.project_id
}

output "public_ip" {
  description = "Reserved regional public IPv4 address of the load balancer."
  value       = google_compute_address.simple_time_service.address
}

output "service_url" {
  description = "Public HTTP endpoint for SimpleTimeService."
  value       = "http://${google_compute_address.simple_time_service.address}"
}

output "billing_budget_name" {
  description = "Resource name of the project-scoped monthly billing budget."
  value       = google_billing_budget.simple_time_service.name
}
