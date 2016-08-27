// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"reflect"
	"runtime"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/runner"
	"github.com/atelierdisko/hoi/store"
)

func handleStatus() ([]store.Entity, error) {
	return Store.ReadAll(), nil
}

func handleLoad(path string) error {
	log.Printf("loading project from: %s", path)

	pCfg, err := project.NewFromFile(path + "/Hoifile")
	if err != nil {
		return fmt.Errorf("failed to parse Hoifile in project %s: %s", pCfg.PrettyName(), err)
	}

	if err = pCfg.Augment(); err != nil {
		return fmt.Errorf("failed to discover configuration of project %s: %s", pCfg.PrettyName(), err)
	}

	if err = pCfg.Validate(); err != nil {
		return fmt.Errorf("cannot load project %s, config did not validate: %s", pCfg.PrettyName(), err)
	}

	steps := make([]func() error, 0)
	for _, r := range runners(*pCfg) {
		steps = append(
			steps,
			r.Disable,
			r.Clean,
			r.Build,
			r.Enable,
			r.Commit,
		)
	}

	if err := Store.Write(pCfg.ID(), *pCfg); err != nil {
		return err
	}
	Store.WriteStatus(pCfg.ID(), project.StatusLoading)

	if err := performSteps(*pCfg, steps); err != nil {
		Store.WriteStatus(pCfg.ID(), project.StatusFailed)
		return err
	}

	log.Printf("project %s is now active :)", pCfg.PrettyName())
	Store.WriteStatus(pCfg.ID(), project.StatusActive)
	return nil
}

func handleUnload(path string) error {
	id := project.ProjectPathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store to unload", id)
	}
	log.Printf("unloading project %s", id)
	Store.WriteStatus(id, project.StatusUnloading)

	pCfg, err := Store.Read(id)
	if err != nil {
		return err
	}

	steps := make([]func() error, 0)
	for _, r := range runners(pCfg) {
		steps = append(
			steps,
			r.Disable,
			r.Clean,
			r.Commit,
		)
	}

	if err := Store.Delete(pCfg.ID()); err != nil {
		Store.WriteStatus(pCfg.ID(), project.StatusFailed)
		return err
	}
	if err := performSteps(pCfg, steps); err != nil {
		Store.WriteStatus(pCfg.ID(), project.StatusFailed)
		return err
	}

	log.Printf("project %s unloaded :(", pCfg.PrettyName())
	return nil
}

func handleDomain(path string, dDrv *project.DomainDirective) error {
	id := project.ProjectPathToID(path)

	if !Store.Has(id) {
		return fmt.Errorf("no project %s in store to add domain to", id)
	}
	log.Printf("adding domain %s to project: %s", dDrv.FQDN, id)
	pCfg, _ := Store.Read(id)

	if _, hasKey := pCfg.Domain[dDrv.FQDN]; hasKey {
		el := pCfg.Domain[dDrv.FQDN]
		el.Aliases = append(pCfg.Domain[dDrv.FQDN].Aliases, dDrv.Aliases...)

		pCfg.Domain[dDrv.FQDN] = el
	} else {
		pCfg.Domain[dDrv.FQDN] = *dDrv
	}

	if err := pCfg.Validate(); err != nil {
		return fmt.Errorf("cannot load project %s, config did not validate: %s", pCfg.PrettyName(), err)
	}

	// Save us iterating through all runners, when the only one
	// needed for domain updates is the web runner.
	runners := make([]runner.Runnable, 0)
	if Config.Web.Enabled {
		runners = append(runners, runner.NewWebRunner(*Config, pCfg))
	}

	steps := make([]func() error, 0)
	for _, r := range runners {
		steps = append(
			steps,
			r.Disable,
			r.Clean,
			r.Build,
			r.Enable,
			r.Commit,
		)
	}

	if err := Store.Write(pCfg.ID(), pCfg); err != nil {
		return err
	}
	Store.WriteStatus(pCfg.ID(), project.StatusUpdating)

	if err := performSteps(pCfg, steps); err != nil {
		Store.WriteStatus(pCfg.ID(), project.StatusFailed)
		return err
	}

	log.Printf("added domain %s to projects %s", dDrv.FQDN, pCfg.PrettyName())
	Store.WriteStatus(pCfg.ID(), project.StatusActive)
	return nil
}

func runners(pCfg project.Config) []runner.Runnable {
	runners := make([]runner.Runnable, 0)

	if Config.PHP.Enabled {
		runners = append(runners, runner.NewPHPRunner(*Config, pCfg))
	}
	if Config.Database.Enabled {
		runners = append(runners, runner.NewDBRunner(*Config, pCfg, MySQLConn))
	}
	if Config.Web.Enabled {
		runners = append(runners, runner.NewWebRunner(*Config, pCfg))
	}
	if Config.Cron.Enabled {
		runners = append(runners, runner.NewCronRunner(*Config, pCfg))
	}
	if Config.Worker.Enabled {
		runners = append(runners, runner.NewWorkerRunner(*Config, pCfg))
	}

	return runners
}

func performSteps(pCfg project.Config, steps []func() error) error {
	getFuncName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}
	for _, s := range steps {
		if err := s(); err != nil {
			return fmt.Errorf("in project %s step %s failed: %s", pCfg.PrettyName(), getFuncName(s), err)
		}
	}
	return nil
}
