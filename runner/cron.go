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
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
	systemd "github.com/coreos/go-systemd/dbus"
)

func NewCronRunner(s *server.Config, p *project.Config, conn *systemd.Conn) *CronRunner {
	return &CronRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KindCron, p, s),
		sys:   system.NewSystemd(system.SystemdKindCron, p, s, conn),
	}
}

// Starts cron jobs using systemd(1) timers and will randomize
// startups to reduce resource congestion.
type CronRunner struct {
	s     *server.Config
	p     *project.Config
	sys   *system.Systemd
	build *builder.Builder
}

func (r CronRunner) Disable() error {
	timers, err := r.sys.ListInstalledTimers()
	if err != nil {
		return err
	}
	for _, uT := range timers {
		if err := r.sys.StopAndDisable(uT); err != nil {
			return err
		}
		if err := r.sys.Uninstall(uT); err != nil {
			return err
		}

		// We cannot list service units for timers via
		// ListInstalledServices(), as they are never enabled.
		uS := strings.Replace(uT, ".timer", ".service", 1)

		// Especially long running Services might currently still be running,
		// kill them first.
		if err := r.sys.Stop(uS); err != nil {
			return err
		}

		// Just the timer units must be disabled the accompanying
		// service units must not.
		if err := r.sys.Uninstall(uS); err != nil {
			return err
		}
	}
	return r.build.Clean()
}

func (r CronRunner) Enable() error {
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
			P *project.Config
			S *server.Config
			C project.CronDirective
		}{
			P: r.p,
			S: r.s,
			C: v,
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s.service", v.GetID()),
			tS,
			tmplData,
		)
		if err != nil {
			return err
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s.timer", v.GetID()),
			tT,
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
		if strings.HasSuffix(f, ".timer") {
			if err := r.sys.EnableAndStart(filepath.Base(f)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r CronRunner) Commit() error {
	return nil
}
