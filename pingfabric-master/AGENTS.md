# Repository Guidelines

## Project Structure & Module Organization
This repository is a small Go CLI that loads probe targets from MySQL, checks TCP connectivity, and writes Prometheus metrics to stdout. Keep runtime logic in [`main.go`](/Users/yi/workspace/code/go/pingfabric/main.go). Dependency metadata lives in `go.mod` and `go.sum`. Database access is currently configured against `metadata.cmdb_vip`. If the codebase grows, move reusable logic into package directories such as `internal/ping/` and keep `main.go` limited to wiring.

## Build, Test, and Development Commands
Use the standard Go toolchain:

- `PINGFABRIC_DB_PASSWORD=... go run .` runs the probe locally against MySQL.
- `go build ./...` compiles the module and catches package-level build errors.
- `go test ./...` runs all tests; the repo currently has no `_test.go` files, so contributors should add them with new behavior.
- `gofmt -w main.go` formats edited files before review.

Example: `PINGFABRIC_DB_PASSWORD=... go run . > metrics.prom` writes the generated Prometheus output to a file for inspection.

## Coding Style & Naming Conventions
Follow idiomatic Go. Use tabs via `gofmt`; do not hand-align indentation. Keep exported identifiers rare unless a package boundary requires them. Prefer short, descriptive names such as `pinger` or `workerCount`; avoid snake_case in Go code. Group constants by purpose and keep concurrency patterns explicit with `sync.WaitGroup`, buffered channels, and clear ownership of shared state.

## Testing Guidelines
Write table-driven tests with the standard `testing` package. Place tests next to the code they cover in files named `*_test.go`, for example `main_test.go`. Cover endpoint parsing, IPv6 normalization, timeout handling, and metric emission where practical. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
This repository currently has no commit history, so use concise imperative commit subjects such as `Add IPv6 endpoint parsing test`. Keep commits focused on one change. Pull requests should include a short description, the commands you ran for verification, and sample output when behavior changes affect emitted metrics.

## Configuration & Contributor Notes
Do not commit database passwords or other secrets. The current implementation expects the password from the `PINGFABRIC_DB_PASSWORD` environment variable. Avoid committing local build artifacts such as the compiled `pingfabric` binary.
