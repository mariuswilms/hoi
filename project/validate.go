// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"fmt"
	"path/filepath"
	"strings"
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

	// Very simple TLD extractor. Domains only!
	TLD := func(domain string) string {
		if dot := strings.LastIndex(domain, "."); dot != -1 {
			return domain[dot+1:]
		}
		return ""
	}

	// TLD mustn't be "dev" outside dev contexts. Common neglect.
	if cfg.Context != ContextDevelopment {
		for _, v := range cfg.Domain {
			if TLD(v.FQDN) == "dev" {
				return fmt.Errorf(".dev TLD in %s context: %s", cfg.Context, v.FQDN)
			}
			for _, alias := range v.Aliases {
				if TLD(alias) == "dev" {
					return fmt.Errorf(".dev TLD in %s context in alias: %s", cfg.Context, alias)
				}
			}
			for _, redirect := range v.Redirects {
				if TLD(redirect) == "dev" {
					return fmt.Errorf(".dev TLD in %s context in redirect: %s", cfg.Context, redirect)
				}
			}
		}
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
	for _, v := range cfg.Domain {
		// Authentication
		//
		// Auth credentials setting must follow dedicated pattern.
		// Passwords mustn't differ for same users.
		// The credentials are stored in one single file per project.
		if v.Auth.User == "" && v.Auth.Password != "" {
			return fmt.Errorf("password set but user empty for domain. %s", v.FQDN)
		}
		if cfg.Context != ContextDevelopment && v.Auth.User != "" && v.Auth.Password == "" {
			return fmt.Errorf("user %s has empty password for domain: %s", v.Auth.User, v.FQDN)
		}
		if password, hasKey := creds[v.Auth.User]; hasKey {
			if password != v.Auth.Password {
				return fmt.Errorf("auth user %s given multiple times but with differing passwords for domain: %s", v.Auth.User, v.FQDN)
			}
		}
		creds[v.Auth.User] = v.Auth.Password

		// SSL
		//
		if v.SSL.CertificateKey != "" && v.SSL.Certificate != "" {
			if v.SSL.CertificateKey[0] == '!' || v.SSL.Certificate[0] == '!' {
				if v.SSL.CertificateKey != v.SSL.Certificate {
					return fmt.Errorf("cert and key indicate mix of special and regular action for domain: %s", v.FQDN)
				}
				if v.SSL.CertificateKey == v.SSL.Certificate && cfg.Context == ContextProduction {
					return fmt.Errorf("self-signed certs are not allowed in %s contexts, domain: %s", cfg.Context, v.FQDN)
				}
			} else if filepath.IsAbs(v.SSL.CertificateKey) || filepath.IsAbs(v.SSL.Certificate) {
				return fmt.Errorf("cert or key path is absolute, must be relative, domain: %s", v.FQDN)
			}
		} else if v.SSL.CertificateKey != "" || v.SSL.Certificate != "" {
			return fmt.Errorf("only cert or key set for domain: %s", v.FQDN)
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
