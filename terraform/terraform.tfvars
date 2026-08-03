# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region                = "us-central1"
project_id_prefix     = "agustinramirodiaz"
image_repository      = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest          = "sha256:bdf79b9093125a9dc77ea354b87b1f79f3ca35ad935d04b2a89819e2e34ff79d"
monthly_budget_amount = 10
proxy_subnet_cidr     = "10.10.1.0/24"
