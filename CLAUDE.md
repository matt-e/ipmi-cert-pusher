# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ipmi-cert-pusher is a Go application that watches TLS certificate files (typically from Kubernetes cert-manager secret mounts) and automatically pushes them to Supermicro IPMI BMCs using the Supermicro Update Agent (SAA) CLI tool. It uses polling (not inotify) because Kubernetes secret mounts use symlink swaps that break file watchers.

## Build & Development Commands

```bash
# Build
go build -v ./...

# Test (with race detection, matching CI)
go test -v -race ./...

# Lint
gofmt -l .          # check formatting
go vet ./...        # static analysis
golangci-lint run   # full lint suite (CI uses v1.64.8)

# Format
gofmt -w .

# Docker build
docker build -t ipmi-cert-pusher .
```

## Architecture

All source is in a flat `package main` (~450 lines across 6 files):

- **main.go** — Entrypoint. Parses `--config` flag, sets up slog JSON logging, starts Prometheus metrics server on `:8080`, runs the watcher with graceful shutdown on SIGTERM/SIGINT.
- **config.go** — YAML config loading and validation. `Config` holds global settings (poll_interval, tls_dial_timeout, saa_binary) and a list of `ServerConfig` entries. Credentials are file paths read fresh on each push to support rotation.
- **watcher.go** — Core polling loop. For each server per cycle: (1) stat cert file, skip if mtime unchanged; (2) compute local SHA-256 fingerprint, skip if unchanged; (3) TLS dial BMC to get remote fingerprint; (4) compare; (5) read credentials from files and invoke SAA if mismatch.
- **certcompare.go** — `FingerprintFromFile()` and `FingerprintFromRemote()` compute SHA-256 fingerprints of X.509 certificates. Remote uses `InsecureSkipVerify` because BMC certs are self-signed.
- **saa.go** — Generates a temporary XML config file and invokes the SAA CLI (`saa -u USER -p PASS -c ChangeBmcCfg -i HOST --file CONFIG.xml`) with a 5-minute timeout.
- **metrics.go** — Prometheus metrics: check requests/errors counters, check duration histogram, push counter, certificate expiry gauge.

## Key Design Decisions

- **Polling over file watching**: Kubernetes secret mounts use symlink swaps incompatible with inotify.
- **Three-level short-circuit**: Each poll skips unnecessary work (mtime check → local fingerprint check → remote fingerprint comparison) before pushing.
- **Credentials read on demand**: Username/password files are read fresh each push cycle, not cached, to support credential rotation without restart.
- **Config validation at startup**: Missing SAA binary or invalid config causes immediate fatal exit rather than runtime failures.

## Deployment

Kubernetes manifests use Kustomize. The deployment mounts a ConfigMap (from config.yaml), TLS secrets from cert-manager, and a credentials secret. A ServiceMonitor configures Prometheus scraping on port 8080.
