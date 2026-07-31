IMAGE_NAME ?= simple-time-service
IMAGE_TAG ?= latest
HOST_PORT ?= 8080
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7

IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build build-multi-platform run build-and-run

build:
	docker build --tag $(IMAGE) ./app

# Multi-platform images must be pushed because Docker cannot load a manifest
# containing multiple architectures into the classic local image store.
build-multi-platform:
	docker buildx build --platform $(PLATFORMS) --tag $(IMAGE) --push ./app

run:
	docker run --rm --publish $(HOST_PORT):8080 $(IMAGE)

build-and-run: build run
