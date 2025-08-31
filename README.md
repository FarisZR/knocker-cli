# Knocker CLI

Knocker is a static Go CLI service that automatically requests a whitelist for the external IP of the device on IP address changes or when the whitelist expires. It runs in the background to ensure you always have access.

## Features

- **Automatic IP Whitelisting:** Automatically detects IP changes and requests a new whitelist.
- **Background Service:** Runs as a background service on Linux, macOS, and Windows.
- **Cross-Platform:** Built to be cross-platform with priority for Linux and macOS.
- **Docker Support:** Can be run in a Docker container.
- **Manual Whitelisting:** Manually trigger a whitelist request at any time.

## How it Works

Knocker can detect IP changes in two ways:

1.  **API-based Detection (Default):** By default, Knocker relies on the remote API to detect the public IP address. When the `knocker run` service is active, it periodically sends a "knock" request to the API. The API server then uses the source IP of that request as the address to be whitelisted. This is the simplest and recommended method.

2.  **External IP Check (Optional):** For more advanced scenarios, Knocker can be configured to use external public IP checking services. In this mode, it will fetch the IP from a third-party service and compare it to the last known IP. If a change is detected, it will then send a request to the Knocker API to whitelist the new IP.

## Installation

### From Source

To install from source, you will need to have Go installed.

```bash
git clone https://github.com/FarisZR/knocker-cli.git
cd knocker-cli
make install
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
```

### Environment Variables

You can also configure Knocker using environment variables:

- `KNOCKER_API_URL`: The URL of the Knocker API.
- `KNOCKER_API_KEY`: Your API key.

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

### Check service status

```bash
knocker status
```

## Development

### Building

To build the binaries for all supported platforms, run:

```bash
make build
```

### Cleaning

To clean up the build artifacts, run:

```bash
make clean