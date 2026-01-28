package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval   time.Duration  `yaml:"poll_interval"`
	TLSDialTimeout time.Duration  `yaml:"tls_dial_timeout"`
	SAABinary      string         `yaml:"saa_binary"`
	Servers        []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Name        string          `yaml:"name"`
	IPMIHost    string          `yaml:"ipmi_host"`
	CertPath    string          `yaml:"cert_path"`
	KeyPath     string          `yaml:"key_path"`
	Credentials CredentialPaths `yaml:"credentials"`
}

type CredentialPaths struct {
	UsernameFile string `yaml:"username_file"`
	PasswordFile string `yaml:"password_file"`
}

// ReadCredentials reads username and password from their respective files.
// Reads fresh each invocation to support credential rotation.
func (c *CredentialPaths) ReadCredentials() (username, password string, err error) {
	u, err := os.ReadFile(c.UsernameFile)
	if err != nil {
		return "", "", fmt.Errorf("reading username file: %w", err)
	}
	p, err := os.ReadFile(c.PasswordFile)
	if err != nil {
		return "", "", fmt.Errorf("reading password file: %w", err)
	}
	return strings.TrimSpace(string(u)), strings.TrimSpace(string(p)), nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{
		PollInterval:   5 * time.Minute,
		TLSDialTimeout: 10 * time.Second,
		SAABinary:      "/opt/saa/saa",
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("at least one server must be configured")
	}

	if _, err := os.Stat(c.SAABinary); err != nil {
		return fmt.Errorf("SAA binary not found at %s: %w", c.SAABinary, err)
	}

	for i, s := range c.Servers {
		if s.Name == "" {
			return fmt.Errorf("server[%d]: name is required", i)
		}
		if s.IPMIHost == "" {
			return fmt.Errorf("server[%d] (%s): ipmi_host is required", i, s.Name)
		}
		if s.CertPath == "" {
			return fmt.Errorf("server[%d] (%s): cert_path is required", i, s.Name)
		}
		if s.KeyPath == "" {
			return fmt.Errorf("server[%d] (%s): key_path is required", i, s.Name)
		}
		if s.Credentials.UsernameFile == "" {
			return fmt.Errorf("server[%d] (%s): credentials.username_file is required", i, s.Name)
		}
		if s.Credentials.PasswordFile == "" {
			return fmt.Errorf("server[%d] (%s): credentials.password_file is required", i, s.Name)
		}
	}

	return nil
}
