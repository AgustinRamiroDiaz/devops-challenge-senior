# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region                = "us-central1"
project_id_prefix     = "agustinramirodiaz"
image_repository      = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest          = "sha256:eb59a9d146a6a7347500052e9b577c090b971a3ec16156aba32583f41f6c04ac"
monthly_budget_amount = 10
proxy_subnet_cidr     = "10.10.1.0/24"
