# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region               = "us-central1"
project_id_prefix    = "agustinramirodiaz"
image_repository     = "docker.io/agustinramirodiaz/simpletimeservice"
image_tag            = "v1.0.0"
workload_subnet_cidr = "10.10.0.0/24"
proxy_subnet_cidr    = "10.10.1.0/24"
