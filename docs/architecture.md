# knocker-cli Architecture

This document outlines the architecture of the `knocker-cli` project.

## High-Level Overview

`knocker-cli` is a Go-based CLI application designed to run as a background service on multiple platforms (Linux, macOS, and Windows). Its primary function is to monitor for changes in the machine's public IP address and, upon detection, make an API call to a remote server to whitelist the new IP.

The project is composed of several key components that work together to provide this functionality.



## Component Breakdown

### 1. CLI (Cobra)

The command-line interface is built using the **Cobra** library. It provides the main entry point for the user and exposes the following commands:

-   `knocker run`: Starts the service in the foreground or as a background daemon.
-   `knocker install`: Installs the service as a system daemon.
-   `knocker uninstall`: Uninstalls the system daemon.
-   `knocker start`: Starts the installed daemon.
-   `knocker stop`: Stops the installed daemon.
-   `knocker status`: Checks the status of the installed daemon.
-   `knocker knock`: Manually triggers an IP whitelist request.

### 2. Configuration (Viper)

Application configuration is managed by the **Viper** library. It allows for flexible configuration from a file (e.g., `.knocker.yaml`), environment variables, or command-line flags. This component is responsible for loading settings such as the API endpoint, API key, and the IP check interval.

### 3. Service Management (kardianos/service)

The **kardianos/service** library is used to create a cross-platform system service (daemon). It handles the complexities of running the application as a background process on different operating systems, including installation, starting, stopping, and uninstallation.

The core logic of the application is wrapped in a `program` struct that implements the `service.Interface`.

### 4. Core Service Logic

This is the heart of the application, located in the `internal/service` package. It contains the main loop that periodically performs the following actions:

1.  **Health Check**: It first checks the `/health` endpoint of the remote API to ensure it is available.
2.  **IP Detection & Knocking**: The service operates in one of two modes:
    *   **Simple Mode (Default):** If no `ip_check_url` is configured, the service sends a "knock" request to the API at each interval. It does not check its own IP. The remote API is responsible for identifying the client's IP from the request and updating the whitelist.
    *   **Comparison Mode (Optional):** If an `ip_check_url` is provided, the service first fetches its public IP from that URL. It compares this IP to the last known IP. If they are different, it then sends a "knock" request to the API to whitelist the new address.

### 5. API Client

A simple HTTP client, located in the `internal/api` package, is responsible for all communication with the remote Knocker API. It handles making requests to the `/health` and `/knock` endpoints and includes retry logic for transient network errors.

### 6. IP Utility

The `internal/util` package contains a utility for fetching the public IP address from external services. This is an optional feature, as the primary method of IP detection is handled by the remote API.

### 7. Build and Release (GoReleaser & Docker)

-   **GoReleaser**: The project uses GoReleaser to automate the build and release process. The `.goreleaser.yml` file defines how to build binaries for different platforms, create archives, and generate release notes.
-   **Docker**: A multi-stage `Dockerfile` is provided to create a minimal, containerized version of the application for easy deployment.

## How It Works: IP Change Detection

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