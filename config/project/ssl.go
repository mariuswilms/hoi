// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package project

import (
	"errors"
	"fmt"
	"path/filepath"
)

// Will use letsencrypt to get a valid cert and renew it automatically.
const CERT_ACME string = "!acme"

// Will generate a self-signed corp cert on the fly.
const CERT_OWNCA string = "!own-ca"

// Will generate a self-signed cert on the fly.
const CERT_SELFSIGNED string = "!self-signed"

// SSL is considered enabled, once a value for Certificate is given.
type SSLDirective struct {
	// Paths to certificate and certificate key. Paths must be relative to
	// project root i.e. config/ssl/example.org.crt.
	Certificate    string
	CertificateKey string
}

func (drv SSLDirective) IsEnabled() bool {
	return drv.Certificate != ""
}

func (drv SSLDirective) GetCertificate() (string, error) {
	switch drv.Certificate {
	case CERT_ACME:
		return "", errors.New("unimplemented")
	case CERT_OWNCA:
		return "", errors.New("unimplemented")
	case CERT_SELFSIGNED:
		return "", errors.New("unimplemented")
	default:
		if filepath.IsAbs(drv.Certificate) {
			return drv.Certificate, fmt.Errorf("cert has absolute path: %s", drv.Certificate)
		}
		return drv.Certificate, nil
	}
}

func (drv SSLDirective) GetCertificateKey() (string, error) {
	switch drv.CertificateKey {
	case CERT_ACME:
		return "", errors.New("unimplemented")
	case CERT_OWNCA:
		return "", errors.New("unimplemented")
	case CERT_SELFSIGNED:
		return "", errors.New("unimplemented")
	default:
		if filepath.IsAbs(drv.CertificateKey) {
			return drv.CertificateKey, fmt.Errorf("cert key has absolute path: %s", drv.CertificateKey)
		}
		return drv.CertificateKey, nil
	}
}
