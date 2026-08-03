# Add e2e tests

- curl it and get current ip
- crash it and check the probes

# Add logging

# Dynamic Backend File Injection (Pre-Init Step)

To switch between local state and GCP bucket state dynamically in your CI/CD or CLI workflow, generate a temporary .tf file right before running terraform init.
Steps:

    Keep your main Terraform configuration free of any backend block.

    In your build/run script, conditionally inject a backend definition file:

For Remote (GCP):
Bash

cat <<EOF > backend_override.tf
terraform {
backend "gcs" {
bucket = "my-gcp-bucket"
prefix = "env/dev"
}
}
EOF

terraform init

For Local:
Simply ensure backend_override.tf is removed or empty. Terraform will automatically fall back to the default local backend (terraform.tfstate).
