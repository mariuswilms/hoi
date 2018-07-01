// Copyright 2017 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"path/filepath"

	"github.com/atelierdisko/hoi/builder"
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
	systemd "github.com/coreos/go-systemd/dbus"
)

func NewAppServiceRunner(s *server.Config, p *project.Config, conn *systemd.Conn) *AppServiceRunner {
	return &AppServiceRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KindAppService, p, s),
		sys:   system.NewSystemd(system.SystemdKindAppService, p, s, conn),
	}
}

type AppServiceRunner struct {
	s     *server.Config
	p     *project.Config
	sys   *system.Systemd
	build *builder.Builder
}

func (r AppServiceRunner) Disable() error {
	services, err := r.sys.ListInstalledServices()
	if err != nil {
		return err
	}
	for _, uS := range services {
		if err := r.sys.StopAndDisable(uS); err != nil {
			return err
		}
		if err := r.sys.Uninstall(uS); err != nil {
			return err
		}
	}
	return r.build.Clean()
}

func (r AppServiceRunner) Enable() error {
	if !r.p.App.HasCommand() {
		return nil // nothing to do
	}

	tS, err := r.build.LoadTemplate("default.service")
	tmplData := struct {
		P *project.Config
		S *server.Config
	}{
		P: r.p,
		S: r.s,
	}
	err = r.build.WriteTemplate(
		"default.service",
		tS,
		tmplData,
	)
	if err != nil {
		return err
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

func (r AppServiceRunner) Commit() error {
	return r.sys.ReloadIfDirty()
}
