// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

const (
	// Advices to keep the www prefix. This will not deploy any
	// redirects and just leave the two domains untouched.
	WWWKeep = "keep"
	// Advices to drop the www prefix and always redirect to the naked
	// domain.
	WWWDrop = "drop"
	// Advices to add the www prefix and redirect to the prefixed
	// domain.
	WWWAdd = "add"
)

// Domains are configured using the naked domain. Handling of the www. prefix can
// be controlled via the "www" option. By default the prefix is dropped.
type DomainDirective struct {
	// The naked domain name; required.
	FQDN string
	// Configures how the www prefix is handled/normalized; optional;
	// either "keep", "drop" or "add"; defaults to "drop".
	WWW string
	// Optionally configures SSL for this domain; by default not
	// enabled. Once SSL is enabled all non SSL traffic will be
	// redirected.
	SSL SSLDirective
	// Allows to protect the domain with authentication; optional; by
	// default not enabled.
	Auth AuthDirective
	// A domain can have one or multiple optional aliases which
	// inherit any configuration from the it. If your alias needs
	// different configuration add it as an additional domain.
	Aliases []string
	// A list of domains that should redirect to this domain;
	// optional; by default empty.
	Redirects []string
}

// Access protection via auth - especially useful for staging/preview
// contexts. When both User and Password are empty, auth will be
// disabled altogether.
type AuthDirective struct {
	// If Password is given, User becomes optional and will
	// default to the naked project name.
	User string
	// Required; must be non-empty except in dev contexts.
	Password string
}

func (drv AuthDirective) IsEnabled() bool {
	return drv.User != "" && drv.Password != ""
}

const (
	// Will generate a self-signed cert on the fly.
	CertSelfSigned = "!self-signed"
)

// Certificate files should be named after the domain they belong to. Symlinks
// if wildcards certs are in use are possible, too.
type SSLDirective struct {
	// Paths to certificate and certificate key. Paths must be
	// relative to project root i.e. config/ssl/example.org.crt.
	Certificate    string
	CertificateKey string
}

func (drv SSLDirective) IsEnabled() bool {
	return drv.Certificate != "" && drv.CertificateKey != ""
}
