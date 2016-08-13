// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"os"
	"testing"

	"github.com/atelierdisko/hoi/project"
)

func TestStore(t *testing.T) {
	file := "/tmp/store-test.db"
	store, _ := New(file)
	cfg, _ := project.NewFromString("name = \"test\"")

	if err := store.Write("fookey", *cfg); err != nil {
		t.Error(err)
	}
	if err := store.Store(); err != nil {
		t.Error(err)
	}
	store.Close()
	os.Remove(file)
}

func TestLoad(t *testing.T) {
	file := "/tmp/store-test.db"
	store, _ := New(file)
	cfg, _ := project.NewFromString("name = \"test\"")

	if err := store.Write("fookey", *cfg); err != nil {
		t.Error(err)
	}
	if err := store.Store(); err != nil {
		t.Error(err)
	}
	store.Close()

	store, err := New(file)
	if err != nil {
		t.Error(err)
	}
	if !store.Has("fookey") {
		t.Error("no key fookey")
	}

	store.Close()
	os.Remove(file)
}
