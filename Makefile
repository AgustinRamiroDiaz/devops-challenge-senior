IMAGE_NAME ?= simple-time-service
IMAGE_TAG ?= latest
HOST_PORT ?= 8080
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7
DOCKERHUB_REPOSITORY ?= agustinramirodiaz/simpletimeservice
TERRAFORM_DIR ?= terraform
GCS_PREFIX ?= simple-time-service
TF_INIT_FLAGS ?=

IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)
TF_BACKEND_OVERRIDE := $(TERRAFORM_DIR)/backend_override.tf

.PHONY: build build-multi-platform run build-and-run publish terraform-init terraform-init-gcs terraform-backend-local terraform-backend-gcs

build:
	docker build --tag $(IMAGE) ./app

# Multi-platform images must be pushed because Docker cannot load a manifest
# containing multiple architectures into the classic local image store.
build-multi-platform:
	docker buildx build --platform $(PLATFORMS) --tag $(IMAGE) --push ./app

publish:
	@test -n "$(TAG)" || (echo "TAG is required. Usage: make publish TAG=v1.0.0"; exit 1)
	docker buildx build --platform $(PLATFORMS) --tag $(DOCKERHUB_REPOSITORY):$(TAG) --push ./app

run:
	docker run --rm --publish $(HOST_PORT):8080 $(IMAGE)

build-and-run: build run

# The absence of a backend block makes Terraform use its local backend.
terraform-backend-local:
	rm -f "$(TF_BACKEND_OVERRIDE)"

terraform-init: terraform-backend-local
	terraform -chdir="$(TERRAFORM_DIR)" init -migrate-state $(TF_INIT_FLAGS)

terraform-backend-gcs:
	@test -n "$(GCS_BUCKET)" || (echo "GCS_BUCKET is required. Usage: make terraform-init-gcs GCS_BUCKET=my-terraform-state-bucket [GCS_PREFIX=env/dev]"; exit 1)
	@printf '%s\n' \
		'terraform {' \
		'  backend "gcs" {' \
		'    bucket = "$(GCS_BUCKET)"' \
		'    prefix = "$(GCS_PREFIX)"' \
		'  }' \
		'}' > "$(TF_BACKEND_OVERRIDE)"

terraform-init-gcs: terraform-backend-gcs
	terraform -chdir="$(TERRAFORM_DIR)" init -migrate-state $(TF_INIT_FLAGS)
