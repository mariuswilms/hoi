// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import "testing"

func TestDecodeWorkerInstances(t *testing.T) {
	hoifile := `
worker foo {
	command = "/bin/echo foo"
	instances = 2
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Worker["foo"].GetInstances() != 2 {
		t.Error("invalid num of instances")
	}
}
