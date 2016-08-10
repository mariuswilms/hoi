// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/atelierdisko/hoi/builder"
	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
	"github.com/atelierdisko/hoi/system"
)

func NewCronRunner(s sConfig.Config, p pConfig.Config) *CronRunner {
	return &CronRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KIND_CRON, p, s),
		sys:   system.NewSystemd(system.SYSTEMD_KIND_CRON, p, s),
	}
}

type CronRunner struct {
	s     sConfig.Config
	p     pConfig.Config
	sys   *system.Systemd
	build *builder.Builder
}

func (r CronRunner) Disable() error {
	timers, err := r.sys.ListInstalledTimers()
	if err != nil {
		return err
	}
	for _, u := range timers {
		if err := r.sys.StopAndDisable(u); err != nil {
			return err
		}
		if err := r.sys.Uninstall(u); err != nil {
			return err
		}
	}

	services, err := r.sys.ListInstalledServices()
	if err != nil {
		return err
	}
	for _, u := range services {
		// Services might be currently running, kill them first.
		if err := r.sys.Stop(u); err != nil {
			return err
		}
		if err := r.sys.Uninstall(u); err != nil {
			return err
		}
	}
	return nil
}

func (r CronRunner) Enable() error {
	if len(r.p.Cron) == 0 {
		return nil // nothing to do
	}
	files, err := r.build.ListAvailable()
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := r.sys.Install(f); err != nil {
			return err
		}
		if strings.HasSuffix(f, ".timer") {
			if err := r.sys.EnableAndStart(filepath.Base(f)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r CronRunner) Clean() error {
	return r.build.Clean()
}

func (r CronRunner) Generate() error {
	if len(r.p.Cron) == 0 {
		return nil // nothing to do
	}

	tS, err := r.build.LoadTemplate("default.service")
	if err != nil {
		return err
	}
	tT, err := r.build.LoadTemplate("default.timer")
	if err != nil {
		return err
	}
	for _, v := range r.p.Cron {
		parsed, err := v.GetCommand(r.p)
		if err != nil {
			return err
		}
		v.Command = parsed

		tmplData := struct {
			P pConfig.Config
			S sConfig.Config
			C pConfig.CronDirective
		}{
			P: r.p,
			S: r.s,
			C: v,
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s.service", v.ID()),
			tS,
			tmplData,
		)
		if err != nil {
			return err
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s.timer", v.ID()),
			tT,
			tmplData,
		)
		if err != nil {
			return err
		}
	}
	return nil
}