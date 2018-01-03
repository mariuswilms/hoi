// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"archive/tar"
	"fmt"
	"log"
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
		fs:    system.NewFilesystem(p, s),
	}
}

type VolumeRunner struct {
	s     *server.Config
	p     *project.Config
	build *builder.Builder
	sys   *system.Systemd
	fs    *system.Filesystem
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
		if err := r.fs.SetupVolume(v); err != nil {
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

// Creates dumps of all persistent volumes.
func (r VolumeRunner) Dump(tw *tar.Writer) error {
	for _, v := range r.p.Volume {
		if v.IsTemporary {
			continue
		}
		log.Printf("dumping volume %s", v.Path)

		if err := r.fs.DumpVolume(v, tw); err != nil {
			return err
		}
	}
	return nil
}
