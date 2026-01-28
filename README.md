# ipmi-cert-pusher

Watches for TLS certificate updates (e.g. from cert-manager mounted as Kubernetes secrets) and pushes them to Supermicro IPMI BMCs using the [Supermicro Update Agent (SAA/SUM)](https://www.supermicro.com/en/solutions/management-software/supermicro-update-manager) CLI tool.

## How it works

1. Polls certificate files on a configurable interval (default 5m)
2. Compares the local certificate's SHA-256 fingerprint against what the BMC is currently serving over HTTPS
3. When a mismatch is detected, invokes SAA to push the new certificate and key to the BMC

Polling is used instead of filesystem watches because Kubernetes secret mounts use symlink swaps that break inotify.

Each poll cycle short-circuits early when possible: unchanged file modification time skips re-reading the cert, an unchanged fingerprint skips the TLS dial, and matching local/remote fingerprints skip the push.

## Configuration

```yaml
poll_interval: 5m          # how often to check for cert changes
tls_dial_timeout: 10s      # timeout for connecting to IPMI HTTPS
saa_binary: /opt/saa/saa   # path to SAA executable

servers:
  - name: server01
    ipmi_host: 192.168.1.11
    cert_path: /certs/server01/tls.crt
    key_path: /certs/server01/tls.key
    credentials:
      username_file: /credentials/server01/username
      password_file: /credentials/server01/password
```

See `config.example.yaml` for a full example.

Credentials are read from files on each push (not cached), so credential rotation is supported without restarting the process.

## Usage

```
ipmi-cert-pusher --config /path/to/config.yaml
```

The `--config` flag defaults to `/etc/ipmi-cert-pusher/config.yaml`.

Sends JSON structured logs to stdout. Shuts down gracefully on SIGTERM or SIGINT.

## Docker

The Dockerfile produces a minimal Debian-based image containing the Go binary and the SAA tool. The SAA download URL is configurable via build arg:

```
docker build -t ipmi-cert-pusher .
docker build --build-arg SAA_URL=https://example.com/saa.tar.gz -t ipmi-cert-pusher .
```

## Error handling

| Scenario | Behavior |
|---|---|
| Config missing/invalid | Fatal at startup |
| SAA binary not found | Fatal at startup |
| Cert/key file not found | Warn, skip server, retry next cycle |
| TLS dial fails | Warn, skip server (BMC may be rebooting) |
| SAA exits non-zero | Log error with stdout/stderr, retry next cycle |
| SAA hangs | Killed after 5 minute timeout |

## Dependencies

- Go 1.23+
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) (only external dependency)
- SAA/SUM CLI tool (bundled in Docker image)
