// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"log"
	"os"
	"os/exec"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	targetKey := "/tmp/test.key"
	//	targetCert := "/tmp/test.crt"
	domain := "example.org"

	cmd := []string{"genrsa", "-out", targetKey, "2048"}
	if err := exec.Command("openssl", cmd...).Run(); err != nil {
	}

	cmd = []string{
		"req", "-new",
		"-x509",
		"-sha256",
		"-nodes",
		"-days", "365",
		"-key", targetKey,
		//	"-out", targetCert,
		"-subj", "/C=DE/ST=Hamburg/L=Hamburg/O=None/OU=None/CN=" + domain,
	}
	c := exec.Command("openssl", cmd...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		log.Printf("error: %s", err)
	}

}
