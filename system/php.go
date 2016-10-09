// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
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

	if err := os.Symlink(path, target); err != nil {
		return fmt.Errorf("PHP failed to install %s -> %s: %s", path, target, err)
	}
	PHPDirty = true
	return nil
}

func (sys PHP) Uninstall() error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("PHPfailed to uninstall %s: %s", target, err)
	}
	PHPDirty = true
	return nil
}

func (sys PHP) ReloadIfDirty() error {
	if !PHPDirty {
		return nil
	}
	PHPLock.Lock()
	defer PHPLock.Unlock()

	if err := exec.Command("systemctl", "reload", "php5-fpm").Run(); err != nil {
		return fmt.Errorf("failed to reload PHP; left in dirty state: %s", err)
	}
	PHPDirty = false
	return nil
}

func (sys PHP) IsInstalled() bool {
	file := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID)
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
