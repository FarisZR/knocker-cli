# knocker-cli

`knocker-cli` is a static Go CLI service that automatically requests a whitelist for the external IP of the device on IP address changes or when the whitelist expires. It runs in the background to ensure you always have access.

## Features

- **Automatic IP Whitelisting:** Automatically detects IP changes and requests a new whitelist.
- **Background Service:** Runs as a background service on Linux, macOS, and Windows.
- **Cross-Platform:** Built to be cross-platform with priority for Linux and macOS.
- **Docker Support:** Can be run in a Docker container.
- **Manual Whitelisting:** Manually trigger a whitelist request at any time.

## How It Works

`knocker-cli` operates in two distinct modes for handling IP changes:

### Simple Mode (Default)

This is the default and recommended mode of operation.

1.  The `knocker-cli` service starts and, at a regular interval (configured by the `interval` setting), sends a "knock" request to your API server.
2.  The service **does not** check its own IP address.
3.  Your API server receives the "knock" request, inspects the source IP address of the request, and updates its whitelist accordingly.

In this mode, `knocker-cli` acts as a simple, periodic "pinger" to keep your whitelist entry fresh.

### Comparison Mode (Optional)

This mode is for more advanced use cases where you want the client to be responsible for detecting IP changes.

1.  To enable this mode, you must provide an `ip_check_url` in your configuration. This URL should point to a service that returns the client's public IP in plain text (e.g., `https://ifconfig.me`).
2.  The `knocker-cli` service starts and fetches its public IP from the `ip_check_url`. It stores this IP in memory.
3.  At each interval, it fetches the IP again.
4.  It compares the new IP with the one stored in memory.
5.  If the IP address has changed, and only if it has changed, the service will send a "knock" request to your API server to whitelist the new address.

## Installation

### From Source

To install from source, you will need to have Go installed.

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
interval: 5 # in minutes
ip_check_url: "" # optional, e.g. "https://ifconfig.me"
ttl: 0 # optional, time to live in seconds for the knock request (0 for server default)
```

### Environment Variables

You can also configure `knocker-cli` using environment variables:

- `KNOCKER_API_URL`: The URL of the Knocker API.
- `KNOCKER_API_KEY`: Your API key.
- `KNOCKER_INTERVAL`: The interval in minutes to check for IP changes.
- `KNOCKER_IP_CHECK_URL`: Optional URL of the external IP checker service.
- `KNOCKER_TTL`: Optional time to live in seconds for the knock request (0 for server default).

When running as the packaged systemd user service, these variables can be placed in `~/.config/knocker/env` using the standard `KEY=value` format.

Setting a non-zero `ttl` automatically shortens the knock interval so that a knock is issued when roughly 90% of the TTL has elapsed (leaving a 10% buffer before expiry) while never exceeding the configured interval.

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
