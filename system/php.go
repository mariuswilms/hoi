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

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
)

type PHP struct {
	p pConfig.Config
	s sConfig.Config
}

func NewPHP(p pConfig.Config, s sConfig.Config) *PHP {
	return &PHP{p: p, s: s}
}

// Installs just the server configuration.
func (sys PHP) Install(path string) error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())

	log.Printf("PHP install: %s -> %s", path, target)
	return os.Symlink(path, target)
}

func (sys PHP) Uninstall() error {
	target := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())

	log.Printf("PHP uninstall: %s", target)
	return os.Remove(target)
}

func (sys PHP) Reload() error {
	log.Printf("PHP reload")
	if os.Getenv("HOI_NOOP") == "yes" {
		return nil
	}
	return exec.Command("systemctl", "reload", "php5-fpm").Run()
}

func (sys PHP) IsInstalled() bool {
	file := fmt.Sprintf("%s/99-project-%s.ini", sys.s.PHP.RunPath, sys.p.ID())
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
