package asr

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

func loadKeyPair(certPath, keyPath string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("asr mtls: load %s/%s: %w", certPath, keyPath, err)
	}
	return cert, nil
}

func loadCertPool(caPath string) (*x509.CertPool, error) {
	raw, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("asr mtls: read CA %s: %w", caPath, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, errors.New("asr mtls: CA file did not contain PEM certificates")
	}
	return pool, nil
}

func buildTLSConfig(cert tls.Certificate, pool *x509.CertPool, serverName string) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}
}
