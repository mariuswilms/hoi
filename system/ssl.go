// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/util"
)

var (
	// no need for mutex: all actions are atomic, we
	// do not reload the whole configuration
	SSLDirty bool
)

func NewSSL(p *project.Config, s *server.Config) *SSL {
	return &SSL{p: p, s: s}
}

// The SSL system manages certificates and keys in a central directory.
//
// Certs and keys are managed in two separate sub-directories. Each
// file is named after the domain they belong to. Certs are suffixed
// with .crt, keys with .key.
type SSL struct {
	p *project.Config
	s *server.Config
}

func (sys *SSL) Install(domain string, ssl project.SSLDirective) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	targetKey := fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, domain)

	switch ssl.CertificateKey {
	case project.CertKeySystem:
		sourceKey, err := sys.s.SSL.GetSystemCertificateKey(domain)
		if err != nil {
			return err
		}
		if err := util.CopyFile(sourceKey, targetKey); err != nil {
			return fmt.Errorf("failed to copy system SSL cert key %s -> %s: %s", sourceKey, targetKey, err)
		}
	case project.CertKeyGenerate:
		cmd := []string{"genrsa", "-out", targetKey, "2048"}
		if err := exec.Command("openssl", cmd...).Run(); err != nil {
			return fmt.Errorf("failed to generate SSL cert key to %s: %s", targetKey, err)
		}
	default:
		sourceKey := filepath.Join(sys.p.Path, ssl.CertificateKey)
		// TODO Ensure target file is 0600, even if source file had different perms,
		// in order to keep system directory clean.
		if err := util.CopyFile(sourceKey, targetKey); err != nil {
			return fmt.Errorf("failed to copy project SSL cert key %s -> %s: %s", sourceKey, targetKey, err)
		}
	}
	SSLDirty = true // is now dirty, ensure is set, we might exit below

	targetCert := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, domain)

	switch ssl.Certificate {
	case project.CertSystem:
		sourceCert, err := sys.s.SSL.GetSystemCertificate(domain)
		if err != nil {
			return err
		}
		if err := util.CopyFile(sourceCert, targetCert); err != nil {
			return fmt.Errorf("failed to copy system SSL cert %s -> %s: %s", sourceCert, targetCert, err)
		}
	case project.CertSelfSigned:
		cmd := []string{
			"req", "-new",
			"-x509",
			"-sha256",
			"-nodes",
			"-days", "365",
			"-key", targetKey,
			"-out", targetCert,
			"-subj",
			// "domain" can be assumed to be the naked domain. The
			// cert will be issued for the naked domains with the www.
			// subdomain in altnames.
			fmt.Sprintf(
				"/C=DE/ST=Hamburg/L=Hamburg/O=None/OU=None/CN=%s/subjectAltName=DNS.1=www.%s",
				domain, domain,
			),
		}
		if err := exec.Command("openssl", cmd...).Run(); err != nil {
			return nil // TODO even when cmd succeeds, it exits with != 0.
			// return fmt.Errorf("failed executing openssl command with %+v, got: %s", cmd, err)
		}
	default:
		sourceCert := filepath.Join(sys.p.Path, ssl.Certificate)

		if err := util.CopyFile(sourceCert, targetCert); err != nil {
			return fmt.Errorf("failed to copy project SSL cert %s -> %s: %s", sourceCert, targetCert, err)
		}
	}

	return nil
}

func (sys *SSL) Uninstall(domain string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, domain)
	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to uninstall SSL cert %s: %s", target, err)
	}
	SSLDirty = true

	target = fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, domain)
	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to uninstall SSL cert key %s: %s", target, err)
	}

	return nil
}

// Checks just the keys subdirectory, if cert is missing
// this is an error that may or may not be detected by install/uninstall.
func (sys SSL) ListInstalled() ([]string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)
	domains := make([]string, 0)

	files, err := filepath.Glob(fmt.Sprintf("%s/private/%s_*.key", sys.s.SSL.RunPath, ns))
	if err != nil {
		return domains, fmt.Errorf("failed listing installed SSL certs: %s", err)
	}
	for _, f := range files {
		domains = append(domains, strings.TrimSuffix(
			strings.TrimPrefix(filepath.Base(f), ns+"_"),
			".key",
		))
	}
	return domains, err
}

func (sys SSL) GetCertificate(fqdn string) (string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, fqdn)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, err
	}
	return target, nil
}

func (sys SSL) GetCertificateKey(fqdn string) (string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, fqdn)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, err
	}
	return target, nil
}
