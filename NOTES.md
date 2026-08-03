# Decisions

## Use Google-managed serverless networking instead of a workload subnet

The exercise asks for public and private subnetworks, compute in a private
subnetwork, and a load balancer in the public networking layer. This deployment
follows the security intent of that requirement but not its literal topology.

Cloud Run and serverless NEGs are Google-managed regional resources; they are
not VMs or pods placed inside a customer VPC subnetwork. The regional external
Application Load Balancer uses the VPC's dedicated proxy-only subnet, while its
public forwarding rule owns the static external IP. The proxy-only subnet is a
special-purpose range for Google-managed proxies rather than a conventional
public workload subnet.

Cloud Run accepts ingress only from internal sources and Cloud Load Balancing,
its default `run.app` URL is disabled, and traffic from the serverless NEG to
the service follows Google-managed routing. These controls prevent clients from
bypassing the load balancer without requiring a private workload subnet or VPC
firewall rules.

Direct VPC egress would attach Cloud Run instances to a private subnet for
outbound connections to VPC resources. It would not place inbound application
traffic in that subnet or make the load-balancer-to-Cloud-Run path more private.
Because this service has no private VPC dependencies, adding that attachment
would create unused networking resources and IP consumption.

This is an intentional simplicity and platform-native design decision. It does,
however, mean the repository does not literally create both public and private
workload subnetworks as requested by the exercise. A strict topology-based
evaluation might require adding a private subnet and Direct VPC egress even
though they do not improve this service's ingress isolation.

## Flat terraform structure

Given that I don't have complex conditional logic, nor big reusable custom resources, I've opted out for a flat structure for all terraform resources. I've split them in files so that it's easier to mentally group and read.

Using a flat structure has the benefits of:

- less code and it's less error prone, since variables and outputs take a lot of space and are places where it's easy to missconfig.
- easier to read, since there are no custom modules with my own defined variables. By using standard terraform modules like `google`, everyone can read the resources and the code is familiar.

## Pin deployed images by digest

Cloud Run deploys the image using its immutable OCI manifest digest rather
than a mutable tag. The published `v1.0.7` multi-platform image currently
resolves to:

```text
docker.io/agustinramirodiaz/simpletimeservice@sha256:eb59a9d146a6a7347500052e9b577c090b971a3ec16156aba32583f41f6c04ac
```

Tags remain useful for publishing and discovering releases, but a registry
owner can move a tag to different content. A digest makes Terraform revisions
reproducible and ensures an apply deploys the image that was reviewed.

After publishing a new release, resolve its multi-platform manifest digest:

```bash
docker buildx imagetools inspect \
  docker.io/agustinramirodiaz/simpletimeservice:v1.0.7
```

Update `image_digest` in `terraform/terraform.tfvars` with the reported
top-level `Digest`, not one of the per-platform manifest digests listed under
`Manifests`. Cloud Run needs the multi-platform index digest so it can select
the `linux/amd64` image variant. The variable validation rejects tags and
malformed digests.

## OpenTelemetry Collector sidecar

The Cloud Run service includes an `otel-collector` sidecar to demonstrate a
real multi-container pattern without exposing another public endpoint. The app
container is still the only ingress container on port `8080`; it exports OTLP
metrics to `localhost:4317`, and the Collector batches and exports those
metrics to Google Cloud Managed Service for Prometheus.

The Collector config is stored in Secret Manager and mounted into the sidecar
as `/etc/otelcol-google/config.yaml`. The config is not secret, but Secret
Manager is the supported Cloud Run mechanism for mounting small configuration
files without building a second custom image. Because Terraform manages the
secret version, the config content is present in Terraform state.

The runtime service account only receives the permissions needed for this
pattern: `roles/secretmanager.secretAccessor` on the Collector config secret
and `roles/monitoring.metricWriter` on the project.

## Continuous-worker stress test

The stress test uses persistent workers that send requests continuously over
reused HTTP connections. This creates a steadier Cloud Run concurrency signal
than synchronized request rounds and avoids measuring a new TCP connection on
every request. The controller starts 80 workers, adds another 80 every five
seconds, and allows up to 1,000 workers by default. It cancels the load as soon
as responses identify three distinct Cloud Run instances. Transient transport,
HTTP 408, HTTP 429, and HTTP 5xx failures are counted but do not stop the load;
workers continue until the replica target or timeout is reached. Invalid client
IPs and malformed successful responses remain fatal because they violate the
test's correctness checks.

In the validation run, the service reached three instances after 12.1 seconds
with 240 active workers, 7,276 successful requests, no errors, and approximately
601 requests per second. The 1,000-worker ceiling was not reached; it remains a
safety limit that gives slower or fully cold deployments room to apply more
pressure before the 90-second timeout.

## Billing budget alerts

Terraform creates a monthly alert-only budget covering all costs attributed to
the project. `monthly_budget_amount` defaults to 10 whole units in the billing
account's currency. Project owners receive actual-spend alerts at 50%, 80%, and
100%, plus a forecasted-spend alert at 100%.

This budget is not an enforceable spending cap. Google's Preview Spend Cap
Budgets are currently configured through the Cloud Billing console and aren't
exposed by the public Budget API or Terraform provider. Keeping this budget
project-wide ensures that it includes Cloud Run, the load balancer, Monitoring,
Secret Manager, and other project costs.

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

gcloud projects add-iam-policy-binding "$APP_PROJECT_ID" \
  --member="serviceAccount:terraform-plan@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.viewer"

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
  roles/resourcemanager.projectIamAdmin \
  roles/monitoring.dashboardEditor \
  roles/secretmanager.admin \
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

gcloud billing accounts add-iam-policy-binding "$BILLING_ACCOUNT_ID" \
  --member="serviceAccount:terraform-apply@$APP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/billing.costsManager"
```

The `roles/resourcemanager.projectIamAdmin` grant lets Terraform attach the
runtime service account's `roles/monitoring.metricWriter` binding. The
`roles/secretmanager.admin` grant lets Terraform create and mount the
OpenTelemetry Collector configuration secret. The
`roles/monitoring.dashboardEditor` grant lets Terraform manage the application
dashboard without granting broader Monitoring administration permissions. The
`roles/billing.costsManager` grant lets it manage the project-scoped billing
budget; `roles/billing.user` is still required to attach the project to the
billing account.

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
