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
	if err := cfg.validateBasics(); err != nil {
		return err
	}
	if cfg.Context != ContextDevelopment {
		if err := cfg.validateDomainsHaveNoTestTLD(); err != nil {
			return err
		}
		if err := cfg.validateDomainsAreUsedOnce(); err != nil {
			return err
		}
	}
	if err := cfg.validateDomainsAuth(); err != nil {
		return err
	}
	if err := cfg.validateDomainsSSL(); err != nil {
		return err
	}
	if err := cfg.validateDatabases(); err != nil {
		return err
	}
	if err := cfg.validateVolumes(); err != nil {
		return err
	}
	return nil
}

// Must have context, we can't autodetect this.
func (cfg Config) validateBasics() error {
	if cfg.Context == ContextUnknown {
		return fmt.Errorf("project has no context: %s", cfg.Path)
	}
	if cfg.Webroot == "" {
		return fmt.Errorf("project has no webroot: %s", cfg.Path)
	} else if filepath.IsAbs(cfg.Webroot) {
		return fmt.Errorf("webroot must not be absolute: %s", cfg.Webroot)
	}
	return nil
}

// TLD mustn't be "test" outside dev contexts. Common neglect.
func (cfg Config) validateDomainsHaveNoTestTLD() error {
	// Very simple TLD extractor. Domains only!
	TLD := func(domain string) string {
		if dot := strings.LastIndex(domain, "."); dot != -1 {
			return domain[dot+1:]
		}
		return ""
	}

	for _, v := range cfg.Domain {
		if TLD(v.FQDN) == "test" {
			return fmt.Errorf("test TLD in %s context: %s", cfg.Context, v.FQDN)
		}
		for _, alias := range v.Aliases {
			if TLD(alias) == "test" {
				return fmt.Errorf("test TLD in %s context in alias: %s", cfg.Context, alias)
			}
		}
		for _, redirect := range v.Redirects {
			if TLD(redirect) == "test" {
				return fmt.Errorf("test TLD in %s context in redirect: %s", cfg.Context, redirect)
			}
		}
	}
	return nil
}

func (cfg Config) validateDomainsAreUsedOnce() error {
	mainSeen := map[string]bool{}

	for _, v := range cfg.Domain {
		if _, ok := mainSeen[v.FQDN]; ok {
			return fmt.Errorf("multiple domains for %s", v.FQDN)
		}
		blockSeen := map[string]bool{v.FQDN: true}

		for _, alias := range v.Aliases {
			if _, ok := blockSeen[alias]; ok {
				return fmt.Errorf("FQDN %s used more than once in domain %s", alias, v.FQDN)
			}
			blockSeen[alias] = true
		}
		for _, redirect := range v.Redirects {
			if _, ok := blockSeen[redirect]; ok {
				return fmt.Errorf("FQDN %s used more than once in domain %s", redirect, v.FQDN)
			}
			blockSeen[redirect] = true
		}
		mainSeen[v.FQDN] = true
	}
	return nil
}

// - Auth credentials setting must follow dedicated pattern.
// - Passwords mustn't differ for same users.
// - The credentials are stored in one single file per project.
func (cfg Config) validateDomainsAuth() error {
	creds := make(map[string]string)

	for _, v := range cfg.Domain {
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

	}
	return nil
}

func (cfg Config) validateDomainsSSL() error {
	for _, v := range cfg.Domain {
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
	return nil
}

// Database names must be unique and users should for security reasons not
// have an empty password (not even for dev contexts).
func (cfg Config) validateDatabases() error {
	seen := map[string]bool{}

	for _, db := range cfg.Database {
		if db.Name == "" {
			return fmt.Errorf("found empty database name")
		}
		if _, ok := seen[db.Name]; ok {
			return fmt.Errorf("found duplicate database name: %s", db.Name)
		}
		if cfg.Context != ContextDevelopment && db.Password == "" {
			return fmt.Errorf("user %s has empty password for database: %s", db.User, db.Name)
		}
		if db.User == "root" {
			return fmt.Errorf("user %s is a MySQL restricted user", db.User)
		}
		seen[db.Name] = true
	}
	return nil
}

func (cfg Config) validateVolumes() error {
	for _, volume := range cfg.Volume {
		if filepath.IsAbs(volume.Path) {
			return fmt.Errorf("volume path is not relative: %s", volume.Path)
		}
	}
	return nil
}
