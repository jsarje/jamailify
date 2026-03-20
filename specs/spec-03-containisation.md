# Dockerization Specification

## Overview
Generate a production-ready `Dockerfile` and a `docker-compose.yml` file to containerize the Go jamailify application. The application now uses `github.com/glebarez/go-sqlite` (a pure Go implementation of SQLite), meaning CGO is disabled and the resulting image should be incredibly minimal. The deployment must ensure that the SQLite database and configuration files persist across container restarts.

## 1. Dockerfile Specifications
Use a **Multi-Stage Build** to keep the final image size as small as possible.

### Stage 1: The Builder
* **Base Image:** `golang:1.21-alpine` (or the latest available 1.x version).
* **Environment:** Explicitly set `CGO_ENABLED=0` and `GOOS=linux` to compile a statically linked binary. This is critical since we are no longer relying on C-based SQLite wrappers.
* **Action:** * Set `WORKDIR` to `/app`.
    * Copy `go.mod` and `go.sum`, then run `go mod download`.
    * Copy the rest of the source code.
    * Build the binary and name it `jamailify` (e.g., `go build -o jamailify .`).

### Stage 2: The Final Image
* **Base Image:** `alpine:latest`.
* **System Packages:** Install `ca-certificates` (required for TLS connections to POP3 and Google APIs) and `tzdata` (for accurate logging).
* **Directory Structure:** * Set `WORKDIR` to `/app`.
    * Create directories for volume mounts: `RUN mkdir -p /app/data /app/config`.
* **Artifacts:** Copy the compiled `jamailify` binary from the builder stage into `/app/`.
* **Execution:** Define the `CMD` or `ENTRYPOINT` to run the binary (`["./jamailify"]`).

## 2. Application Constraints for Docker
* **Database Path:** Ensure the Go application code is hardcoded (or uses a default environment variable) to write the SQLite database file to `/app/data/sync_state.db`.
* **Config Path:** Ensure the Go application looks for the configuration file at `/app/config/config.json` by default.

## 3. Docker Compose Specifications (`docker-compose.yml`)
Generate a standard compose file for easy deployment on a home server (like a Raspberry Pi, NAS, or Linux VPS).

* **Service Name:** `jamailify`
* **Build/Image:** Point the `build` context to the local directory containing the `Dockerfile`.
* **Restart Policy:** `unless-stopped` so the background daemon boots up automatically if the host machine restarts.
* **Volumes:**
    * Bind mount for the configuration: `./config:/app/config:ro` (Read-only is safer so the app can't accidentally overwrite its own config).
    * Bind mount for the database: `./data:/app/data:rw` (Read-write is required for SQLite).
* **Logging:** Configure sensible JSON logging limits so the background polling doesn't consume all the host's disk space over time (e.g., `max-size: "10m"`, `max-file: "3"`).