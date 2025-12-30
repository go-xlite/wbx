package weblite

import (
	"crypto/tls"
	"fmt"
	"sync"
)

type SSL struct {
	CertFilePath string
	KeyFilePath  string
	// Raw SSL/TLS data (alternative to file paths)
	CertData []byte
	KeyData  []byte
	// Multi-domain certificate support
	Certificates map[string]*tls.Certificate // domain -> certificate
	mu           sync.RWMutex
}

// SetFromFiles configures SSL/TLS using file paths (single certificate for all domains)
func (s *SSL) SetFromFiles(certFile, keyFile string) {
	s.CertFilePath = certFile
	s.KeyFilePath = keyFile
	s.CertData = nil
	s.KeyData = nil
}

// AddCertificateForDomain adds a certificate for a specific domain
func (s *SSL) AddCertificateForDomain(domain, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate for domain %s: %w", domain, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Certificates == nil {
		s.Certificates = make(map[string]*tls.Certificate)
	}
	s.Certificates[domain] = &cert
	return nil
}

// AddCertificateForDomainFromData adds a certificate for a specific domain from raw data
func (s *SSL) AddCertificateForDomainFromData(domain string, certData, keyData []byte) error {
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return fmt.Errorf("failed to parse certificate for domain %s: %w", domain, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Certificates == nil {
		s.Certificates = make(map[string]*tls.Certificate)
	}
	s.Certificates[domain] = &cert
	return nil
}

// SetFromData configures SSL/TLS using raw certificate and key data
func (s *SSL) SetFromData(certData, keyData []byte) {
	s.CertData = certData
	s.KeyData = keyData
	s.CertFilePath = ""
	s.KeyFilePath = ""
}

// SetFromText configures SSL/TLS using raw certificate and key text
func (s *SSL) SetFromText(certText, keyText string) {
	s.SetFromData([]byte(certText), []byte(keyText))
}

// IsConfigured returns true if SSL is configured (either from files or data)
func (s *SSL) IsConfigured() bool {
	return s.HasData() || s.HasFiles()
}

// HasData returns true if SSL is configured from raw data
func (s *SSL) HasData() bool {
	return len(s.CertData) > 0 && len(s.KeyData) > 0
}

// HasFiles returns true if SSL is configured from file paths
func (s *SSL) HasFiles() bool {
	return s.CertFilePath != "" && s.KeyFilePath != ""
}

// GetTLSConfig creates and returns a TLS configuration with SNI support
func (s *SSL) GetTLSConfig() (*tls.Config, error) {
	s.mu.RLock()
	hasDomainCerts := len(s.Certificates) > 0
	s.mu.RUnlock()

	// If we have domain-specific certificates, use SNI
	if hasDomainCerts {
		config := &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				s.mu.RLock()
				defer s.mu.RUnlock()

				// Try exact match first
				if cert, ok := s.Certificates[hello.ServerName]; ok {
					return cert, nil
				}

				// Try wildcard match
				for domain, cert := range s.Certificates {
					if matchWildcardDomain(domain, hello.ServerName) {
						return cert, nil
					}
				}

				// Fall back to default certificate if available
				if s.HasData() {
					cert, err := tls.X509KeyPair(s.CertData, s.KeyData)
					if err == nil {
						return &cert, nil
					}
				}

				if s.HasFiles() {
					cert, err := tls.LoadX509KeyPair(s.CertFilePath, s.KeyFilePath)
					if err == nil {
						return &cert, nil
					}
				}

				return nil, fmt.Errorf("no certificate found for %s", hello.ServerName)
			},
		}
		return config, nil
	}

	// Single certificate configuration
	if s.HasData() {
		cert, err := tls.X509KeyPair(s.CertData, s.KeyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate and key: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}, nil
	}

	if s.HasFiles() {
		cert, err := tls.LoadX509KeyPair(s.CertFilePath, s.KeyFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate and key: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}, nil
	}

	return nil, fmt.Errorf("SSL not configured")
}
