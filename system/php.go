// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"os"
	"sync"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/coreos/go-systemd/dbus"
)

var (
	PHPLock  sync.RWMutex
	PHPDirty bool
)

func NewPHP(p *project.Config, s *server.Config, conn *dbus.Conn) *PHP {
	return &PHP{p: p, s: s, conn: conn}
}

type PHP struct {
	p    *project.Config
	s    *server.Config
	conn *dbus.Conn
}

// Installs just the server configuration.
func (sys PHP) Install(path string) error {
	runPath, err := sys.p.App.GetRunPath(sys.p, sys.s)
	if err != nil {
		return err
	}
	target := fmt.Sprintf("%s/99-project-%s.ini", runPath, sys.p.ID)

	if err := os.Symlink(path, target); err != nil {
		return fmt.Errorf("PHP failed to install %s -> %s: %s", path, target, err)
	}
	PHPDirty = true
	return nil
}

func (sys PHP) Uninstall() error {
	runPath, err := sys.p.App.GetRunPath(sys.p, sys.s)
	if err != nil {
		return err
	}
	target := fmt.Sprintf("%s/99-project-%s.ini", runPath, sys.p.ID)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("PHP failed to uninstall %s: %s", target, err)
	}
	PHPDirty = true
	return nil
}

func (sys PHP) ReloadIfDirty() error {
	if !PHPDirty {
		return nil
	}
	service, err := sys.p.App.GetService(sys.p, sys.s)
	if err != nil {
		return err
	}
	PHPLock.Lock()
	defer PHPLock.Unlock()

	done := make(chan string)

	if _, err := sys.conn.ReloadUnit(service, "replace", done); err != nil {
		return fmt.Errorf("failed to reload PHP; left in dirty state: %s", err)
	}
	if r := <-done; r != "done" {
		return fmt.Errorf("failed to reload PHP; systemd job states: %s", r)
	}
	PHPDirty = false
	return nil
}

func (sys PHP) IsInstalled() (bool, error) {
	runPath, err := sys.p.App.GetRunPath(sys.p, sys.s)
	if err != nil {
		return false, err
	}
	file := fmt.Sprintf("%s/99-project-%s.ini", runPath, sys.p.ID)
	_, err = os.Stat(file)
	return !os.IsNotExist(err), nil
}
