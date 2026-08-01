terraform {
  required_version = ">= 1.10, < 2.0"

  required_providers {
    external = {
      source  = "hashicorp/external"
      version = "~> 2.3"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 7.40"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.7"
    }
  }
}
