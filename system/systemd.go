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
	"github.com/coreos/go-systemd/dbus"
	"github.com/coreos/go-systemd/unit"
)

// The hoi-internal kind of units we manage.
const (
	SystemdKindAppService = "app_service"
	SystemdKindCron       = "cron"
	SystemdKindWorker     = "worker"
	SystemdKindVolume     = "volume"
)

var (
	// SystemdDirty indicates wheter the systemd daemon needs to be
	// reloaded. Flag will be reset once daemon reloaded.
	SystemdDirty bool
)

func NewSystemd(kind string, p *project.Config, s *server.Config, conn *dbus.Conn) *Systemd {
	return &Systemd{kind: kind, p: p, s: s, conn: conn}
}

type Systemd struct {
	kind string
	p    *project.Config
	s    *server.Config
	conn *dbus.Conn
}

// Escapes unit names. Detects paths automatically and escapes them
// using special path escape.
//
// > Mount units must be named after the mount point directories they
// > control. Example: the mount point /home/lennart must be configured
// > in a unit file home-lennart.mount. For details about the escaping
// > logic used to convert a file system path to a unit name, see
// > systemd.unit(5).
func (sys Systemd) EscapeUnitName(name string) string {
	if sys.kind == SystemdKindVolume { // KindVolume uses mount units
		return unit.UnitNamePathEscape(name)
	}
	return unit.UnitNameEscape(name)
}

// Prefix for units namespaces units to each project.
//
// Mount unit names cannot be prefixed with a crafted project
// namespace as they must reflect the actual path. They are however
// naturally namespaced by the absolute project path.
func (sys Systemd) getPrefix() string {
	if sys.kind == SystemdKindVolume {
		return unit.UnitNamePathEscape(sys.p.Path) + "-"
	}
	return fmt.Sprintf("project_%s_%s_", sys.p.ID, sys.kind)
}

// Copies a unit file into the systemd configuration directory. Takes an absolute
// path to the source unit file. Using copies instead of symlinks is more robust:
// not all locations are valid symlink targets (i.e. files under /etc).
func (sys Systemd) Install(path string) error {
	target := fmt.Sprintf("%s/%s%s", sys.s.Systemd.RunPath, sys.getPrefix(), filepath.Base(path))

	if err := util.CopyFile(path, target); err != nil {
		return fmt.Errorf("failed to copy systemd unit %s -> %s: %s", path, target, err)
	}
	SystemdDirty = true
	return nil
}

// Removes a copy/link of a unit file inside the systemd configuration directory. Takes
// the unprefixed unit name including type suffix (i.e. "example.service", "tmp-cache.mount").
func (sys Systemd) Uninstall(unit string) error {
	target := fmt.Sprintf("%s/%s%s", sys.s.Systemd.RunPath, sys.getPrefix(), unit)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to remove systemd unit %s: %s", target, err)
	}
	SystemdDirty = true
	return nil
}

func (sys Systemd) ReloadIfDirty() error {
	if !SystemdDirty {
		return nil
	}
	if err := sys.conn.Reload(); err != nil {
		return fmt.Errorf("failed to reload systemd; left in dirty state: %s", err)
	}
	SystemdDirty = false
	return nil
}

func (sys Systemd) ListInstalledServices() ([]string, error) {
	return sys.listInstalledUnits("service")
}

func (sys Systemd) ListInstalledTimers() ([]string, error) {
	return sys.listInstalledUnits("timer")
}

func (sys Systemd) ListInstalledMounts() ([]string, error) {
	return sys.listInstalledUnits("mount")
}

// Enables a unit for automatic startup at system boot and immediately starts the unit. Takes
// an unprefixed unit name including the type suffix (i.e. "example.service", "tmp-cache.mount").
func (sys Systemd) EnableAndStart(unit string) error {
	target := fmt.Sprintf("%s%s", sys.getPrefix(), unit)

	_, _, err := sys.conn.EnableUnitFiles(
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

// Disables a unit for automatic startup at system boot and immediately stops the unit. Takes
// an unprefixed unit name including the type suffix (i.e. "example.service", "tmp-cache.mount").
func (sys Systemd) StopAndDisable(unit string) error {
	target := fmt.Sprintf("%s%s", sys.getPrefix(), unit)

	_, err := sys.conn.DisableUnitFiles(
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

// Immediately stops the unit. Takes an unprefixed unit name including the
// type suffix (i.e. "example.service", "tmp-cache.mount").
func (sys Systemd) Stop(unit string) error {
	target := fmt.Sprintf("%s%s", sys.getPrefix(), unit)

	_, err := sys.conn.StopUnit(target, "replace", nil)
	if err != nil {
		return fmt.Errorf("failed to stop systemd unit %s: %s", target, err)
	}
	return nil
}

// Lists installed units. Strips prefix, leaving just the plain unit
// name including its suffix (i.e. "example.service").
func (sys Systemd) listInstalledUnits(suffix string) ([]string, error) {
	prefix := sys.getPrefix()
	var uns []string // unit names including suffix, excluding ns

	if sys.s.Systemd.UseLegacy {
		us, err := sys.conn.ListUnits()
		if err != nil {
			return uns, err
		}
		for _, u := range us {
			matched, err := regexp.MatchString(
				fmt.Sprintf("^%s.*\\.*%s$", prefix, suffix),
				u.Name,
			)
			if err != nil {
				return uns, nil
			}
			if matched {
				uns = append(uns, strings.TrimPrefix(u.Name, prefix))
			}
		}
		return uns, nil
	}
	// List...ByPatterns available since v230
	us, err := sys.conn.ListUnitsByPatterns(
		[]string{"active", "inactive", "failed"}, // all possible states
		[]string{fmt.Sprintf("%s*.%s", prefix, suffix)},
	)
	if err != nil {
		return uns, err
	}
	for _, u := range us {
		uns = append(uns, strings.TrimPrefix(u.Name, prefix))
	}
	return uns, nil
}
