// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/util"
	systemd "github.com/coreos/go-systemd/dbus"
)

// The hoi-internal kind of units we manage.
const (
	SystemdKindCron   = "cron"
	SystemdKindWorker = "worker"
)

func NewSystemd(kind string, p project.Config, s server.Config, conn *systemd.Conn) *Systemd {
	return &Systemd{kind: kind, p: p, s: s, conn: conn}
}

type Systemd struct {
	kind string
	p    project.Config
	s    server.Config
	conn *systemd.Conn
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
	return sys.listInstalledUnits("service")
}

func (sys Systemd) ListInstalledTimers() ([]string, error) {
	return sys.listInstalledUnits("timer")
}

func (sys Systemd) EnableAndStart(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	var err error

	_, _, err = sys.conn.EnableUnitFiles(
		[]string{target},
		false, // false means persistently
		false, // unit files not cleaned up previously are an error
	)
	if err != nil {
		return fmt.Errorf("failed to enable systemd unit %s: %s", target, err)
	}

	_, err = sys.conn.StartUnit(target, "replace", nil)
	if err != nil {
		return fmt.Errorf("failed to start systemd unit %s: %s", target, err)
	}
	return nil
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) StopAndDisable(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	var err error

	_, err = sys.conn.DisableUnitFiles(
		[]string{target},
		false, // false means persistently
	)
	if err != nil {
		return fmt.Errorf("failed to disable systemd unit %s: %s", target, err)
	}

	_, err = sys.conn.StopUnit(target, "replace", nil)
	if err != nil {
		return fmt.Errorf("failed to stop systemd unit %s: %s", target, err)
	}
	return nil
}

// Disable needs unit name, doesn't work on full path.
func (sys Systemd) Stop(unit string) error {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	target := ns + "_" + unit

	_, err := sys.conn.StopUnit(target, "replace", nil)
	if err != nil {
		return fmt.Errorf("failed to stop systemd unit %s: %s", target, err)
	}
	return nil
}

// Lists installed units. Strips project namespace, leaving just the
// plain unit name including its suffix.
func (sys Systemd) listInstalledUnits(suffix string) ([]string, error) {
	ns := fmt.Sprintf("project_%s_%s", sys.p.ID, sys.kind)
	var uns []string // unit names including suffix, excluding ns

	if sys.s.Systemd.UseLegacy {
		us, err := sys.conn.ListUnits()
		if err != nil {
			return uns, err
		}
		for _, u := range us {
			matched, err := regexp.MatchString(
				fmt.Sprintf("^%s.*\\.*%s$", ns, suffix),
				u.Name,
			)
			if err != nil {
				return uns, nil
			}
			if matched {
				uns = append(uns, strings.TrimPrefix(u.Name, ns+"_"))
			}
		}
		return uns, nil
	}
	// List...ByPatterns available since v230
	us, err := sys.conn.ListUnitsByPatterns(
		[]string{"active", "inactive", "failed"}, // all possible states
		[]string{fmt.Sprintf("%s*.%s", ns, suffix)},
	)
	if err != nil {
		return uns, err
	}
	for _, u := range us {
		uns = append(uns, strings.TrimPrefix(u.Name, ns+"_"))
	}
	return uns, nil
}
