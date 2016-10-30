// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import "testing"

func TestDecodeRoot(t *testing.T) {
	hoifile := `
name = "foo"
context = "prod"
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "foo" {
		t.Error("Name is not foo")
	}
}

func TestDecodeMakesArray(t *testing.T) {
	hoifile := `
cron high-freq {
}
cron medium-freq {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Cron) != 2 {
		t.Error("No 2 crons parsed")
	}
}

func TestDecodeSetsName(t *testing.T) {
	hoifile := `
cron high-freq {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cron["high-freq"].Name != "high-freq" {
		t.Error("failed to parse name")
	}
}

func TestDecodeDomainGetsFQDN(t *testing.T) {
	hoifile := `
domain "example.com" {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Domain["example.com"].FQDN != "example.com" {
		t.Error("failed to compare FQDN")
	}
}

func TestAccessCertPath(t *testing.T) {
	hoifile := `
domain "example.com" {
	ssl = {
		certificate = "foo.crt"
		certificateKey = "foo.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Domain["example.com"].FQDN != "example.com" {
		t.Error("failed to compare FQDN")
	}
}

func TestSimpleDomain(t *testing.T) {
	hoifile := `
	domain example.org {}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Domain) != 1 || cfg.Domain["example.org"].FQDN != "example.org" {
		t.Error("failed to use simple domain syntax")
	}
}

func TestDomainRedirectsGetParsed(t *testing.T) {
	hoifile := `
	domain example.org {
		redirects = [
			"uor.org",
			"uor.com"
		]
	}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Domain["example.org"].Redirects) != 2 {
		t.Error("did not parse 2 redirects")
	}
}

func TestNamelessDatabase(t *testing.T) {
	hoifile := `
database foo {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Database) != 1 {
		t.Error("No 1 db parsed")
	}
}
