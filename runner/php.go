// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package runner

import (
	"log"

	"github.com/atelierdisko/hoi/builder"
	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
	"github.com/atelierdisko/hoi/system"
)

func NewPHPRunner(s sConfig.Config, p pConfig.Config) *PHPRunner {
	return &PHPRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KIND_PHP, p, s),
		sys:   system.NewPHP(p, s),
	}
}

type PHPRunner struct {
	s     sConfig.Config
	p     pConfig.Config
	sys   *system.PHP
	build *builder.Builder
}

func (r PHPRunner) Disable() error {
	if !r.sys.IsInstalled() {
		log.Print("not installed")
		return nil // nothing to disable
	}
	return r.sys.Uninstall()
}

func (r PHPRunner) Enable() error {
	if !r.p.UsePHP {
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

func (r PHPRunner) Commit() error {
	return r.sys.ReloadIfDirty()
}

func (r PHPRunner) Clean() error {
	return r.build.Clean()
}

func (r PHPRunner) Generate() error {
	if !r.p.UsePHP {
		return nil // nothing to do
	}
	tS, err := r.build.LoadTemplate("php.ini")
	if err != nil {
		return err
	}
	tmplData := struct {
		P pConfig.Config
		S sConfig.Config
	}{
		P: r.p,
		S: r.s,
	}
	return r.build.WriteTemplate("php.ini", tS, tmplData)
}
