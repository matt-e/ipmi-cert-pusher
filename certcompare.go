package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"time"
)

// FingerprintFromFile reads a PEM-encoded certificate file and returns its
// SHA-256 fingerprint and expiry time.
func FingerprintFromFile(path string) ([sha256.Size]byte, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [sha256.Size]byte{}, time.Time{}, fmt.Errorf("reading cert file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return [sha256.Size]byte{}, time.Time{}, fmt.Errorf("no PEM block found in %s", path)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return [sha256.Size]byte{}, time.Time{}, fmt.Errorf("parsing certificate: %w", err)
	}

	return sha256.Sum256(cert.Raw), cert.NotAfter, nil
}

// FingerprintFromRemote connects to host:443 via TLS and returns the SHA-256
// fingerprint and expiry time of the server's leaf certificate. Uses
// InsecureSkipVerify since BMC certs are self-signed or use internal CAs.
func FingerprintFromRemote(host string, timeout time.Duration) ([sha256.Size]byte, time.Time, error) {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp",
		net.JoinHostPort(host, "443"),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		return [sha256.Size]byte{}, time.Time{}, fmt.Errorf("TLS dial to %s: %w", host, err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return [sha256.Size]byte{}, time.Time{}, fmt.Errorf("no peer certificates from %s", host)
	}

	return sha256.Sum256(certs[0].Raw), certs[0].NotAfter, nil
}
