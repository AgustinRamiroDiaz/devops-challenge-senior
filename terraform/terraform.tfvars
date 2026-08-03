# billing_account_id is selected automatically from gcloud unless explicitly
# supplied through TF_VAR_billing_account_id or a local .tfvars file.
region            = "us-central1"
project_id_prefix = "agustinramirodiaz"
image_repository  = "docker.io/agustinramirodiaz/simpletimeservice"
image_digest      = "sha256:73e6c8742e31d45414d6b58beb3ecbbf51b971ff5973b1d332aae1a48d3c7cf7"
proxy_subnet_cidr = "10.10.1.0/24"
