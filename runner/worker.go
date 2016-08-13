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
)

type WorkerRunner struct {
	s     server.Config
	p     project.Config
	sys   *system.Systemd
	build *builder.Builder
}

func NewWorkerRunner(s server.Config, p project.Config) *WorkerRunner {
	return &WorkerRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KIND_WORKER, p, s),
		sys:   system.NewSystemd(system.SYSTEMD_KIND_WORKER, p, s),
	}
}

func (r WorkerRunner) Disable() error {
	units, err := r.sys.ListInstalledServices()
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
	return nil
}

func (r WorkerRunner) Enable() error {
	if len(r.p.Worker) == 0 {
		return nil // nothing to do
	}
	files, err := r.build.ListAvailable()
	if err != nil {
		return err
	}
	for _, f := range files {
		// Map back to worker directive, we need this to get instances.
		w := r.p.Worker[strings.TrimSuffix(f, filepath.Ext(f))]

		if err := r.sys.Install(f); err != nil {
			return err
		}

		// Using service template to start n number of instances of the service.
		// http://serverfault.com/questions/730239/start-n-processes-with-one-systemd-service-file
		for i := uint(1); i <= w.GetInstances(); i++ {
			// By simply replacing, we safe us the headaches of matching the file name we
			// do not exactly know.
			unit := strings.Replace(filepath.Base(f), "@.service", fmt.Sprintf("@%d.service", i), 1)

			if err := r.sys.EnableAndStart(unit); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r WorkerRunner) Commit() error {
	return nil
}

func (r WorkerRunner) Clean() error {
	return r.build.Clean()
}

func (r WorkerRunner) Build() error {
	if len(r.p.Worker) == 0 {
		return nil // nothing to do
	}
	tS, err := r.build.LoadTemplate("default@.service")
	if err != nil {
		return err
	}
	for _, v := range r.p.Worker {
		parsed, err := v.GetCommand(r.p)
		if err != nil {
			return err
		}
		v.Command = parsed

		tmplData := struct {
			P project.Config
			S server.Config
			W project.WorkerDirective
		}{
			P: r.p,
			S: r.s,
			W: v,
		}
		err = r.build.WriteTemplate(
			fmt.Sprintf("%s@.service", v.ID()),
			tS,
			tmplData,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
