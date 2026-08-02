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
