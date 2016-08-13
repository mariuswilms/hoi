// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

type PHP struct {
	p     project.Config
	s     server.Config
	dirty bool
}

func NewPHP(p project.Config, s server.Config) *PHP {
	return &PHP{p: p, s: s}
}

// Installs just the server configuration.
func (sys *PHP) Install(path string) error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())

	log.Printf("PHP install: %s -> %s", path, target)

	sys.dirty = true
	return os.Symlink(path, target)
}

func (sys *PHP) Uninstall() error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())

	log.Printf("PHP uninstall: %s", target)

	sys.dirty = true
	return os.Remove(target)
}

func (sys PHP) Reload() error {
	log.Printf("PHP reload")
	return exec.Command("systemctl", "reload", "php5-fpm").Run()
}

func (sys *PHP) ReloadIfDirty() error {
	if !sys.dirty {
		return nil
	}
	log.Printf("PHP reload")

	if err := exec.Command("systemctl", "reload", "php5-fpm").Run(); err != nil {
		log.Printf("PHP reload: left in dirty state")
		return err
	}
	sys.dirty = false
	return nil
}

func (sys PHP) IsInstalled() bool {
	file := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
