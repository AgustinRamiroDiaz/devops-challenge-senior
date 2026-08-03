# Contributing

## Automated checks

The **Checks** GitHub Actions workflow runs on pushes to `main` and pull
requests targeting `main`. It checks Go formatting, runs `go vet` and the Go
tests, then checks Terraform formatting and validates the Terraform
configuration. Terraform CI does not run a plan or require GCP credentials.

## Local development

Run the Go service directly:

```bash
cd app
go test ./...
go run .
```

In another terminal:

```bash
curl http://localhost:8080/
```

Build or run the local container from the repository root:

```bash
make build
make run
make build-and-run
```

## Stress test

After deploying the current application image, run the production stress test
from the repository root. Pass the internet-facing IP that the service should
report for your machine or CI runner:

```bash
make stress-test EXPECTED_CLIENT_IP=203.0.113.10
```

The test reads the load-balancer URL and static IP from Terraform outputs. Each
worker continuously sends requests, reusing its HTTP connection, while the
controller starts 80 workers and adds another 80 every five seconds up to 1,000.
It stops after observing three Cloud Run instance IDs or reaching the 90-second
timeout. It validates response contents and reports throughput, errors,
per-instance request counts, and latency percentiles without enforcing a
latency threshold.

The load can be tuned with `STRESS_INITIAL_WORKERS`, `STRESS_WORKER_STEP`,
`STRESS_MAX_WORKERS`, `STRESS_RAMP_INTERVAL`, `STRESS_SCALE_TIMEOUT`, and
`EXPECTED_INSTANCES`.
