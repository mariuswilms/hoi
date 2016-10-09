// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

var (
	NGINXLock  sync.RWMutex
	NGINXDirty bool
)

func NewNGINX(p project.Config, s server.Config) *NGINX {
	return &NGINX{p: p, s: s}
}

type NGINX struct {
	p     project.Config
	s     server.Config
	dirty bool
}

// Installs just the server configuration.
func (sys *NGINX) Install(path string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)
	target := fmt.Sprintf("%s/%s_%s", sys.s.NGINX.RunPath, ns, filepath.Base(path))

	if err := os.Symlink(path, target); err != nil {
		return fmt.Errorf("NGINX failed to install %s -> %s: %s", path, target, err)
	}
	NGINXDirty = true
	return nil
}

func (sys *NGINX) Uninstall(server string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)
	target := fmt.Sprintf("%s/%s_%s", sys.s.NGINX.RunPath, ns, server)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("NGINX failed to uninstall %s: %s", target, err)
	}
	NGINXDirty = true
	return nil
}

func (sys *NGINX) Reload() error {
	NGINXLock.Lock()
	defer NGINXLock.Unlock()

	if err := exec.Command("systemctl", "reload", "nginx").Run(); err != nil {
		return fmt.Errorf("failed to reload NGINX; possibly left in dirty state: %s", err)
	}
	NGINXDirty = false
	return nil
}

func (sys *NGINX) ReloadIfDirty() error {
	if !NGINXDirty {
		return nil
	}
	NGINXLock.Lock()
	defer NGINXLock.Unlock()

	if err := exec.Command("systemctl", "reload", "nginx").Run(); err != nil {
		return fmt.Errorf("failed to reload NGINX; left in dirty state: %s", err)
	}
	NGINXDirty = false
	return nil
}

func (sys NGINX) ListInstalled() ([]string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	files, err := filepath.Glob(fmt.Sprintf("%s/%s_*", sys.s.NGINX.RunPath, ns))
	if err != nil {
		return files, fmt.Errorf("failed to list projects installed in NGINX: %s", err)
	}
	servers := make([]string, 0)
	for _, f := range files {
		servers = append(servers, strings.TrimPrefix(filepath.Base(f), ns+"_"))
	}
	return servers, err
}
