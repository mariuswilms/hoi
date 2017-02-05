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
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, filepath.Base(path))

	if sys.s.Systemd.UseLegacy {
		if err := util.CopyFile(path, target); err != nil {
			return fmt.Errorf("failed to copy systemd unit %s -> %s: %s", path, target, err)
		}
	} else {
		if err := os.Symlink(path, target); err != nil {
			return fmt.Errorf("failed to symlink systemd unit %s -> %s: %s", path, target, err)
		}
	}
	return nil
}

func (sys Systemd) Uninstall(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := fmt.Sprintf("%s/%s_%s", sys.s.Systemd.RunPath, ns, unit)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to remove systemd unit %s: %s", target, err)
	}
	return nil
}

func (sys Systemd) ListInstalledServices() ([]string, error) {
	return listInstalledUnits(
		fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind),
		"service",
	)
}

func (sys Systemd) ListInstalledTimers() ([]string, error) {
	return listInstalledUnits(
		fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind),
		"timer",
	)
}

func (sys Systemd) EnableAndStart(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	if sys.s.Systemd.UseLegacy {
		// --now cannot be used with at least 215
		if err := exec.Command("systemctl", "enable", target).Run(); err != nil {
			return fmt.Errorf("failed to enable systemd unit %s: %s", target, err)
		}
		if err := exec.Command("systemctl", "start", target).Run(); err != nil {
			return fmt.Errorf("failed to start systemd unit %s: %s", target, err)
		}
	} else {
		if err := exec.Command("systemctl", "enable", "--now", target).Run(); err != nil {
			return fmt.Errorf("failed to enable+start systemd unit %s: %s", target, err)
		}
	}
	return nil
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) StopAndDisable(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	if sys.s.Systemd.UseLegacy {
		// --now cannot be used with at least 215
		if err := exec.Command("systemctl", "stop", target).Run(); err != nil {
			return fmt.Errorf("failed to disable systemd unit %s: %s", target, err)
		}
		if err := exec.Command("systemctl", "disable", target).Run(); err != nil {
			return fmt.Errorf("failed to disable systemd unit %s: %s", target, err)
		}
	} else {
		if err := exec.Command("systemctl", "disable", "--now", target).Run(); err != nil {
			return fmt.Errorf("failed to stop+disable systemd unit %s: %s", target, err)
		}
	}
	return nil
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) Stop(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	if err := exec.Command("systemctl", "stop", target).Run(); err != nil {
		return fmt.Errorf("failed to stop systemd unit %s: %s", target, err)
	}
	return nil
}

// Lists installed units. Strips project namespace, leaving just the
// plain unit name including its suffix.
func listInstalledUnits(ns string, suffix string) ([]string, error) {
	args := []string{
		"list-units",
		"--no-legend",
		"--no-pager",
		fmt.Sprintf("%s_*.%s", ns, suffix),
	}
	out, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		return nil, err
	}
	return parseListUnits(string(out), ns, suffix)
}

// line format:
// worker@1.service loaded active running Worker aaa for project ad@dev
func parseListUnits(out string, ns string, suffix string) ([]string, error) {
	units := make([]string, 0)

	if len(out) == 0 {
		return units, nil
	}
	if !strings.Contains(out, "\n") {
		return units, fmt.Errorf("failed to parse unit names, invalid input: %s", out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)

		if len(fields) < 1 {
			return units, fmt.Errorf("failed to parse unit name from list line: %s", line)
		}
		units = append(units, strings.TrimPrefix(fields[0], ns+"_"))
	}
	return units, nil
}
