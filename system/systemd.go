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
	"github.com/atelierdisko/hoi/util"
)

// The hoi-internal kind of units we manage.
const (
	SystemdKindCron   = "cron"
	SystemdKindWorker = "worker"
)

func NewSystemd(kind string, p project.Config, s server.Config) *Systemd {
	return &Systemd{kind: kind, p: p, s: s}
}

type Systemd struct {
	kind string
	p    project.Config
	s    server.Config
}

// When installing unit files, they are prefixed as to namespace them by project.
func (sys Systemd) Install(path string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, filepath.Base(path))

	log.Printf("systemd install: %s -> %s", path, target)
	if sys.s.Systemd.UseLegacy {
		return util.CopyFile(path, target)
	}
	return os.Symlink(path, target)
}

func (sys Systemd) Uninstall(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, unit)

	log.Printf("systemd uninstall: %s", target)
	return os.Remove(target)
}

// Lists installed service units. Strips project namespace.
func (sys Systemd) ListInstalledServices() ([]string, error) {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	units := make([]string, 0)

	out, err := exec.Command("systemctl", "list-units", fmt.Sprintf("'%s_*.service'", ns), "--no-legend", "--no-pager").Output()
	if err != nil {
		return units, err
	}

	if len(out) != 0 {
		// line format:
		// worker@1.service loaded active running Worker aaa for project ad@dev
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			units = append(units, strings.TrimPrefix(fields[0], ns+"_"))
		}
	}
	log.Printf("systemd found %d installed service unit/s:\n%v", len(units), units)
	return units, err
}

// Lists installed timer  units. Strips project namespace.
func (sys Systemd) ListInstalledTimers() ([]string, error) {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	units := make([]string, 0)

	out, err := exec.Command("systemctl", "list-units", fmt.Sprintf("'%s_*.timer'", ns), "--no-legend", "--no-pager").Output()
	if err != nil {
		return units, err
	}
	if len(out) != 0 {
		// line format:
		// worker@1.service loaded active running Worker aaa for project ad@dev
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			units = append(units, strings.TrimPrefix(fields[0], ns+"_"))
		}
	}
	log.Printf("systemd found %d installed timer unit/s:\n%v", len(units), units)
	return units, err
}

func (sys Systemd) EnableAndStart(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	log.Printf("systemd enable+start: %s", ns+"_"+unit)

	if sys.s.Systemd.UseLegacy {
		// --now cannot be used with at least 215
		if err := exec.Command("systemctl", "enable", ns+"_"+unit).Run(); err != nil {
			return err
		}
		return exec.Command("systemctl", "start", ns+"_"+unit).Run()
	}
	return exec.Command("systemctl", "enable", "--now", ns+"_"+unit).Run()
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) StopAndDisable(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	log.Printf("systemd stop+disable: %s", ns+"_"+unit)

	if sys.s.Systemd.UseLegacy {
		// --now cannot be used with at least 215
		if err := exec.Command("systemctl", "stop", ns+"_"+unit).Run(); err != nil {
			return err
		}
		return exec.Command("systemctl", "disable", ns+"_"+unit).Run()
	}
	return exec.Command("systemctl", "disable", "--now", ns+"_"+unit).Run()
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) Stop(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	log.Printf("systemd stop: %s", ns+"_"+unit)

	return exec.Command("systemctl", "stop", ns+"_"+unit).Run()
}
