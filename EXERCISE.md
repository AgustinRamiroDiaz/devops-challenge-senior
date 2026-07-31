**NOTE**: Running this challenge on your account will create resources that are
billable. You are responsible for cleaning up your own account. Make sure you do
that to avoid unnecessary charges. We do not provide any reimbursement for any
costs incurred when you work through this challenge.

# Welcome to the Particle41 DevOps Team Challenge

This challenge is for candidates who want to join the Particle41 DevOps team.

It is designed to assess your level of familiarity with common modern
development and operations tools and concepts.

## Summary

We aim to hire software engineers who embrace the DevOps mindset, especially
taking an infrastructure-as-code approach to software and infrastructure
deployment.

This challenge is designed to evaluate your abilities in the following
technologies and concepts:

- Software development (in general), by creating an extremely minimal web
  service.
- Containers, including creating a container image from scratch and publishing
  it.
- Cloud infrastructure, including networking and compute resources on the cloud
  provider of your choice.
- Terraform/OpenTofu, including writing modules and using them to deploy
  infrastructure.
- Documentation, including a short blurb about the purpose/contents of your repo
  as well as simple deployment instructions.

This assessment asks you to create a small application, containerize it, and
deploy it end-to-end to a cloud provider of your choice using
Terraform/OpenTofu. There is an extra-credit section at the end for
overachievers.

## Documentation is MANDATORY

It is mandatory to include documentation for your repository explaining how to
use it.

Imagine that someone with less experience than you will need to clone your
repository and deploy your containerized application, or deploy your
Terraform/OpenTofu infrastructure.

With that in mind, you must provide all the instructions they will need to do
that successfully. These must include any prerequisites for deployment; mention
of needed tools and links to their installation pages; how to configure
credentials for the tool of your choice; and what commands to run for deploying
your code.

_We want to see your ability to properly document and communicate about your
work with the team._

- Add a `README.md` to the root directory of your project, with instructions for
  the team to deploy the projects you created. Include any notes (purpose, etc.)
  that you might add to the README if this were a real project.
- Publish your code to a public Git repository in a platform of your choice
  (e.g. GitHub, GitLab, Bitbucket, etc.), so that it can be cloned by the team.

## A word about generative AI

It is ok to use generative AI to complete this challenge, but we want to be sure
that you know what you're doing.

The acceptance criteria for the solution are clearly defined below. Regardless
of using generative AI, your solution must pass those acceptance criteria. If it
passes, we're ok with it and you'll move on to the next step in the selection
process.

So, this is our advice: if you use generative AI, make sure that your solution
works as expected and passes the criteria as explained below. Don't waste our
time (and yours!) submitting a solution that doesn't work.

---

# The Challenge

## Part 1 - Build and Containerize 'SimpleTimeService'

- Create a simple microservice (which we will call "SimpleTimeService") in any
  programming language of your choice: Go, NodeJS, Python, C#, Ruby, whatever
  you like.
- The application should be a web server that returns a pure JSON response with
  the following structure, when its `/` URL path is accessed:

```json
{
  "timestamp": "<current date and time>",
  "ip": "<the IP address of the visitor>"
}
```

### Dockerize SimpleTimeService

- Create a Dockerfile for this microservice.
- Your application MUST be configured to run as a non-root user in the
  container.
- Publish the image to a public container registry (for example, DockerHub) so
  we can pull it for testing.

## Part 2 - Deploy to the Cloud with Terraform/OpenTofu

Using Terraform or OpenTofu, deploy your containerized application to the
**cloud provider of your choice** (AWS, GCP, Azure, or any other major cloud
provider).

You are free to choose the architecture that best fits the task. Some examples
include, but are not limited to:

- A managed Kubernetes cluster (EKS, GKE, AKS) with your application deployed
  to it.
- A managed container service (ECS, Cloud Run, Azure Container Apps) running
  your application.
- A VM-based deployment with Docker installed.

Regardless of which architecture you choose, the following infrastructure
requirements apply:

- Your Terraform/OpenTofu code must create the networking layer (VPC or
  equivalent) with both public and private subnets/subnetworks.
- Your application's compute resources (containers, pods, VMs) must run in
  private subnets only.
- Your application must be reachable from the internet via a load balancer,
  gateway, or equivalent mechanism deployed in the public subnets.

You are free to use popular modules from the Terraform/OpenTofu registry to
accomplish this.

### Push your code to a public git repository

- Push your code to a public git repository in the platform of your choice (e.g.
  GitHub, GitLab, Bitbucket, etc.). MAKE SURE YOU DON'T PUSH ANY SECRETS LIKE
  API KEYS TO A PUBLIC REPO!
- We have a recommended repository structure [here](#suggested-repo-structure).

---

## Acceptance Criteria

### Part 1

- Your container image must be publicly available in a container registry.
- Your application must return the correct JSON response when accessed.
- Your container must run as a non-root user.
- Your container image should be as small as possible, without unnecessary bloat.

### Part 2

Your task will be considered successful if a colleague is able to deploy the
infrastructure to a cloud account and access your running application.

```sh
terraform plan # or tofu plan
```

and

```sh
terraform apply # or tofu apply
```

must be the only commands needed to deploy the full infrastructure and
application.

You MUST NOT commit any credentials to the git repository. Instead, provide
instructions in the README about how to authenticate to your chosen cloud
provider to deploy the infrastructure.

### General

- Documentation: you MUST add a `README` file with instructions to deploy your
  application and infrastructure.
- Code quality and style: your code must be easy for others to read, and
  properly documented when relevant.
- Terraform/OpenTofu best practices: Use variables in your infrastructure root
  module, and provide sensible defaults in a `terraform.tfvars` file.
- Justify your architecture choice in the README: briefly explain why you chose
  the architecture you did.

---

# Notes, Suggestions, and the opportunity to 'show off'!

## Suggested Repo Structure

```
.
├── app <-- app files/directories and Dockerfile go here
└── terraform <-- Terraform/OpenTofu files go here (i.e. we will run `terraform plan`/`terraform apply` from here)
```

## Extra Credit!

**THIS SECTION IS _COMPLETELY OPTIONAL_! THERE IS NO PENALTY FOR IGNORING
THIS!**

Are you an overachiever? Demonstrate your mastery of cloud-native IaC tooling by
doing any of these:

- Apply production-hardening best practices to your deployment (e.g. resource
  limits, health checks, auto-scaling, pod disruption budgets).
- Add a sidecar container of some kind, such as Fluent Bit.
- Initialize and use a remote Terraform/OpenTofu backend for state and locking
  instead of a local `.tfstate` file.
- Create a simple CI/CD pipeline (GitHub Actions, Bitbucket Pipelines, GitLab
  CI, etc.) to build and publish your container image, and commit the
  configuration to your solution repo.
- Anything else that might demonstrate that you know what's up.
