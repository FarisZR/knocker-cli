# Knocker Architecture

This document outlines the architecture of the Knocker CLI service.

## High-Level Overview

Knocker is a Go-based CLI application designed to run as a background service on multiple platforms (Linux, macOS, and Windows). Its primary function is to monitor for changes in the machine's public IP address and, upon detection, make an API call to a remote server to whitelist the new IP.

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

Application configuration is managed by the **Viper** library. It allows for flexible configuration from a file (e.g., `config.yaml`), environment variables, or command-line flags. This component is responsible for loading settings such as the API endpoint, API key, and the IP check interval.

### 3. Service Management (kardianos/service)

The **kardianos/service** library is used to create a cross-platform system service (daemon). It handles the complexities of running the application as a background process on different operating systems, including installation, starting, stopping, and uninstallation.

The core logic of the application is wrapped in a `program` struct that implements the `service.Interface`.

### 4. Core Service Logic

This is the heart of the application, located in the `internal/service` package. It contains the main loop that periodically performs the following actions:

1.  **Health Check**: It first checks the `/health` endpoint of the remote API to ensure it is available.
2.  **IP Detection**: It fetches the current public IP address. By default, Knocker relies on the remote API to determine the IP address from the incoming request. Optionally, it can be configured to use external IP checker services.
3.  **IP Comparison**: It compares the current IP with the previously recorded IP.
4.  **API Call**: If the IP has changed, it calls the `/knock` endpoint on the remote API to whitelist the new IP.

### 5. API Client

A simple HTTP client, located in the `internal/api` package, is responsible for all communication with the remote Knocker API. It handles making requests to the `/health` and `/knock` endpoints and includes retry logic for transient network errors.

### 6. IP Utility

The `internal/util` package contains a utility for fetching the public IP address from external services. This is an optional feature, as the primary method of IP detection is handled by the remote API.

### 7. Build and Release (GoReleaser & Docker)

-   **GoReleaser**: The project uses GoReleaser to automate the build and release process. The `.goreleaser.yml` file defines how to build binaries for different platforms, create archives, and generate release notes.
-   **Docker**: A multi-stage `Dockerfile` is provided to create a minimal, containerized version of the application for easy deployment.

## How It Works: IP Change Detection

1.  The background service starts and retrieves the initial public IP address by making a "knock" request to the API. The API records the IP of the machine making the request.
2.  The service stores this IP in memory.
3.  At a configurable interval (e.g., every 5 minutes), the service wakes up and performs a health check on the API.
4.  If the health check is successful, it makes another "knock" request.
5.  The remote API receives the request and can see the source IP. If this new IP is different from the one previously whitelisted for that client, the API updates its records.
6.  The Knocker service itself can also be configured to use an external IP checking service. In this mode, it fetches the IP from the external service and compares it to the last known IP. If it has changed, it then calls the `/knock` endpoint with the new IP. By default, this external check is disabled.