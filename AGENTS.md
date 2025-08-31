# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Core Concepts & Non-Obvious Patterns

- **Configuration is Global**: Configuration is initialized in `cmd/knocker/main.go` using `cobra.OnInitialize` and is accessed globally via `viper.Get...()` calls. There is no central config struct passed around.
- **Service Logic is Decoupled**: The `kardianos/service` implementation is in the `program` struct in `cmd/knocker/program.go`. This is a thin wrapper that starts the main application logic, which resides entirely in `internal/service/service.go`.
- **Graceful Shutdown**: The service is stopped cleanly using a `quit` channel. This channel is created in `program.Start`, passed to the core service's `Run` method, and closed in `program.Stop`. This is the only way to terminate the service loop.
- **Conditional IP Detection**: The core function `checkAndKnock()` in `internal/service/service.go` has two distinct behaviors. If `ip_check_url` is an empty string in the config, it simply "knocks" on the API. Otherwise, it fetches and compares the IP before knocking. This is a critical conditional branch.
- **Shared Logger**: A single, shared logger for the `main` package is initialized in a `PersistentPreRun` function on the `rootCmd` in `cmd/knocker/main.go`. This makes the `logger` variable globally available to all command files (e.g., `run.go`, `knock.go`).

## Commands

- **Build**: `go build -o knocker ./cmd/knocker`
- **Test**: `go test ./...`
- **Install**: `go install ./...`
- **Run Foreground**: `knocker run`
- **Release (Cross-Platform)**: `goreleaser release --snapshot --clean`

## Development Workflow

- **Run Tests**: After any significant code change, you must run the test suite with `go test ./...` to ensure no regressions have been introduced.
- **Update Documentation**: If you add or modify a feature, you must update the relevant documentation in `README.md` and `docs/architecture.md` to reflect the changes.