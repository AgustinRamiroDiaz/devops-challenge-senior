# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region            = "us-central1"
project_id_prefix = "agustinramirodiaz"
image_repository  = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest      = "sha256:4353a100455343231e916f411a1adb3293bd564932b42348c8df8164fcc79346"
proxy_subnet_cidr = "10.10.1.0/24"
