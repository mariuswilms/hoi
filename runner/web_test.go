// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"os"
	"testing"

	"github.com/atelierdisko/hoi/project"
)

/*
func TestMain(m *testing.M) {
	os.RemoveAll("/tmp/test")

	mRun := m.Run()

	os.RemoveAll("/tmp/test")
	os.Exit(mRun)
}
*/

func TestAPR1ImplementationToKnown(t *testing.T) {
	r := computeAPR1("musik", "buZHPOTP")
	e := "$apr1$buZHPOTP$36ES04x2pWJCZFz18irxw."

	if r != e {
		t.Errorf("result: %s | expected: %s", r, e)
	}
}

/*
func TestDeactivate(t *testing.T) {
	simulateSystem()

	pCfg, _ := project.New()
	pCfg.Path = "/tmp/test/var/www/foo"

	sCfg, _ := server.New()
	sCfg.NGINX.RunPath = "/tmp/test/etc/nginx/sites-enabled"

	err := deactivateWeb(pCfg, sCfg)
	if err != nil {
		t.Fail()
	}
}
*/

func simulateSystem() {
	root := "/tmp/test"
	os.RemoveAll(root)
	os.Mkdir(root, 0755)

	dirs := []string{
		"/etc/hoi",
		"/etc/nginx/sites-enabled",
		"/etc/systemd/system",
		"/var/www",
	}
	for _, d := range dirs {
		os.MkdirAll(root+d, 0755)
	}
}

// Simulate mutating happening in preparation for template rendering.
func TestDoesNotModifyMasterStruct(t *testing.T) {
	hoifile := `
name = "foo"
domain example.org {
  SSL = {
    certificate = "config/ssl/example.org.crt"
    certificateKey = "config/ssl/example.org.key"
  }
}
`
	cfg, err := project.NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	mutate := func(cfg project.Config) {
		cfg.Name = "MUTATED!"

		ds := map[string]project.DomainDirective{}
		for k, _ := range cfg.Domain {
			e := cfg.Domain[k]
			e.SSL.Certificate = "MUTATED!"

			ds[k] = e
		}
		cfg.Domain = ds
	}
	mutate(*cfg)
	if cfg.Domain["example.org"].SSL.Certificate == "MUTATED!" {
		t.Error("detected mutated domain config")
		t.Logf("%#v", cfg)
	}
	if cfg.Name == "MUTATED!" {
		t.Error("detected mutated name config")
		t.Logf("%#v", cfg)
	}
}
