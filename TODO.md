# Add e2e test

# Add probes

health and readiness

# Pick up billing account automatically from gcloud

```
# 1. Define an optional input variable (defaults to empty string)
variable "billing_account_id" {
  type        = string
  default     = ""
  description = "GCP Billing Account ID. If left empty, will automatically fetch the open account via gcloud."
  sensitive   = true
}

# 2. Only execute gcloud if var.billing_account_id is NOT provided
data "external" "gcloud_billing_account" {
  count = var.billing_account_id == "" ? 1 : 0

  program = [
    "bash", "-c",
    <<-EOF
      ACCOUNT_ID=$(gcloud billing accounts list --filter="open=true" --format="value(name)" --limit=1)
      echo "{\"billing_account_id\": \"$ACCOUNT_ID\"}"
    EOF
  ]
}

# 3. Create a local value that resolves the override or falls back to the data source
locals {
  billing_account_id = var.billing_account_id != "" ? var.billing_account_id : data.external.gcloud_billing_account[0].result.billing_account_id
}

# 4. Use the resolved local value in your resources
resource "google_project" "my_project" {
  name            = "My App Project"
  project_id      = "my-app-project-12345"
  billing_account = local.billing_account_id
}
```
