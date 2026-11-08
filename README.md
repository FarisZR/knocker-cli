# knocker-cli

`knocker-cli` is a static Go CLI service that automatically requests a whitelist for the external IP of the device on IP address changes or when the whitelist expires. It runs in the background to ensure you always have access.

## Features

- **Automatic IP Whitelisting:** Automatically detects IP changes and requests a new whitelist.
- **Background Service:** Runs as a background service on Linux, macOS, and Windows.
- **Cross-Platform:** Built to be cross-platform with priority for Linux and macOS.
- **Docker Support:** Can be run in a Docker container.
- **Manual Whitelisting:** Manually trigger a whitelist request at any time.
- **Structured journald events:** Emits machine-readable journald events alongside human logs for desktop integrations.

## How It Works

`knocker-cli` operates in two distinct modes for handling IP changes:

### Simple Mode (Default)

The service knocks immediately on startup and refreshes at roughly 90% of the
server-returned TTL. If no TTL is known, it uses a five-minute cadence. The server
identifies the request's source IP.

### Network Monitoring (Optional)

Set `watch_network: true` (or `KNOCKER_WATCH_NETWORK=true`, or the
`--watch_network` flag). This requires a server with the authenticated
[`GET /status` endpoint](https://github.com/FarisZR/knocker/pull/117).

- At startup, check server status. If coverage is active, schedule renewal from
  the remaining TTL; otherwise knock immediately.
- On Linux, listen for kernel netlink interface, IPv4/IPv6 address, and route
  events, including disconnects, reconnects, and Ethernet/Wi-Fi or VPN changes.
  This listener requires no root privileges. Events are debounced for one second.
- On macOS and Windows, compare local interface/address snapshots every two
  seconds. Route-only changes are covered by periodic server checks.
- After a network change, query `/status` with the API key. If already whitelisted
  (including CIDR coverage), keep the renewal schedule unchanged. Otherwise knock
  without specifying an IP, then reset renewal from the knock response's TTL.
- Continue TTL-based renewal even when the IP does not change. `check_interval`
  independently controls periodic status checks, covering upstream public IP
  changes or missed local events.
- Failed status checks mean unknown coverage: do not knock based on that failure.
  Retry after five seconds, doubling consecutive failure delays up to
  `check_interval`; a successful status check resets the delay. Network events
  still trigger fresh checks. Failed scheduled
  renewals also retry after five seconds.

No third-party IP lookup is used. For existing configurations, a nonempty
`ip_check_url` enables this mode as a deprecated compatibility alias; its URL is
ignored. Servers without `/status` can continue using simple mode.

Status describes only the route and address family used by the API connection.

## Installation

### Homebrew on Linux

Install the prebuilt CLI on Linux x86_64 or ARM64 using Homebrew 6 or newer:

```bash
brew update
brew tap fariszr/tap
brew trust fariszr/tap
brew install --cask knocker-cli
knocker --help
```

Review the tap before granting trust. The package is named `knocker-cli` to
distinguish it from the core Knocker project; its executable remains `knocker`.
After tapping and trusting, the short package name works without a tap prefix.
To keep the CLI up to date:

```bash
brew update
brew upgrade --cask knocker-cli
```

The [Homebrew tap](https://github.com/FarisZR/homebrew-tap) follows published GitHub
releases automatically. It checks for releases every six hours; maintainers can
also configure immediate release notifications as described in the tap README.
Homebrew installs the CLI without starting its background service. See the
service installation instructions below to configure and start it.

### From Source

To install from source, you will need Go 1.26 or newer.

```bash
git clone https://github.com/FarisZR/knocker-cli.git
cd knocker-cli
go install ./...
```

### Using Docker

You can also run Knocker using Docker.

```bash
docker build -t knocker-cli .
docker run -d --name knocker-cli -e KNOCKER_API_URL=... -e KNOCKER_API_KEY=... knocker-cli
```

## Configuration

Knocker can be configured via a configuration file or environment variables.

### Configuration File

Create a file named `.knocker.yaml` in your home directory with the following content:

```yaml
api_url: "http://your-knocker-api-url"
api_key: "your-api-key"
watch_network: false # optional, check server status on network changes
check_interval: 5 # fallback server status check interval in minutes
ttl: 0 # optional, time to live in seconds for the knock request (0 for server default)
```

### Environment Variables

You can also configure `knocker-cli` using environment variables:

- `KNOCKER_API_URL`: The URL of the Knocker API.
- `KNOCKER_API_KEY`: Your API key.
- `KNOCKER_WATCH_NETWORK`: Enable network monitoring and server status checks (default `false`).
- `KNOCKER_CHECK_INTERVAL`: Fallback server status check interval in minutes (default `5`).
- `KNOCKER_IP_CHECK_URL`: Deprecated compatibility alias; any nonempty value enables network monitoring. The URL is ignored.
- `KNOCKER_TTL`: Optional time to live in seconds for the knock request (0 for server default).

When running as the packaged systemd user service, these variables can be placed in `~/.config/knocker/env` using the standard `KEY=value` format.

## Structured journald events

When the service runs under systemd (for example as `knocker.service`), every operational log is mirrored to journald with a human-friendly `MESSAGE` and a stable set of `KNOCKER_*` fields. These events let desktop integrations such as GNOME shell extensions consume Knocker state without polling a separate API.

- **Stream the events:** `journalctl --user -u knocker.service -o json -f | jq 'select(.KNOCKER_EVENT != null)'` to follow only structured entries, or pin to a specific type with `KNOCKER_EVENT=StatusSnapshot` as needed (`journalctl` only supports `FIELD=value` comparisons per its manual).
- **Schema version:** All entries include `KNOCKER_SCHEMA_VERSION=1` for forward compatibility.
- **Event types:**
  - `ServiceState` — lifecycle notifications (`started`, `stopping`, `stopped`) with optional `KNOCKER_VERSION`.
  - `StatusSnapshot` — current whitelist, TTL, and next scheduled knock.
  - `WhitelistApplied` / `WhitelistExpired` — whitelist changes with expiry metadata.
  - `NextKnockUpdated` — upcoming knock timestamp (or `0` when cleared).
  - `KnockTriggered` — manual (`cli`), scheduled (`schedule`), and network/status (`network`) knocks with success/failure result.
  - `Error` — surfaced issues that should be shown in the UI.

Manual invocations of `knocker knock` produce the same `KnockTriggered` and `WhitelistApplied` events so external consumers stay in sync even when the background service is idle.

## Usage

### Run as a foreground process

```bash
knocker run
```

### Manually trigger a whitelist

Even if the background service is running, you can manually trigger a whitelist request at any time.

```bash
knocker knock
```

### Install as a service

```bash
knocker install
```

> **Note:** On Linux the installer registers a per-user systemd unit at `~/.config/systemd/user/knocker.service`. Start it immediately with `systemctl --user enable --now knocker`. On macOS the installer writes `~/Library/LaunchAgents/knocker.plist`; load it with `launchctl bootstrap gui/$UID ~/Library/LaunchAgents/knocker.plist`.

### Start the installed service

```bash
knocker start
```

### Stop the running service

```bash
knocker stop
```

### Uninstall the service

```bash
knocker uninstall
```

### Check service status

```bash
knocker status
```

## Development

### Building

To build the binary for your current platform, run:

```bash
go build -o knocker ./cmd/knocker
```

### Cross-Platform Releases (with GoReleaser)

To create cross-platform builds, archives, and releases, you can use [GoReleaser](https://goreleaser.com/).

```bash
# This will create builds for all platforms defined in .goreleaser.yml
goreleaser release --snapshot --clean
```

### Local server integration test

With `uv`, Python 3, Go, and a current checkout of the Knocker server available:

```bash
python3 scripts/e2e.py /path/to/knocker
```

This starts a disposable server on loopback with firewall integration disabled and
an isolated whitelist. It tests CIDR coverage, network changes, TTL renewal, and
recovery after a simulated disconnect, then removes the test data. Source changes
are simulated using forwarded addresses from the trusted local test transport;
it does not switch your machine's adapters.
