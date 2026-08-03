# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region            = "us-central1"
project_id_prefix = "agustinramirodiaz"
image_repository  = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest      = "sha256:404aa5e856c08fcefd87906ae625eb316cad9c919910e82f6e0db89873869c95"
proxy_subnet_cidr = "10.10.1.0/24"
