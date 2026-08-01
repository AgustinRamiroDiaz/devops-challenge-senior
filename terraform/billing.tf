data "external" "gcloud_billing_account" {
  count = var.billing_account_id == "" ? 1 : 0

  program = [
    "bash",
    "-c",
    <<-EOT
      set -euo pipefail

      if ! command -v gcloud >/dev/null 2>&1; then
        echo "gcloud is required when billing_account_id is not provided" >&2
        exit 1
      fi

      account_id=$(gcloud billing accounts list --filter="open=true" --format="value(name)" --limit=1 | sed 's#^billingAccounts/##')

      if [ -z "$account_id" ]; then
        echo "No open billing account is available to the authenticated gcloud identity" >&2
        exit 1
      fi

      printf '{"billing_account_id":"%s"}\n' "$account_id"
    EOT
  ]
}
