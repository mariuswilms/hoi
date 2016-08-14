// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
)

func NewSSLRunner(s server.Config, p project.Config) *SSLRunner {
	return &SSLRunner{
		s:   s,
		p:   p,
		sys: system.NewSSL(p, s),
	}
}

// The SSL runner manages certificates and keys in a central directory
// (most often this is /etc/ssl).
type SSLRunner struct {
	s   server.Config
	p   project.Config
	sys *system.SSL
	// does not use a builder, works directly from projects
}

func (r SSLRunner) Disable() error {
	domains, err := r.sys.ListInstalled()
	if err != nil {
		return err
	}
	for _, domain := range domains {
		if err := r.sys.Uninstall(domain); err != nil {
			return err
		}
	}
	return nil
}

// Installs certs/keys directly from project directories.
func (r SSLRunner) Enable() error {
	certs := r.p.GetCerts()

	if len(certs) == 0 {
		return nil // nothing to do
	}
	for domain, ssl := range certs {
		if err := r.sys.Install(domain, ssl); err != nil {
			return err
		}
	}
	return nil
}

func (r SSLRunner) Commit() error {
	return nil
}

func (r SSLRunner) Clean() error {
	return nil
}

func (r SSLRunner) Build() error {
	return nil
}
