// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"os"
	"testing"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
)

func TestMain(m *testing.M) {
	os.RemoveAll("/tmp/test")

	mRun := m.Run()

	os.RemoveAll("/tmp/test")
	os.Exit(mRun)
}

func TestAPR1ImplementationToKnown(t *testing.T) {
	r := computeAPR1("musik", "buZHPOTP")
	e := "$apr1$buZHPOTP$36ES04x2pWJCZFz18irxw."

	if r != e {
		t.Errorf("result: %s | expected: %s", r, e)
	}
}

func TestDeactivate(t *testing.T) {
	simulateSystem()

	pCfg, _ := pConfig.New()
	pCfg.Path = "/tmp/test/var/www/foo"

	sCfg, _ := sConfig.New()
	sCfg.NGINX.RunPath = "/tmp/test/etc/nginx/sites-enabled"

	err := deactivateWeb(pCfg, sCfg)
	if err != nil {
		t.Fail()
	}
}

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
