// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import "testing"

func TestDomainWithoutTLD(t *testing.T) {
	tld := TLD("localhost")
	if tld != "" {
		t.Error("failed to handle domain without TLD")
	}
}

func TestSecondLevelDomain(t *testing.T) {
	tld := TLD("example.org")
	if tld != "org" {
		t.Error("failed to handle second-level domain")
	}
}

func TestThirdLevelDomain(t *testing.T) {
	tld := TLD("www.example.net")
	if tld != "net" {
		t.Error("failed to handle third-level domain")
	}
}
