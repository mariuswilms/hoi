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
	"sync"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

var (
	PHPLock  sync.RWMutex
	PHPDirty bool
)

func NewPHP(p project.Config, s server.Config) *PHP {
	return &PHP{p: p, s: s}
}

type PHP struct {
	p project.Config
	s server.Config
}

// Installs just the server configuration.
func (sys PHP) Install(path string) error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID)

	log.Printf("PHP is installing: %s -> %s", path, target)

	PHPDirty = true
	return os.Symlink(path, target)
}

func (sys PHP) Uninstall() error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID)

	log.Printf("PHP is uninstalling: %s", target)

	PHPDirty = true
	return os.Remove(target)
}

func (sys PHP) ReloadIfDirty() error {
	if !PHPDirty {
		return nil
	}
	log.Printf("PHP is reloading")

	PHPLock.Lock()
	defer PHPLock.Unlock()

	if err := exec.Command("systemctl", "reload", "php5-fpm").Run(); err != nil {
		return fmt.Errorf("PHP left in dirty state: %s", err)
	}
	PHPDirty = false
	return nil
}

func (sys PHP) IsInstalled() bool {
	file := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID)
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
