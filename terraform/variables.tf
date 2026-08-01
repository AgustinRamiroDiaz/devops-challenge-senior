variable "billing_account_id" {
  description = "Billing account to attach to the new project, in XXXXXX-XXXXXX-XXXXXX format."
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$", var.billing_account_id))
    error_message = "billing_account_id must use the XXXXXX-XXXXXX-XXXXXX format."
  }
}

variable "region" {
  description = "GCP region for the VPC subnets, Cloud Run service, and load balancer."
  type        = string
  default     = "us-central1"
}

variable "project_id_prefix" {
  description = "Prefix for the globally unique project ID. Terraform appends six hexadecimal characters."
  type        = string
  default     = "agustinramirodiaz"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,21}[a-z0-9]$", var.project_id_prefix))
    error_message = "project_id_prefix must be 6-23 lowercase letters, digits, or hyphens, start with a letter, and end with a letter or digit."
  }
}

variable "image_repository" {
  description = "Public container image repository consumed by Cloud Run."
  type        = string
  default     = "docker.io/agustinramirodiaz/simpletimeservice"
}

variable "image_tag" {
  description = "Container image tag deployed to Cloud Run."
  type        = string
  default     = "latest"

  validation {
    condition     = length(trimspace(var.image_tag)) > 0
    error_message = "image_tag cannot be empty."
  }
}

variable "workload_subnet_cidr" {
  description = "Primary IPv4 range used by Cloud Run Direct VPC egress."
  type        = string
  default     = "10.10.0.0/24"

  validation {
    condition     = can(cidrnetmask(var.workload_subnet_cidr))
    error_message = "workload_subnet_cidr must be a valid IPv4 CIDR range."
  }
}

variable "proxy_subnet_cidr" {
  description = "Primary IPv4 range reserved for regional managed load-balancer proxies."
  type        = string
  default     = "10.10.1.0/24"

  validation {
    condition     = can(cidrnetmask(var.proxy_subnet_cidr))
    error_message = "proxy_subnet_cidr must be a valid IPv4 CIDR range."
  }
}
