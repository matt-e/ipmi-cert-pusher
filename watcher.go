package main

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"os"
	"time"
)

type serverState struct {
	lastFingerprint [sha256.Size]byte
	lastModTime     time.Time
}

type Watcher struct {
	cfg   *Config
	state map[string]*serverState
}

func NewWatcher(cfg *Config) *Watcher {
	state := make(map[string]*serverState, len(cfg.Servers))
	for _, s := range cfg.Servers {
		state[s.Name] = &serverState{}
	}
	return &Watcher{cfg: cfg, state: state}
}

// Run performs an initial check then enters the polling loop until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	slog.Info("running initial certificate check")
	w.checkAll(ctx)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down watcher")
			return
		case <-ticker.C:
			w.checkAll(ctx)
		}
	}
}

func (w *Watcher) checkAll(ctx context.Context) {
	for _, server := range w.cfg.Servers {
		if ctx.Err() != nil {
			return
		}
		w.checkServer(ctx, server)
	}
}

func (w *Watcher) checkServer(ctx context.Context, server ServerConfig) {
	log := slog.With("server", server.Name, "host", server.IPMIHost)
	st := w.state[server.Name]

	// Step 1: Check cert file modification time.
	info, err := os.Stat(server.CertPath)
	if err != nil {
		log.Warn("cert file not accessible, skipping", "path", server.CertPath, "error", err)
		return
	}

	modTime := info.ModTime()
	if modTime.Equal(st.lastModTime) {
		log.Debug("cert file unchanged, skipping")
		return
	}

	// Step 2: Compute local fingerprint.
	localFP, err := FingerprintFromFile(server.CertPath)
	if err != nil {
		log.Warn("failed to read local cert, skipping", "error", err)
		return
	}

	if localFP == st.lastFingerprint && st.lastFingerprint != [sha256.Size]byte{} {
		log.Debug("local fingerprint unchanged, skipping")
		st.lastModTime = modTime
		return
	}

	// Step 3: Get remote fingerprint.
	remoteFP, err := FingerprintFromRemote(server.IPMIHost, w.cfg.TLSDialTimeout)
	if err != nil {
		log.Warn("failed to get remote cert, skipping (BMC may be rebooting)", "error", err)
		return
	}

	// Step 4: Compare fingerprints.
	if localFP == remoteFP {
		log.Info("certificates match, no push needed")
		st.lastFingerprint = localFP
		st.lastModTime = modTime
		return
	}

	log.Info("certificate mismatch detected, pushing new cert")

	// Step 5: Read credentials and push.
	username, password, err := server.Credentials.ReadCredentials()
	if err != nil {
		log.Error("failed to read credentials, skipping", "error", err)
		return
	}

	if err := RunSAA(ctx, w.cfg.SAABinary, server.IPMIHost, username, password, server.CertPath, server.KeyPath); err != nil {
		log.Error("SAA push failed", "error", err)
		return
	}

	log.Info("certificate pushed successfully")
	st.lastFingerprint = localFP
	st.lastModTime = modTime
}
