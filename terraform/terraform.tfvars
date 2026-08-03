# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region            = "us-central1"
project_id_prefix = "agustinramirodiaz"
image_repository  = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest      = "sha256:ba0aec0ff23e87d1b9ecfb30d1e7b0bb06255b37c49710a1c319e5a674ae5012"
proxy_subnet_cidr = "10.10.1.0/24"
