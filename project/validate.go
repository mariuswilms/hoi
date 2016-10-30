// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"fmt"
	"path/filepath"
)

// Validates several aspects and looks for typical human errors. This
// must operate over the whole configuration and cannot be split into
// multiple validation methods per directive, as cross-directive
// information is often needed to determine actual validity.
func (cfg Config) Validate() error {
	stringInSlice := func(a string, list []string) bool {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	// Basic
	//
	// Must have context, we can't autodetect this.
	if cfg.Context == ContextUnknown {
		return fmt.Errorf("project has no context: %s", cfg.Path)
	}

	if cfg.Webroot == "" {
		return fmt.Errorf("project has no webroot: %s", cfg.Path)
	} else if filepath.IsAbs(cfg.Webroot) {
		return fmt.Errorf("webroot must not be absolute: %s", cfg.Webroot)
	}

	creds := make(map[string]string)
	for k, v := range cfg.Domain {
		// Authentication
		//
		// Auth credentials should be complete and not vary passwords between
		// same users. The credentials are stored in one single file per project.
		if !v.Auth.IsEnabled() {
			continue
		}
		if v.Auth.User == "" {
			return fmt.Errorf("empty user for domain: %s", v.FQDN)
		}
		if cfg.Context != ContextDevelopment && v.Auth.Password == "" {
			return fmt.Errorf("user %s has empty password for domain: %s", v.Auth.User, v.FQDN)
		}
		if _, hasKey := creds[k]; hasKey {
			if creds[k] == v.Auth.Password {
				return fmt.Errorf("auth user %s given multiple times but with differing passwords for domain: %s", v.Auth.User, v.FQDN)
			}
		}
		creds[v.Auth.User] = v.Auth.Password

		// SSL
		//
		if v.SSL.IsEnabled() {
			if v.SSL.CertificateKey == "" {
				return fmt.Errorf("SSL enabled but no certificate key for domain: %s", v.FQDN)
			}
			if string(v.SSL.CertificateKey[0]) == "!" {
				if v.SSL.Certificate != v.SSL.CertificateKey {
					return fmt.Errorf("special action requested for key but not for cert: %s != %s", v.SSL.Certificate, v.SSL.CertificateKey)
				}
			} else {
				if filepath.IsAbs(v.SSL.CertificateKey) {
					return fmt.Errorf("certificate key path is not relative: %s", v.SSL.CertificateKey)
				}
			}

			if v.SSL.Certificate == "" {
				return fmt.Errorf("SSL enabled but no certificate for domain: %s", v.FQDN)
			}
			if string(v.SSL.Certificate[0]) == "!" {
				if v.SSL.CertificateKey != v.SSL.Certificate {
					return fmt.Errorf("special action requested for cert but not for key: %s != %s", v.SSL.Certificate, v.SSL.CertificateKey)
				}
				if cfg.Context == ContextProduction && v.SSL.Certificate == CertSelfSigned {
					return fmt.Errorf("self-signed certs are not allowed in %s contexts", cfg.Context)
				}
			} else {
				if filepath.IsAbs(v.SSL.Certificate) {
					return fmt.Errorf("certificate path is not relative: %s", v.SSL.Certificate)
				}
			}
		}
	}

	// Database
	//
	// Database names must be unique and users should for security reasons not
	// have an empty password (not even for dev contexts).
	seenDatabases := make([]string, 0)
	for _, db := range cfg.Database {
		if db.Name == "" {
			return fmt.Errorf("found empty database name")
		}
		if stringInSlice(db.Name, seenDatabases) {
			return fmt.Errorf("found duplicate database name: %s", db.Name)
		}
		if cfg.Context != ContextDevelopment && db.Password == "" {
			return fmt.Errorf("user %s has empty password for database: %s", db.User, db.Name)
		}
		seenDatabases = append(seenDatabases, db.Name)
	}

	// Volume
	//
	for _, volume := range cfg.Volume {
		if filepath.IsAbs(volume.Path) {
			return fmt.Errorf("volume path is not relative: %s", volume.Path)
		}
	}

	return nil
}
