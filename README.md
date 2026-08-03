# SimpleTimeService

SimpleTimeService is a minimal Go web service that returns the current UTC time and the visitor's IP address as JSON. This repository builds the application as a small, non-root container and deploys it to a Terraform-managed Google Cloud project.

This repo is the solution to the [tech exercise](EXERCISE.md) of Particle41

## Architecture

The deployment uses a regional external Application Load Balancer in front of a Cloud Run service:

```mermaid
flowchart LR
  client[Internet clients]

  subgraph project["GCP project"]
    subgraph vpc["VPC"]
      proxy_subnet["Proxy-only subnet<br/>10.10.1.0/24<br/>Regional managed proxy"]
    end

    ip["Static regional public IP"]

    subgraph alb["Regional external HTTP Application Load Balancer"]
      forwarding_rule["Forwarding rule<br/>Port 80"]
      http_proxy["HTTP proxy"]
      url_map["URL map"]
      backend["Backend service"]
    end

    neg["Serverless NEG"]
    cloud_run["Cloud Run service<br/>Ingress: internal and Cloud Load Balancing only"]
  end

  client --> ip --> forwarding_rule --> http_proxy --> url_map --> backend --> neg
  neg -- "Google-managed serverless routing" --> cloud_run
  proxy_subnet -. "Google-managed ALB proxies" .- alb
```

Terraform creates a custom VPC with `lb-proxy-subnet`, a regional proxy-only
subnet reserved for Google-managed load-balancer proxies.

Cloud Run and the serverless NEG are managed Google Cloud resources rather than
VMs placed inside a VPC subnet. The backend hop uses private Google-managed
routing outside the VPC firewall path. Because this service does not initiate
connections to private VPC resources, it does not need Direct VPC egress or a
workload subnet. The Cloud Run default URL is disabled and its ingress setting
accepts external traffic only through Cloud Load Balancing, so the load
balancer remains the only public path to the container.

This design keeps the service and networking operationally small, scales the application to zero when idle, and gives the public endpoint a stable IP.

I've decided to not use DNS for this project because:

- it's very hard to generate the domain name programatically without collisions
- usually requires to reserve the domain for at least a year
- it's expensive for a take-home challenge
- it's not part of the acceptance criteria

Useful docs:

