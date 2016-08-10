// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package system

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
)

const SYSTEMD_KIND_CRON = "cron"
const SYSTEMD_KIND_WORKER = "worker"

func NewSystemd(kind string, p pConfig.Config, s sConfig.Config) *Systemd {
	return &Systemd{kind: kind, p: p, s: s}
}

type Systemd struct {
	kind string
	p    pConfig.Config
	s    sConfig.Config
}

// When installing unit files, they are prefixed as to namespace them by project.
func (sys Systemd) Install(path string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, filepath.Base(path))

	log.Printf("systemd install: %s -> %s", path, target)
	if sys.s.Systemd.UseLegacy {
		return copyFile(path, target)
	}
	return os.Symlink(path, target)
}

func (sys Systemd) Uninstall(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, unit)

	log.Printf("systemd uninstall: %s", target)
	return os.Remove(target)
}

// TODO Use systemctl list-units with pattern to also find units
// that have been started using a service template (i.e. workers)
// Lists installed service units. Strips project namespace.
func (sys Systemd) ListInstalledServices() ([]string, error) {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	path := fmt.Sprintf("%s/%s*.service", sys.s.Systemd.RunPath, ns)

	files, err := filepath.Glob(path)
	if err != nil {
		return files, err
	}
	units := make([]string, 0)

	for _, f := range files {
		units = append(units, strings.TrimPrefix(filepath.Base(f), ns+"_"))
	}
	log.Printf("systemd found %d installed service unit/s matched by: %s\n%v", len(units), path, units)
	return units, err
}

// FIXME Use systemctl list-timers with pattern to also find units
// that have been started using a service template.
// Lists installed timer  units. Strips project namespace.
func (sys Systemd) ListInstalledTimers() ([]string, error) {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID(), sys.kind)
	path := fmt.Sprintf("%s/%s*.timer", sys.s.Systemd.RunPath, ns)

	files, err := filepath.Glob(path)
	if err != nil {
		return files, err
	}
	units := make([]string, 0)

	for _, f := range files {
		units = append(units, strings.TrimPrefix(filepath.Base(f), ns+"_"))
	}
	log.Printf("systemd found %d installed timer unit/s matched by: %s\n%v", len(units), path, units)
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

func copyFile(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, info.Mode())
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return nil
}
