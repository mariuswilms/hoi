// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atelierdisko/hoi/builder"
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
	"github.com/coreos/go-systemd/dbus"
)

func NewWorkerRunner(s server.Config, p project.Config, conn *dbus.Conn) *WorkerRunner {
	return &WorkerRunner{
		s:     s,
		p:     p,
		build: builder.NewBuilder(builder.KindWorker, p, s),
		sys:   system.NewSystemd(system.SystemdKindWorker, p, s, conn),
	}
}

// Starts long running worker processes using systemd(1). Uses
// resource controls (i.e. MemoryMax) to keep resource usage of
// processes inside reasonable bounds. This is especially useful if
// processes are leaking memory or otherwise don't behave well. A
// feature desperately missing from alternatives like supervisord.
type WorkerRunner struct {
	s     server.Config
	p     project.Config
	sys   *system.Systemd
	build *builder.Builder
}

// Regex with capturing group to extract unit base name from a templated unit name.
var templatedUnitRegex = regexp.MustCompile(`^(.*)@[0-9]+\.service`)

func (r WorkerRunner) Disable() error {
	units, err := r.sys.ListInstalledServices()
	if err != nil {
		return err
	}
	var lastTemplate string

	for _, u := range units {
		if err := r.sys.StopAndDisable(u); err != nil {
			return err
		}
		// Intentionally not calling Uninstall(), as worker unit files
		// are always derived from templated units, thus do not exist
		// physically.

		// Where a unit using a template is, a template must also exist.
		// As templates are not included in ListInstalledServices we
		// map back manually to clean up.
		//
		// unit name is i.e. worker_media-processor@1.service
		// template name is i.e. worker_media-processor@.service
		matches := templatedUnitRegex.FindStringSubmatch(u)
		if matches == nil {
			return fmt.Errorf("failed to parse unit template name from unit: %s", u)
		}

		// Try only just once to remove the template file, as we want
		// to error out when uninstall fails. There is now way to
		// check if that file exists from the runner.
		if lastTemplate == matches[1] {
			continue
		}
		lastTemplate = matches[1]

		if err := r.sys.Uninstall(lastTemplate + "@.service"); err != nil {
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
		k := filepath.Base(strings.TrimSuffix(f, "@"+filepath.Ext(f)))
		if _, ok := r.p.Worker[k]; !ok {
			return fmt.Errorf("failed to lookup worker by name %s, parsed incorrectly?", k)
		}
		w := r.p.Worker[k]

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
			fmt.Sprintf("%s@.service", v.GetID()),
			tS,
			tmplData,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
