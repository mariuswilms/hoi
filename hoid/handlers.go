// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/runner"
	"github.com/atelierdisko/hoi/store"
)

func handleStatus(path string) (store.Entity, error) {
	return Store.Read(project.PathToID(path))
}

func handleStatusAll() ([]store.Entity, error) {
	return Store.ReadAll(), nil
}

func handleLoad(path string) error {
	log.Printf("loading project from: %s", path)

	pCfg, err := project.NewFromFile(path + "/Hoifile")
	if err != nil {
		return fmt.Errorf("failed to parse Hoifile in project %s: %s", pCfg.PrettyName(), err)
	}
	if err = pCfg.Augment(); err != nil {
		return fmt.Errorf("failed to augment config in project %s: %s", pCfg.PrettyName(), err)
	}
	if err = pCfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config in project %s: %s", pCfg.PrettyName(), err)
	}

	steps := make([]func() error, 0)
	for _, r := range runners(pCfg) {
		steps = append(
			steps,
			r.Disable,
			r.Enable,
			r.Commit,
		)
	}

	if err := Store.Write(pCfg.ID, pCfg); err != nil {
		return err
	}
	Store.WriteStatus(pCfg.ID, project.StatusLoading)

	if err := performSteps(pCfg, steps); err != nil {
		Store.WriteStatus(pCfg.ID, project.StatusFailed)
		return fmt.Errorf("failed to load project %s: %s", pCfg.PrettyName(), err)
	}

	log.Printf("project %s is now active :)", pCfg.PrettyName())
	Store.WriteStatus(pCfg.ID, project.StatusActive)
	return nil
}

func handleUnload(path string) error {
	id := project.PathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store", id)
	}
	Store.WriteStatus(id, project.StatusUnloading)

	e, err := Store.Read(id)
	if err != nil {
		return fmt.Errorf("failed unloading project, cannot read id %s from store: %s", id, err)
	}

	steps := make([]func() error, 0)
	for _, r := range runners(e.Project) {
		steps = append(
			steps,
			r.Disable,
			r.Commit,
		)
	}

	if err := performSteps(e.Project, steps); err != nil {
		Store.WriteStatus(e.Project.ID, project.StatusFailed)
		return fmt.Errorf("failed to unload project %s: %s", e.Project.PrettyName(), err)
	}
	if err := Store.Delete(e.Project.ID); err != nil {
		Store.WriteStatus(e.Project.ID, project.StatusFailed)
		return err
	}

	log.Printf("project %s unloaded :(", e.Project.PrettyName())
	return nil
}

func handleUnloadAll() error {
	for _, e := range Store.ReadAll() {
		Store.WriteStatus(e.Project.ID, project.StatusUnloading)

		steps := make([]func() error, 0)
		for _, r := range runners(e.Project) {
			steps = append(
				steps,
				r.Disable,
				r.Commit,
			)
		}
		if err := performSteps(e.Project, steps); err != nil {
			Store.WriteStatus(e.Project.ID, project.StatusFailed)
			return fmt.Errorf("failed to unload project %s: %s", e.Project.PrettyName(), err)
		}
		if err := Store.Delete(e.Project.ID); err != nil {
			Store.WriteStatus(e.Project.ID, project.StatusFailed)
			return err
		}
	}

	log.Printf("all projects unloaded :(")
	return nil
}

func handleReload(path string) error {
	id := project.PathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store", id)
	}

	e, err := Store.Read(id)
	if err != nil {
		return fmt.Errorf("failed to read project at path %s from store: %s", path, err)
	}

	pCfg, err := project.NewFromFile(e.Project.Path + "/Hoifile")
	if err != nil {
		return fmt.Errorf("failed to parse Hoifile in project %s: %s", pCfg.PrettyName(), err)
	}
	if err = pCfg.Augment(); err != nil {
		return fmt.Errorf("failed to augment config in project %s: %s", pCfg.PrettyName(), err)
	}
	if err = pCfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config in project %s: %s", pCfg.PrettyName(), err)
	}

	steps := make([]func() error, 0)
	for _, r := range runners(pCfg) {
		steps = append(
			steps,
			r.Disable,
			r.Enable,
			r.Commit,
		)
	}

	if err := Store.Write(pCfg.ID, pCfg); err != nil {
		return err
	}
	Store.WriteStatus(pCfg.ID, project.StatusReloading)

	if err := performSteps(pCfg, steps); err != nil {
		Store.WriteStatus(pCfg.ID, project.StatusFailed)
		return fmt.Errorf("failed to reload project %s: %s", pCfg.PrettyName(), err)
	}

	log.Printf("project %s reloaded", pCfg.PrettyName())
	Store.WriteStatus(pCfg.ID, project.StatusActive)
	return nil
}

