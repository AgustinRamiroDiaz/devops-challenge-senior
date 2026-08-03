# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region                = "us-central1"
project_id_prefix     = "agustinramirodiaz"
image_repository      = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest          = "sha256:4048b7cf769e7b2deedd06d9bb019932a55a39a21fc410b7946d262513e0fefa"
monthly_budget_amount = 10
proxy_subnet_cidr     = "10.10.1.0/24"