- https://docs.cloud.google.com/load-balancing/docs/https/setting-up-reg-ext-https-serverless
- https://docs.cloud.google.com/run/docs/securing/private-networking

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) 1.10 or newer
- [Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
- [Docker](https://docs.docker.com/engine/install/) with Buildx to publish the application image
- A Google Cloud billing account
- Bootstrap permissions:
  - Project Creator (`roles/resourcemanager.projectCreator`)
  - Billing Account User (`roles/billing.user`) on the billing account

The public image must exist before Terraform deploys Cloud Run. I've already made the necessary release with:

```bash
docker login
make publish TAG=v1.0.2
```

Alternatively, use the manually triggered **Build and publish Docker image**
GitHub Actions workflow. Configure these repository secrets first:

- `DOCKERHUB_USERNAME`: Docker Hub username with access to the repository.
- `DOCKERHUB_TOKEN`: Docker Hub access token with permission to push images.

Run the workflow from the GitHub Actions page and provide the desired
`image_tag`, such as `v1.0.0`. It publishes that exact tag for `linux/amd64`,
`linux/arm64`, and `linux/arm/v7` to
`agustinramirodiaz/simpletimeservice`. The workflow only runs through manual
dispatch; repository pushes and Git tags do not trigger it.

## Authenticate to Google Cloud

Authenticate the Google Cloud CLI, then create Application Default Credentials for the Google provider:

```bash
gcloud auth login
gcloud auth application-default login
```

By default, Terraform selects the first open billing account available to the authenticated `gcloud` identity. You can inspect which accounts are available:

```bash
gcloud billing accounts list
```

No billing variable is required for the default path. To select a different account explicitly, use either of these overrides:

1. Configure with a file

   Copy the example file

   ```bash
   cp terraform/my.auto.tfvars.example terraform/my.auto.tfvars
   ```

   Then edit `terraform/my.auto.tfvars` with the desired account ID.

2. Configure with environment variables

   Export the chosen account ID. Environment variables keep account-specific data out of version control:

   ```bash
   export TF_VAR_billing_account_id="000000-000000-000000"
   ```

## Deploy

Run Terraform from the `terraform` directory:

```bash
cd terraform
terraform init
terraform plan -out=simple-time-service.tfplan
```

Review the saved plan before applying it:

```bash
terraform show simple-time-service.tfplan
```

Then apply that exact plan:

```bash
terraform apply simple-time-service.tfplan
```

Saving the plan ensures that `terraform apply` executes the same changes that
were reviewed, instead of recalculating a potentially different plan. If the
configuration, variables, provider selections, credentials, or existing
infrastructure change after the plan is created, discard it and generate a new
one before applying.

Saved plan files can contain sensitive configuration and state data. Keep them
local, do not share or commit them, and remove them when they are no longer
needed. Files ending in `.tfplan` are already excluded by this repository's
`.gitignore`.

The first apply creates a project named `agustinramirodiaz-<random-hex>`, links it to the billing account, enables the required APIs, and deploys all networking and application resources. Internal resources use stable descriptive names; only the globally unique project ID has a random suffix.

Test the service through the load balancer:

```bash
curl "$(terraform output -raw service_url)"
```

Expected response shape:

```json
{
  "timestamp": "2026-08-01T12:00:00Z",
  "ip": "203.0.113.10"
}
```

Load-balancer provisioning can take several minutes. If the first request fails, wait briefly and retry.

Useful outputs are available with:

```bash
terraform output
```

## Advanced

### Select the Terraform state backend

The committed Terraform configuration intentionally has no `backend` block,
so Terraform uses the local backend and stores state in
`terraform/terraform.tfstate` by default. From the repository root, initialize
or switch back to local state with:

```bash
make terraform-init
```

To use an existing Google Cloud Storage bucket instead, provide its name to the
GCS initialization target:

```bash
make terraform-init-gcs \
  GCS_BUCKET=my-terraform-state-bucket \
  GCS_PREFIX=env/dev
```

`GCS_PREFIX` is optional and defaults to `simple-time-service`. The command
generates `terraform/backend_override.tf` immediately before initialization:

```hcl
terraform {
  backend "gcs" {
    bucket = "my-terraform-state-bucket"
    prefix = "env/dev"
  }
}
```

The generated file is ignored by Git. Both initialization commands use
`terraform init -migrate-state`, so Terraform can copy existing state when you
switch between local and GCS backends. Review and confirm any migration prompt;
do not treat a newly empty state as interchangeable with the existing state.

For non-interactive initialization after the backend and state migration are
already established, pass additional initialization flags explicitly:

```bash
make terraform-init-gcs \
  GCS_BUCKET=my-terraform-state-bucket \
  GCS_PREFIX=env/dev \
  TF_INIT_FLAGS="-input=false"
```

The GCS bucket must exist before `terraform init`; it cannot be created by this
same root configuration because Terraform initializes the backend before it
creates resources. The authenticated identity also needs object access to the
bucket. Enable Object Versioning on the bucket so previous state versions can
be recovered after accidental deletion or corruption.

### Run Terraform from GitHub Actions

The repository has two keyless Terraform workflows:

- `terraform-plan.yml` runs against pull requests targeting `main`. It uses a
  read-only service account and does not lock or modify the remote state. Pull
  requests from forks continue to run the credential-free checks, but the
  remote plan is skipped so forked code cannot read the state.
- `terraform-apply.yml` runs after a push to `main`. It creates a fresh saved
  plan from that exact commit and applies it in the same job. A concurrency
  group prevents two production applies from running simultaneously.

Both workflows authenticate with GitHub's OIDC token through Google Cloud
Workload Identity Federation. There are no downloadable service-account keys.
The apply job uses the `production` GitHub environment; configure that
environment to allow only `main` and, when desired, require a reviewer before
the job can obtain credentials.

The project resource is protected with Terraform's `prevent_destroy`
lifecycle rule. A plan that would delete or replace the project fails before
apply. To intentionally delete the project, first remove that rule in a
reviewed change and run the destruction explicitly.

#### One-time Google Cloud bootstrap

The CI identities and federation resources intentionally live outside this
Terraform state: they must already exist before GitHub can run Terraform. The
current deployment uses these values:

```bash
APP_PROJECT_ID=agustinramirodiaz-cd1472
APP_PROJECT_NUMBER=449819592596
BILLING_ACCOUNT_ID=XXXXXX-XXXXXX-XXXXXX
REPOSITORY=AgustinRamiroDiaz/devops-challenge-senior
REPOSITORY_ID=1318802083
STATE_BUCKET=simple-time-service-tfstate
```

For a new deployment, obtain the generated project ID with
`terraform output -raw project_id`, its number with `gcloud projects describe`,
and the immutable repository ID with `gh api repos/$REPOSITORY --jq .id`.

Enable token exchange and create the two service accounts:

```bash
gcloud services enable \
  iamcredentials.googleapis.com \
  sts.googleapis.com \
  --project="$APP_PROJECT_ID"

gcloud iam service-accounts create terraform-plan \
  --project="$APP_PROJECT_ID" \
  --display-name="Terraform PR plan"

gcloud iam service-accounts create terraform-apply \
  --project="$APP_PROJECT_ID" \
  --display-name="Terraform main apply"
```

Grant the plan identity read-only access. Plans use `-lock=false`, allowing the
state bucket role to remain read-only:

```bash
gcloud projects add-iam-policy-binding "$APP_PROJECT_ID" \
  --member="serviceAccount:terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/viewer"

gcloud projects add-iam-policy-binding "$APP_PROJECT_ID" \
  --member="serviceAccount:terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/serviceusage.serviceUsageConsumer"

gcloud storage buckets add-iam-policy-binding "gs://$STATE_BUCKET" \
  --member="serviceAccount:terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectViewer"

gcloud billing accounts add-iam-policy-binding "$BILLING_ACCOUNT_ID" \
  --member="serviceAccount:terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/billing.viewer"
```

Grant the apply identity write access only to the product areas managed by this
configuration and to the state objects:

```bash
for role in \
  roles/viewer \
  roles/compute.networkAdmin \
  roles/compute.loadBalancerAdmin \
  roles/run.admin \
  roles/iam.serviceAccountAdmin \
  roles/iam.serviceAccountUser \
  roles/serviceusage.serviceUsageAdmin
do
  gcloud projects add-iam-policy-binding "$APP_PROJECT_ID" \
    --member="serviceAccount:terraform-apply@$APP_PROJECT_ID.iam.gserviceaccount.com" \
    --role="$role"
done

gcloud storage buckets add-iam-policy-binding "gs://$STATE_BUCKET" \
  --member="serviceAccount:terraform-apply@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

gcloud billing accounts add-iam-policy-binding "$BILLING_ACCOUNT_ID" \
  --member="serviceAccount:terraform-apply@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/billing.user"
```

Create a provider that accepts tokens only from this immutable repository ID,
then allow all repository refs to impersonate the read-only identity while only
`main` can impersonate the apply identity:

```bash
gcloud iam workload-identity-pools create github-actions \
  --project="$APP_PROJECT_ID" \
  --location=global \
  --display-name="GitHub Actions"

gcloud iam workload-identity-pools providers create-oidc github \
  --project="$APP_PROJECT_ID" \
  --location=global \
  --workload-identity-pool=github-actions \
  --display-name="devops-challenge-senior" \
  --issuer-uri="https://token.actions.githubusercontent.com/" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_id=assertion.repository_id,attribute.ref=assertion.ref,attribute.event_name=assertion.event_name" \
  --attribute-condition="assertion.repository_id=='$REPOSITORY_ID'"

gcloud iam service-accounts add-iam-policy-binding \
  "terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --project="$APP_PROJECT_ID" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/$APP_PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions/attribute.repository_id/$REPOSITORY_ID"

gcloud iam service-accounts add-iam-policy-binding \
  "terraform-apply@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --project="$APP_PROJECT_ID" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/$APP_PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions/attribute.ref/refs/heads/main"
```

Finally, configure these GitHub settings. The billing account ID is a repository
secret, the apply identity is scoped to the `production` environment, and the
remaining identifiers are repository variables:

| Setting | Type | Value for this deployment |
| --- | --- | --- |
| `GCP_PROJECT_ID` | Variable | `agustinramirodiaz-cd1472` |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Variable | `projects/449819592596/locations/global/workloadIdentityPools/github-actions/providers/github` |
| `GCP_PLAN_SERVICE_ACCOUNT` | Variable | `terraform-plan@agustinramirodiaz-cd1472.iam.gserviceaccount.com` |
| `GCP_APPLY_SERVICE_ACCOUNT` | `production` environment variable | `terraform-apply@agustinramirodiaz-cd1472.iam.gserviceaccount.com` |
| `GCP_BILLING_ACCOUNT_ID` | Secret | Your GCP billing account ID |
| `TF_STATE_BUCKET` | Variable | `simple-time-service-tfstate` |
| `TF_STATE_PREFIX` | Variable | `simple-time-service` |

The current repository already has these variables configured. Supplying the
billing account explicitly prevents CI from invoking the local `gcloud`
billing-account discovery fallback.

## Clean up

The project lifecycle guard intentionally blocks a normal destroy. To remove
the deployment, first delete `prevent_destroy = true` from the project resource
in a reviewed change, run and inspect a destruction plan, and then destroy all
resources to avoid ongoing charges:

```bash
terraform destroy
```

Terraform destroys the child resources and then schedules the entire project for deletion. Project deletion is initially recoverable in Google Cloud, so confirm the project is shut down and no billable resources remain.

## Contributing

See [Contributing.md](Contributing.md) for local development and automated check instructions.
