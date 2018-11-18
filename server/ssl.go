// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"path/filepath"
)

type SSLDirective struct {
	Enabled bool
	RunPath string
	// A list of system certificates and keys.
	System map[string]SystemSSLDirective
}

type SystemSSLDirective struct {
	// Shell file name pattern the FQDN will be matched against.
	Pattern string
	// Absolute paths to certificate and key.
	Certificate    string
	CertificateKey string
}

// Iterates through available system certificates and finds one that
// matches the given domain.
func (drv SSLDirective) GetSystemCertificate(domain string) (string, error) {
	for _, v := range drv.System {
		matched, err := filepath.Match(v.Pattern, domain)
		if err != nil {
			return "", fmt.Errorf("Failed to match system certificate for FQDN %s: %s", domain, err)
		}
		if matched {
			return v.Certificate, nil
		}
	}
	return "", fmt.Errorf("No system certificate found for FQDN %s", domain)
}

// Pendant to GetSystemCertificate().
func (drv SSLDirective) GetSystemCertificateKey(domain string) (string, error) {
	for _, v := range drv.System {
		matched, err := filepath.Match(v.Pattern, domain)
		if err != nil {
			return "", fmt.Errorf("Failed to match system certificate key for FQDN %s: %s", domain, err)
		}
		if matched {
			return v.CertificateKey, nil
		}
	}
	return "", fmt.Errorf("No system certificate key found for FQDN %s", domain)
}
