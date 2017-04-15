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
	store := New(file)
	cfg, _ := project.NewFromString("name = \"test\"")

	if err := store.Write("fookey", cfg); err != nil {
		t.Error(err)
	}
	store.Close()
	os.Remove(file)
}

func TestLoad(t *testing.T) {
	file := "/tmp/store-test.db"
	store := New(file)
	cfg, _ := project.NewFromString("name = \"test\"")

	if err := store.Write("fookey", cfg); err != nil {
		t.Error(err)
	}
	store.Close()

	store = New(file)
	store.Load()
	if !store.Has("fookey") {
		t.Error("no key fookey")
	}

	store.Close()
	os.Remove(file)
}

func TestStoreCount(t *testing.T) {
	file := "/tmp/store-test.db"
	store := New(file)
	cfg, _ := project.NewFromString("name = \"test\"")

	if err := store.Write("fookey", cfg); err != nil {
		t.Error(err)
	}
	if len(store.data) != 1 {
		t.Errorf("expected a single entry")
	}
	store.Close()
}
