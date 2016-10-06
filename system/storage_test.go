// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestParseMounts(t *testing.T) {
	out, err := exec.Command("mount").Output()
	if err != nil {
		t.Fatal(err)
	}
	mounts, err := parseMountOutput(string(out))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", mounts)
}

func TestPersist(t *testing.T) {
	d1 := []byte("proc /proc proc defaults 0 0\n/dev/md/0 none swap sw 0 0\n/dev/md/1 /boot ext3 defaults 0 0")

	ioutil.WriteFile("/tmp/fstab.test", d1, 0644)
	defer os.Remove("/tmp/fstab.test")

	if err := persistBindMount("/tmp/fstab.test", "/from/here", "/to/there"); err != nil {
		t.Error(err)
	}
	if err := unpersistBindMount("/tmp/fstab.test", "/to/there"); err != nil {
		t.Error(err)
	}

	d2, _ := ioutil.ReadFile("/tmp/fstab.test")

	if string(d1) != string(d2) {
		t.Logf("d1: %s", d1)
		t.Logf("d2: %s", d2)
		t.Fail()
	}
}

func TestPersistFstabHasNL(t *testing.T) {
	d1 := []byte("proc /proc proc defaults 0 0\n/dev/md/0 none swap sw 0 0\n/dev/md/1 /boot ext3 defaults 0 0\n")

	ioutil.WriteFile("/tmp/fstab.test", d1, 0644)
	defer os.Remove("/tmp/fstab.test")

	if err := persistBindMount("/tmp/fstab.test", "/from/here", "/to/there"); err != nil {
		t.Error(err)
	}
	if err := unpersistBindMount("/tmp/fstab.test", "/to/there"); err != nil {
		t.Error(err)
	}

	d2, _ := ioutil.ReadFile("/tmp/fstab.test")

	if string(d1) != string(d2) {
		t.Logf("d1: %s", d1)
		t.Logf("d2: %s", d2)
		t.Fail()
	}
}
