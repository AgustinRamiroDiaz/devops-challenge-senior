variable "billing_account_id" {
  description = "Billing account to attach to the new project. When empty, Terraform selects the first open account returned by gcloud."
  type        = string
  default     = ""
  sensitive   = true

  validation {
    condition     = var.billing_account_id == "" || can(regex("^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$", var.billing_account_id))
    error_message = "billing_account_id must be empty or use the XXXXXX-XXXXXX-XXXXXX format."
  }
}

variable "region" {
  description = "GCP region for the VPC subnets, Cloud Run service, and load balancer."
  type        = string
  default     = "us-central1"
}

variable "monthly_budget_amount" {
  description = "Monthly project budget in whole units of the billing account's currency."
  type        = number
  default     = 10

  validation {
    condition     = var.monthly_budget_amount > 0 && floor(var.monthly_budget_amount) == var.monthly_budget_amount
    error_message = "monthly_budget_amount must be a positive whole number."
  }
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

variable "image_digest" {
  description = "Immutable multi-platform container image digest deployed to Cloud Run."
  type        = string
  default     = "sha256:4048b7cf769e7b2deedd06d9bb019932a55a39a21fc410b7946d262513e0fefa"

  validation {
    condition     = can(regex("^sha256:[0-9a-f]{64}$", var.image_digest))
    error_message = "image_digest must be a lowercase sha256 digest."
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
