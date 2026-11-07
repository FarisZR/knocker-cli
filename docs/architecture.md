# knocker-cli Architecture

This document outlines the architecture of the `knocker-cli` project.

## High-Level Overview

`knocker-cli` is a Go-based CLI application designed to run as a background service on multiple platforms (Linux, macOS, and Windows). Its primary function is to monitor for changes in the machine's public IP address and, upon detection, make an API call to a remote server to whitelist the new IP.

The project is composed of several key components that work together to provide this functionality.



## Component Breakdown

### 1. CLI (Cobra)

The command-line interface is built using the **Cobra** library. It provides the main entry point for the user and exposes the following commands:

- `knocker run`: Starts the service in the foreground or as a background daemon.
- `knocker install`: Installs the service as a user-level daemon (systemd user unit or LaunchAgent).
- `knocker uninstall`: Uninstalls the user-level daemon.
- `knocker start`: Starts the installed user daemon.
- `knocker stop`: Stops the installed user daemon.
- `knocker status`: Checks the status of the installed user daemon.
- `knocker knock`: Manually triggers an IP whitelist request.

### 2. Configuration (Viper)

Application configuration is managed by the **Viper** library. It allows for flexible configuration from a file (e.g., `.knocker.yaml`), environment variables, or command-line flags. This component is responsible for loading settings such as the API endpoint, API key, and `watch_network` and the fallback status `check_interval`.

### 3. Service Management (kardianos/service)

The **kardianos/service** library is used to create a cross-platform system service (daemon). It handles the complexities of running the application as a background process on different operating systems, including installation, starting, stopping, and uninstallation.

The core logic of the application is wrapped in a `program` struct that implements the `service.Interface`.

On Linux the installer targets the user-level systemd instance, producing `~/.config/systemd/user/knocker.service` and driving it through `systemctl --user`. On macOS a LaunchAgent manifest is written to `~/Library/LaunchAgents/knocker.plist` so the service runs in the user's session.

### 4. Core Service Logic

The `internal/service` loop serializes status checks, knocks, and state changes.
Two independent schedules prevent network events from postponing renewal:

- A renewal timer starts from configured TTL and follows the server-returned TTL
  after a successful knock, refreshing at approximately 90% of TTL. Unknown TTL
  uses five minutes. Renewal runs in both modes, even without an IP change.
- With `watch_network` enabled, a separate status timer uses
  `check_interval`. Network events bring that check forward to one second away;
  further events cannot postpone an already pending check.
  Unknown status or a failed network knock retries after five seconds, doubling
  consecutive failure delays up to `check_interval` and resetting after a
  successful check. Network events still request fresh checks.

Simple mode knocks immediately at startup. Monitoring mode first checks status;
active startup coverage seeds state and renewal using its remaining TTL. For
subsequent checks, active coverage updates the observed state without changing
renewal. Inactive coverage triggers a knock and resets renewal only on success.
Failed status requests never imply inactive coverage. Scheduled renewal failures
retry after five seconds. The startup health probe is removed so transient
unavailability can recover through the service's retry paths.

### 5. API Client

`internal/api` implements `/health`, `/knock`, and authenticated `GET /status`.
Status uses `X-Api-Key` and validates the response's IP, explicit whitelist boolean,
and expiry metadata. Unknown or malformed responses produce errors, never an
implicit inactive result. Knocks omit the IP so the server resolves the source of
that request, including when connectivity changes between status and knock.
HTTP requests have a ten-second timeout. Status checks close idle connections to
avoid retaining a previous network's connection.

### 6. Network Detection

On Linux, a read-only `NETLINK_ROUTE` socket subscribes to link, IPv4/IPv6 address,
and route multicast groups without elevated privileges. A buffered notification
channel coalesces events. A bounded nonblocking socket poll permits cancellation
and socket cleanup at shutdown. Listener failures leave periodic status checking
available. Netlink receive-buffer overflow also requests a status check.

Other platforms compare interface flags and address snapshots every two seconds.
Periodic server status checks catch route-only and upstream changes. All local
listeners stop with the service. No third-party IP lookup participates in the
service; a nonempty legacy `ip_check_url` enables monitoring but is never fetched.


### 7. Build and Release (GoReleaser & Docker)

- **GoReleaser**: The project uses GoReleaser to automate the build and release process. The `.goreleaser.yml` file defines how to build binaries for different platforms, create archives, and generate release notes.
- **Homebrew tap**: The GoReleaser v2 config pins `project_name: knocker-cli`, preserving the Linux x86_64/arm64 archive names and SHA-256 checksums consumed by `FarisZR/homebrew-tap`. The release workflow builds the requested tag and dispatches the tap's sync workflow only after publishing succeeds. `TAP_GITHUB_TOKEN` needs only Actions read/write on the tap; the tap uses its own repository-scoped `GITHUB_TOKEN` to commit casks. Its scheduled sync also works without a PAT. Release builds do not run `go mod tidy` or change dependency files.
- **Docker**: A multi-stage `Dockerfile` is provided to create a minimal, containerized version of the application for easy deployment.

### 8. Structured Journald Events

Linux deployments run the service as a systemd user unit. To support UI integrations (e.g. a GNOME Shell extension) the runtime mirrors critical state changes to journald with structured metadata. The `internal/journald` package wraps `github.com/coreos/go-systemd/v22/journal` behind a small abstraction so non-Linux builds compile with a stub.

- Every structured entry carries both the human-readable message and a machine contract based on `KNOCKER_*` fields. The schema version is frozen at `KNOCKER_SCHEMA_VERSION=1`.
- The core service emits:
    - `ServiceState` when entering `started`, `stopping`, or `stopped` transitions (including `KNOCKER_VERSION`).
    - `StatusSnapshot` whenever material state changes (whitelist, TTL, next knock timestamp) so consumers can seed their UI.
    - `WhitelistApplied`, `WhitelistExpired`, `NextKnockUpdated`, and `KnockTriggered` as the whitelist lifecycle evolves.
    - `Error` whenever a problem (status check or knock) should surface in the UI, tagged with `KNOCKER_ERROR_CODE`.
- Manual invocations of `knocker knock` reuse the same contract, emitting `KnockTriggered` and `WhitelistApplied` events from the CLI path to keep consumers in sync even if the background service is idle.

Consumers can tail these events with `journalctl --user -u knocker.service KNOCKER_EVENT= -o json` and update their state using the accompanying structured fields.

## Network Status and Renewal

Enable monitoring with `watch_network: true`, `KNOCKER_WATCH_NETWORK=true`, or
`--watch_network`. The default remains periodic TTL-based renewal. Monitoring
requires the server's authenticated `/status` endpoint. Status includes dynamic
CIDR coverage and reports the connection's address family and route; it cannot
promise coverage for services reached over a different route or family.

Network-triggered knocks emit `KnockTriggered` with source `network`; failed status
checks emit `Error` with code `status_check_failed`. Observed existing coverage
updates `StatusSnapshot` with source `external` without emitting a fictitious
`WhitelistApplied` event or changing the next renewal deadline.
