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
	"path/filepath"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

type NGINX struct {
	p     project.Config
	s     server.Config
	dirty bool
}

func NewNGINX(p project.Config, s server.Config) *NGINX {
	return &NGINX{p: p, s: s}
}

// Installs just the server configuration.
func (sys *NGINX) Install(path string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID())
	target := fmt.Sprintf("%s/%s_%s", sys.s.NGINX.RunPath, ns, filepath.Base(path))

	log.Printf("NGINX is installing: %s -> %s", path, target)

	sys.dirty = true
	return os.Symlink(path, target)
}

func (sys *NGINX) Uninstall(server string) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID())
	target := fmt.Sprintf("%s/%s_%s", sys.s.NGINX.RunPath, ns, server)

	log.Printf("NGINX is uninstalling: %s", target)

	sys.dirty = true
	return os.Remove(target)
}

func (sys NGINX) Reload() error {
	log.Printf("NGINX is reloading")
	return exec.Command("systemctl", "reload", "nginx").Run()
}

func (sys *NGINX) ReloadIfDirty() error {
	if !sys.dirty {
		return nil
	}
	log.Printf("NGINX is reloading")

	if err := exec.Command("systemctl", "reload", "nginx").Run(); err != nil {
		log.Printf("NGINX is left in dirty state")
		return err
	}
	sys.dirty = false
	return nil
}

func (sys NGINX) ListInstalled() ([]string, error) {
	ns := fmt.Sprintf("project_%s", sys.p.ID())

	files, err := filepath.Glob(fmt.Sprintf("%s/%s_*", sys.s.NGINX.RunPath, ns))
	if err != nil {
		return files, err
	}
	servers := make([]string, 0)
	for _, f := range files {
		servers = append(servers, strings.TrimPrefix(filepath.Base(f), ns+"_"))
	}
	return servers, err
}
