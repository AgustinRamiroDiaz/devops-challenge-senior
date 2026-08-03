# Decisions

## Flat terraform structure

Given that I don't have complex conditional logic, nor big reusable custom resources, I've opted out for a flat structure for all terraform resources. I've split them in files so that it's easier to mentally group and read.

Using a flat structure has the benefits of:

- less code and it's less error prone, since variables and outputs take a lot of space and are places where it's easy to missconfig.
- easier to read, since there are no custom modules with my own defined variables. By using standard terraform modules like `google`, everyone can read the resources and the code is familiar.

## Pin deployed images by digest

Cloud Run deploys the image using its immutable OCI manifest digest rather
than a mutable tag. The published `v1.0.3` multi-platform image currently
resolves to:

```text
docker.io/agustinramirodiaz/simpletimeservice@sha256:4353a100455343231e916f411a1adb3293bd564932b42348c8df8164fcc79346
```

Tags remain useful for publishing and discovering releases, but a registry
owner can move a tag to different content. A digest makes Terraform revisions
reproducible and ensures an apply deploys the image that was reviewed.

After publishing a new release, resolve its multi-platform manifest digest:

```bash
docker buildx imagetools inspect \
  docker.io/agustinramirodiaz/simpletimeservice:v1.0.3
```

Update `image_digest` in `terraform/terraform.tfvars` with the reported
top-level `Digest`, not one of the per-platform manifest digests listed under
`Manifests`. Cloud Run needs the multi-platform index digest so it can select
the `linux/amd64` image variant. The variable validation rejects tags and
malformed digests.

# Current design is a bit excessive

A custom load balancer and public IP is really not needed for this simple app, since we could simply use Google's Front End (GFE). But given that the requirements I've opted for using the load balancer with public static IP.

# Run Terraform from GitHub Actions

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

Example runs:

- Plan: https://github.com/AgustinRamiroDiaz/devops-challenge-senior/actions/runs/30821603018/job/91712613816?pr=1
- Apply: https://github.com/AgustinRamiroDiaz/devops-challenge-senior/actions/runs/30820696317/job/91711401365

## One-time Google Cloud bootstrap

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

Enable the APIs required for token exchange and for Terraform to read the
project and its billing association, then create the two service accounts:

```bash
gcloud services enable \
  cloudresourcemanager.googleapis.com \
  cloudbilling.googleapis.com \
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

Cloud Resource Manager and Cloud Billing are bootstrap dependencies. They must
be enabled manually because Terraform calls them while refreshing the
`google_project` resource, before it can plan any API-enablement resources.

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

| Setting                          | Type                              | Value for this deployment                                                                      |
| -------------------------------- | --------------------------------- | ---------------------------------------------------------------------------------------------- |
| `GCP_PROJECT_ID`                 | Variable                          | `agustinramirodiaz-cd1472`                                                                     |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Variable                          | `projects/449819592596/locations/global/workloadIdentityPools/github-actions/providers/github` |
| `GCP_PLAN_SERVICE_ACCOUNT`       | Variable                          | `terraform-plan@agustinramirodiaz-cd1472.iam.gserviceaccount.com`                              |
| `GCP_APPLY_SERVICE_ACCOUNT`      | `production` environment variable | `terraform-apply@agustinramirodiaz-cd1472.iam.gserviceaccount.com`                             |
| `GCP_BILLING_ACCOUNT_ID`         | Secret                            | Your GCP billing account ID                                                                    |
| `TF_STATE_BUCKET`                | Variable                          | `simple-time-service-tfstate`                                                                  |
| `TF_STATE_PREFIX`                | Variable                          | `simple-time-service`                                                                          |

The current repository already has these settings configured. Supplying the
billing account explicitly prevents CI from invoking the local `gcloud`
billing-account discovery fallback.
