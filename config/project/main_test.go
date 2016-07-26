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
PHPVersion = 56
`
	cfg, err := NewFromString(hoifile)

	if err != nil {
		t.Error(err)
	}
	if cfg.Name != "foo" {
		t.Error("Name is not foo")
	}
}

func TestDecodeMakesArray(t *testing.T) {
	hoifile := `
cron high-freq {
	schedule = "*/10 * * * *"
	command = "cd bin && ./li3.php jobs runFrequency high"
}
cron medium-freq {
	schedule = "0 * * * *"
	command = "cd bin && ./li3.php jobs runFrequency medium"
}
`
	cfg, err := NewFromString(hoifile)

	if err != nil {
		t.Error(err)
	}
	if len(cfg.Cron) != 2 {
		t.Error("No 2 crons parsed")
	}
}

func TestDecodeDomainGetsFqdn(t *testing.T) {
	hoifile := `
domain "example.com" {
}
`
	cfg, err := NewFromString(hoifile)

	if err != nil {
		t.Error(err)
	}
	if cfg.Domain["example.com"].FQDN != "example.com" {
		t.Error("failed to compare FQDN")
	}
}
