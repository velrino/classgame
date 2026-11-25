package redis

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

func buildTLSConfig(input Options) (*tls.Config, error) {
	if strings.TrimSpace(input.CACertPath) == "" && strings.TrimSpace(input.CACertPEM) == "" {
		return nil, nil
	}

	if strings.TrimSpace(input.CACertPath) != "" && strings.TrimSpace(input.CACertPEM) != "" {
		return nil, fmt.Errorf("configure either redis ca cert path or pem, not both")
	}

	certPEM, err := readCACertPEM(input)
	if err != nil {
		return nil, err
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if ok := rootCAs.AppendCertsFromPEM(certPEM); !ok {
		return nil, fmt.Errorf("parse redis ca certificate")
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}, nil
}

func readCACertPEM(input Options) ([]byte, error) {
	if strings.TrimSpace(input.CACertPEM) != "" {
		return []byte(input.CACertPEM), nil
	}

	certPEM, err := os.ReadFile(input.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("read redis ca certificate: %w", err)
	}

	return certPEM, nil
}

func mergeTLSConfig(current *tls.Config, custom *tls.Config) *tls.Config {
	if current == nil {
		return custom
	}

	merged := current.Clone()
	merged.RootCAs = custom.RootCAs
	if merged.MinVersion == 0 {
		merged.MinVersion = custom.MinVersion
	}

	return merged
}
