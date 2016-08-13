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

	pConfig "github.com/atelierdisko/hoi/config/project"
	"github.com/atelierdisko/hoi/runner"
)

func handleStatus() (map[string]pConfig.Config, error) {
	Store.RLock()
	defer Store.RUnlock()
	return Store.data, nil
}

func handleLoad(path string) error {
	Store.Lock()
	defer Store.Unlock()

	log.Printf("loading project from: %s", path)

	pCfg, err := pConfig.NewFromFile(path + "/Hoifile")
	if err != nil {
		log.Printf("[project %s] failed to parse Hoifile: %s", pCfg.PrettyName(), err)
		return err
	}

	if err = pCfg.Augment(); err != nil {
		log.Printf("[project %s] failed to discover config: %s", pCfg.PrettyName(), err)
		return err
	}

	if err = pCfg.Validate(); err != nil {
		log.Printf("[project %s] did not validate: %s", pCfg.PrettyName(), err)
		return err
	}

	steps := make([]func() error, 0)
	for _, r := range runners(*pCfg) {
		steps = append(steps, r.Disable)
		steps = append(steps, r.Clean)
		steps = append(steps, r.Build)
		steps = append(steps, r.Enable)
		steps = append(steps, r.Commit)
	}
	if err := performSteps(*pCfg, steps); err != nil {
		return err
	}

	log.Printf("[project %s] loaded :)", pCfg.PrettyName())
	Store.data[pCfg.ID()] = *pCfg
	return nil
}

func handleUnload(path string) error {
	Store.Lock()
	defer Store.Unlock()

	id := pConfig.ProjectPathToID(path)
	log.Printf("unloading project: %s", id)

	if _, hasKey := Store.data[id]; !hasKey {
		return fmt.Errorf("no project %s in store to unload", id)
	}
	pCfg := Store.data[id]

	steps := make([]func() error, 0)
	for _, r := range runners(pCfg) {
		steps = append(steps, r.Disable)
		steps = append(steps, r.Clean)
		steps = append(steps, r.Commit)
	}
	if err := performSteps(pCfg, steps); err != nil {
		return err
	}

	log.Printf("[project %s] unloaded :(", pCfg.PrettyName())
	delete(Store.data, pCfg.ID())
	return nil
}

func handleDomain(path string, dDrv *pConfig.DomainDirective) error {
	Store.Lock()
	defer Store.Unlock()

	id := pConfig.ProjectPathToID(path)
	log.Printf("adding domain %s to project: %s", dDrv.FQDN, id)

	if _, hasKey := Store.data[id]; !hasKey {
		return fmt.Errorf("no project %s in store to add domain to", id)
	}
	pCfg := Store.data[id]

	if _, hasKey := pCfg.Domain[dDrv.FQDN]; hasKey {
		el := pCfg.Domain[dDrv.FQDN]
		el.Aliases = append(pCfg.Domain[dDrv.FQDN].Aliases, dDrv.Aliases...)

		pCfg.Domain[dDrv.FQDN] = el
	} else {
		pCfg.Domain[dDrv.FQDN] = *dDrv
	}

	if err := pCfg.Validate(); err != nil {
		log.Printf("[project %s] did not validate: %s", pCfg.PrettyName(), err)
		return err
	}

	runners := make([]runner.Runnable, 0)
	if Config.Web.Enabled {
		runners = append(runners, runner.NewWebRunner(*Config, pCfg))
	}

	steps := make([]func() error, 0)
	for _, r := range runners {
		steps = append(steps, r.Disable)
		steps = append(steps, r.Clean)
		steps = append(steps, r.Build)
		steps = append(steps, r.Enable)
		steps = append(steps, r.Commit)
	}
	if err := performSteps(pCfg, steps); err != nil {
		return err
	}

	log.Printf("[project %s] added domain: %s", pCfg.PrettyName(), dDrv.FQDN)
	Store.data[pCfg.ID()] = pCfg
	return nil
}

func runners(pCfg pConfig.Config) []runner.Runnable {
	runners := make([]runner.Runnable, 0)

	if Config.Web.Enabled {
		runners = append(runners, runner.NewWebRunner(*Config, pCfg))
	}
	if Config.PHP.Enabled {
		runners = append(runners, runner.NewPHPRunner(*Config, pCfg))
	}
	if Config.Cron.Enabled {
		runners = append(runners, runner.NewCronRunner(*Config, pCfg))
	}
	if Config.Worker.Enabled {
		runners = append(runners, runner.NewWorkerRunner(*Config, pCfg))
	}
	if Config.Database.Enabled {
		runners = append(runners, runner.NewDBRunner(*Config, pCfg, MySQLConn))
	}

	return runners
}

func performSteps(pCfg pConfig.Config, steps []func() error) error {
	getFuncName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}
	for _, s := range steps {
		if err := s(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), getFuncName(s), err)
			return err
		}
	}
	return nil
}
