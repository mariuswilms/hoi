// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"log"
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

func NewSSL(p project.Config, s server.Config) *SSL {
	return &SSL{p: p, s: s}
}

// The SSL system manages certificates and keys in a central directory.
//
// Certs and keys are managed in two separate sub-directories. Each
// file is named after the domain they belong to. Certs are suffixed
// with .crt, keys with .key.
type SSL struct {
	p project.Config
	s server.Config
}

func (sys *SSL) Install(domain string, ssl project.SSLDirective) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	targetKey := fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, domain)
	log.Printf("SSL is installing: %s -> %s", ssl.CertificateKey, targetKey)

	switch ssl.CertificateKey {
	case project.CertSelfSigned:
		cmd := []string{"genrsa", "-out", targetKey, "2048"}
		if err := exec.Command("openssl", cmd...).Run(); err != nil {
			return err
		}
	default:
		path := filepath.Join(sys.p.Path, ssl.CertificateKey)
		if err := util.CopyFile(path, targetKey); err != nil {
			return err
		}
	}
	SSLDirty = true

	targetCert := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, domain)
	log.Printf("SSL is installing: %s -> %s", ssl.Certificate, targetCert)

	switch ssl.Certificate {
	case project.CertSelfSigned:
		cmd := []string{
			"req", "-new",
			"-x509",
			"-sha256",
			"-nodes",
			"-days", "365",
			"-key", targetKey,
			"-out", targetCert,
			"-subj", "/C=DE/ST=Hamburg/L=Hamburg/O=None/OU=None/CN=" + domain,
		}
		if err := exec.Command("openssl", cmd...).Run(); err != nil {
			return nil // TODO even when cmd succeeds, it exits with != 0.
			// return fmt.Errorf("failed executing openssl command with %+v, got: %s", cmd, err)
		}
	default:
		path := filepath.Join(sys.p.Path, ssl.Certificate)
		if err := util.CopyFile(path, targetCert); err != nil {
			return err
		}
	}

	return nil
}

func (sys *SSL) Uninstall(domain string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, domain)
	log.Printf("SSL is uninstalling: %s", target)
	if err := os.Remove(target); err != nil {
		return err
	}
	SSLDirty = true

	target = fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, domain)
	log.Printf("SSL is uninstalling: %s", target)
	if err := os.Remove(target); err != nil {
		return err
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
		return domains, err
	}
	for _, f := range files {
		domains = append(domains, strings.TrimSuffix(
			strings.TrimPrefix(filepath.Base(f), ns+"_"),
			".key",
		))
	}
	return domains, err
}

func (sys SSL) GetCertificate(domain string) (string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/certs/%s_%s.crt", sys.s.SSL.RunPath, ns, domain)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, err
	}
	return target, nil
}

func (sys SSL) GetCertificateKey(domain string) (string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	target := fmt.Sprintf("%s/private/%s_%s.key", sys.s.SSL.RunPath, ns, domain)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, err
	}
	return target, nil
}
