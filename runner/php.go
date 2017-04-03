// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"github.com/atelierdisko/hoi/builder"
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
	systemd "github.com/coreos/go-systemd/dbus"
)

func NewPHPRunner(s server.Config, p project.Config, conn *systemd.Conn) *PHPRunner {
	return &PHPRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KindPHP, p, s),
		sys:   system.NewPHP(p, s, conn),
	}
}

// The PHP runner allows to configure PHP on a per project basis.
//
// This is achieved by putting a PHP configuration file into a place where PHP
// generally looks for autoload-able configuration files while using the PATH[0]
// feature. Other approaches have proven to be buggy[1].
//
// [0] http://php.net/manual/pl/ini.sections.php
// [1] https://bugs.php.net/bug.php?id=63965
type PHPRunner struct {
	s     server.Config
	p     project.Config
	sys   *system.PHP
	build *builder.Builder
}

func (r PHPRunner) Build() error {
	if r.p.Kind != project.KindPHP {
		return nil // nothing to do
	}
	tS, err := r.build.LoadTemplate("php.ini")
	if err != nil {
		return err
	}
	tmplData := struct {
		P project.Config
		S server.Config
	}{
		P: r.p,
		S: r.s,
	}
	return r.build.WriteTemplate("php.ini", tS, tmplData)
}

func (r PHPRunner) Clean() error {
	return r.build.Clean()
}

func (r PHPRunner) Enable() error {
	if r.p.Kind != project.KindPHP {
		return nil // nothing to do
	}
	files, err := r.build.ListAvailable()
	if err != nil {
		return err
	}
	for _, v := range files {
		if err := r.sys.Install(v); err != nil {
			return err
		}
	}
	return nil
}

func (r PHPRunner) Disable() error {
	if !r.sys.IsInstalled() {
		return nil // nothing to disable
	}
	return r.sys.Uninstall()
}

func (r PHPRunner) Commit() error {
	return r.sys.ReloadIfDirty()
}