func handleReloadAll() error {
	for _, e := range Store.ReadAll() {
		pCfg, err := project.NewFromFile(e.Project.Path + "/Hoifile")
		if err != nil {
			return fmt.Errorf("failed to parse Hoifile in project %s: %s", pCfg.PrettyName(), err)
		}
		if err = pCfg.Augment(); err != nil {
			return fmt.Errorf("failed to augment config in project %s: %s", pCfg.PrettyName(), err)
		}
		if err = pCfg.Validate(); err != nil {
			return fmt.Errorf("failed to validate config in project %s: %s", pCfg.PrettyName(), err)
		}
		if err := Store.Write(pCfg.ID, pCfg); err != nil {
			return err
		}
		Store.WriteStatus(pCfg.ID, project.StatusReloading)

		steps := make([]func() error, 0)
		for _, r := range runners(pCfg) {
			steps = append(
				steps,
				r.Disable,
				r.Enable,
				r.Commit,
			)
		}
		if err := performSteps(pCfg, steps); err != nil {
			Store.WriteStatus(pCfg.ID, project.StatusFailed)
			return fmt.Errorf("failed to reload project %s: %s", pCfg.PrettyName(), err)
		}
		Store.WriteStatus(pCfg.ID, project.StatusActive)
	}

	log.Printf("all projects reloaded")
	return nil
}

func handleDomain(path string, dDrv *project.DomainDirective) error {
	id := project.PathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store", id)
	}
	e, _ := Store.Read(id)

	if _, hasKey := e.Project.Domain[dDrv.FQDN]; hasKey {
		el := e.Project.Domain[dDrv.FQDN]
		el.AddAliases(dDrv.Aliases...)
		el.WWW = dDrv.WWW

		e.Project.Domain[dDrv.FQDN] = el
	} else {
		e.Project.Domain[dDrv.FQDN] = *dDrv
	}

	if err := e.Project.Validate(); err != nil {
		return fmt.Errorf("failed adding/modifying domain %s for project %s, config did not validate: %s", dDrv.FQDN, e.Project.PrettyName(), err)
	}

	// Save us iterating through all runners, when the only one
	// needed for domain updates is the web runner.
	runners := make([]runner.Runnable, 0)
	if Config.Web.Enabled {
		runners = append(runners, runner.NewWebRunner(Config, e.Project, SystemdConn))
	}

	steps := make([]func() error, 0)
	for _, r := range runners {
		steps = append(
			steps,
			r.Disable,
			r.Enable,
			r.Commit,
		)
	}

	if err := Store.Write(e.Project.ID, e.Project); err != nil {
		return err
	}
	Store.WriteStatus(e.Project.ID, project.StatusUpdating)

	if err := performSteps(e.Project, steps); err != nil {
		Store.WriteStatus(e.Project.ID, project.StatusFailed)
		return fmt.Errorf("failed to add/modify domain %s for project %s: %s", dDrv.FQDN, e.Project.PrettyName(), err)
	}

	log.Printf("added/modified domain %s for projects %s", dDrv.FQDN, e.Project.PrettyName())
	Store.WriteStatus(e.Project.ID, project.StatusActive)
	return nil
}

func handleDump(path string, target string) error {
	id := project.PathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store", id)
	}
	e, _ := Store.Read(id)

	dumpers := make([]runner.Dumper, 0)

	if Config.Volume.Enabled {
		dumpers = append(dumpers, runner.NewVolumeRunner(Config, e.Project, SystemdConn))
	}
	if Config.Database.Enabled {
		dumpers = append(dumpers, runner.NewDBRunner(Config, e.Project, MySQLConn))
	}

	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to dump project %s into %s: %s", e.Project.PrettyName(), target, err)
	}
	tw := tar.NewWriter(file)

	defer file.Close()
	defer tw.Close()

	for _, r := range dumpers {
		if err := r.Dump(tw); err != nil {
			return fmt.Errorf("failed to dump project %s: %s", e.Project.PrettyName(), err)
		}
	}
	return nil
}

func runners(pCfg *project.Config) []runner.Runnable {
	runners := make([]runner.Runnable, 0)

	if Config.Volume.Enabled && len(pCfg.Volume) > 0 {
		runners = append(runners, runner.NewVolumeRunner(Config, pCfg, SystemdConn))
	}
	if Config.Database.Enabled {
		runners = append(runners, runner.NewDBRunner(Config, pCfg, MySQLConn))
	}
	if Config.PHP.Enabled {
		runners = append(runners, runner.NewPHPRunner(Config, pCfg, SystemdConn))
	}
	if Config.AppService.Enabled {
		runners = append(runners, runner.NewAppServiceRunner(Config, pCfg, SystemdConn))
	}
	if Config.Web.Enabled && len(pCfg.Domain) > 0 {
		runners = append(runners, runner.NewWebRunner(Config, pCfg, SystemdConn))
	}
	if Config.Cron.Enabled && len(pCfg.Cron) > 0 {
		runners = append(runners, runner.NewCronRunner(Config, pCfg, SystemdConn))
	}
	if Config.Worker.Enabled && len(pCfg.Worker) > 0 {
		runners = append(runners, runner.NewWorkerRunner(Config, pCfg, SystemdConn))
	}

	return runners
}

func performSteps(pCfg *project.Config, steps []func() error) error {
	getFuncName := func(i interface{}) string {
		name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
		// github.com/atelierdisko/hoi/runner.(Runnable).Build-fm

		// extract just the "Build" part from the fully qualified fn name
		re := regexp.MustCompile("\\.([A-Za-z]+)-fm$")
		match := re.FindStringSubmatch(name)

		if len(match) < 2 {
			return name
		}
		return strings.ToLower(match[1])
	}
	for _, s := range steps {
		if err := s(); err != nil {
			return fmt.Errorf("the *%s* step failed: %s", getFuncName(s), err)
		}
	}
	return nil
}
