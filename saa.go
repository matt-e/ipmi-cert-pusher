package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

const saaTimeout = 5 * time.Minute

const saaXMLTemplate = `<?xml version="1.0"?>
<BmcCfg>
  <OemCfg Action="Change">
    <Certification Action="Change">
      <Configuration>
        <CertFile>%s</CertFile>
        <PrivKeyFile>%s</PrivKeyFile>
      </Configuration>
    </Certification>
  </OemCfg>
</BmcCfg>
`

// RunSAA generates a temporary XML config and invokes the SAA CLI to push a certificate
// to a BMC. The context should carry a cancellation signal for graceful shutdown; an
// additional 5-minute timeout is applied for the SAA process itself.
func RunSAA(ctx context.Context, saaBinary, host, username, password, certPath, keyPath string) error {
	xmlContent := fmt.Sprintf(saaXMLTemplate, certPath, keyPath)

	tmpFile, err := os.CreateTemp("", "saa-config-*.xml")
	if err != nil {
		return fmt.Errorf("creating temp XML file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(xmlContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp XML file: %w", err)
	}
	tmpFile.Close()

	cmdCtx, cancel := context.WithTimeout(ctx, saaTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, saaBinary,
		"-u", username,
		"-p", password,
		"-c", "ChangeBmcCfg",
		"-i", host,
		"--file", tmpPath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Info("invoking SAA", "host", host, "user", username)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SAA failed for %s: %w\nstdout: %s\nstderr: %s",
			host, err, stdout.String(), stderr.String())
	}

	slog.Info("SAA completed successfully", "host", host, "stdout", stdout.String())
	return nil
}
