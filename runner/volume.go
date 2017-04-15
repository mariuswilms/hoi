// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/atelierdisko/hoi/builder"
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
	"github.com/coreos/go-systemd/dbus"
)

func NewVolumeRunner(s *server.Config, p *project.Config, conn *dbus.Conn) *VolumeRunner {
	return &VolumeRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KindVolume, p, s),
		sys:   system.NewSystemd(system.SystemdKindVolume, p, s, conn),
	}
}

type VolumeRunner struct {
	s     *server.Config
	p     *project.Config
	sys   *system.Systemd
	build *builder.Builder
}

func (r VolumeRunner) Disable() error {
	units, err := r.sys.ListInstalledMounts()
	if err != nil {
		return err
	}
	for _, u := range units {
		if err := r.sys.StopAndDisable(u); err != nil {
			return err
		}
		if err := r.sys.Uninstall(u); err != nil {
			return err
		}
	}
	return r.build.Clean()
}

func (r VolumeRunner) Enable() error {
	if len(r.p.Volume) == 0 {
		return nil // nothing to do
	}
	t, err := r.build.LoadTemplate("default.mount")
	if err != nil {
		return err
	}

	for _, v := range r.p.Volume {
		if err := r.setupDirs(v); err != nil {
			return err
		}

		tmplData := struct {
			P *project.Config
			S *server.Config
			V project.VolumeDirective
		}{
			P: r.p,
			S: r.s,
			V: v,
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s.mount", r.sys.EscapeUnitName(v.Path)),
			t,
			tmplData,
		)
		if err != nil {
			return err
		}
	}

	files, err := r.build.ListAvailable()
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := r.sys.Install(f); err != nil {
			return err
		}
		if err := r.sys.EnableAndStart(filepath.Base(f)); err != nil {
			return err
		}
	}
	return nil
}

func (r VolumeRunner) Commit() error {
	return nil
}

func (r VolumeRunner) setupDirs(v project.VolumeDirective) error {
	// Use our own (poor-man's) Chown here, so we do not need to
	// lookup the uid/gid, which would require cgo, which isn't
	// available during cross compilation.
	chown := func(path string, user string, group string) error {
		if err := exec.Command("chown", user+":"+group, path).Run(); err != nil {
			return fmt.Errorf("failed to chown %s to user %s and group %s: %s", path, user, group, err)
		}
		return nil
	}

	// Sets FS ACLs, so user + group rights are inherited by sub-directories and files.
	setfacl := func(path string) error {
		if err := exec.Command("setfacl", "-d", "-m", "g::rwx", path).Run(); err != nil {
			return fmt.Errorf("failed to set ACLs on mount source %s: %s", path, err)
		}
		return nil
	}

	// 1. owned by global user and group
	// 2. and have the sticky flag set, so when new files are created owner is the same
	// 3. user and group can read AND write, others cannot do anything
	// 4. perms are persisted even for new files
	setup := func(path string, user string, group string) error {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(path, 1770); err != nil {
				return err
			}
			if err := chown(path, user, group); err != nil {
				return err
			}
			if err := setfacl(path); err != nil {
				return err
			}
			log.Printf("setup new volume path: %s", path)
		} else {
			log.Printf("reusing volume path: %s", path)
		}
		return nil
	}

	if err := setup(v.GetSource(r.p, r.s), r.s.User, r.s.Group); err != nil {
		return err
	}
	return setup(v.GetTarget(r.p), r.s.User, r.s.Group)
}
