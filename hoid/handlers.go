// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"log"
	"reflect"

	pConfig "github.com/atelierdisko/hoi/config/project"
	"github.com/atelierdisko/hoi/runner"
)

func handleStatus() (map[string]pConfig.Config, error) {
	Store.RLock()
	defer Store.RUnlock()
	return Store.data, nil
}

func handleLoad(pDrv *pConfig.ProjectDirective) error {
	Store.Lock()
	defer Store.Unlock()

	log.Printf("loading project from: %s", pDrv.Path)

	pCfg, err := pConfig.NewFromFile(pDrv.Path + "/Hoifile")
	if err != nil {
		log.Printf("[project %s] failed to parse Hoifile: %s", pCfg.PrettyName(), err)
		return err
	}

	err = pCfg.Augment()
	if err != nil {
		log.Printf("[project %s] failed to discover config: %s", pCfg.PrettyName(), err)
		return err
	}

	for _, r := range runners(*pCfg) {
		log.Printf("[project %s] ------- %s begins steps", pCfg.PrettyName(), reflect.TypeOf(r))

		if err := r.Disable(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), "disable", err)
			return err
		}
		if err := r.Clean(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), "clean", err)
			return err
		}
		if err := r.Generate(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), "generate", err)
			return err
		}
		if err := r.Enable(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), "enable", err)
			return err
		}
		if err := r.Commit(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pCfg.PrettyName(), "commit", err)
			return err
		}
		log.Printf("[project %s] ------- %s finished all steps", pCfg.PrettyName(), reflect.TypeOf(r))
	}

	log.Printf("[project %s] ===== all runners done!", pCfg.PrettyName())

	// Only add to store if all steps finished.
	Store.data[pCfg.ID()] = *pCfg

	log.Printf("[project %s] loaded :)", pCfg.PrettyName())
	return nil
}

func handleUnload(pDrv *pConfig.ProjectDirective) error {
	Store.Lock()
	defer Store.Unlock()

	log.Printf("unloading project: %s", pDrv.ID())

	if _, hasKey := Store.data[pDrv.ID()]; !hasKey {
		return fmt.Errorf("no project %s in store to unload", pDrv.ID())
	}

	for _, r := range runners(Store.data[pDrv.ID()]) {
		if err := r.Disable(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pDrv.PrettyName(), "disable", err)
			return err
		}
		if err := r.Clean(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pDrv.PrettyName(), "clean", err)
			return err
		}
		if err := r.Commit(); err != nil {
			log.Printf("[project %s] step %s failed: %s", pDrv.PrettyName(), "commit", err)
			return err
		}
	}

	log.Printf("[project %s] unloaded :(", pDrv.PrettyName())
	delete(Store.data, pDrv.ID())
	return nil
}

func handleDomain(pDrv *pConfig.ProjectDirective, dDrv *pConfig.DomainDirective) error {
	Store.Lock()
	defer Store.Unlock()

	log.Printf("adding domain %s to project: %s", dDrv.FQDN, pDrv.ID())

	if _, hasKey := Store.data[pDrv.ID()]; !hasKey {
		return fmt.Errorf("no project %s in store to add domain to", pDrv.ID())
	}
	pCfg := Store.data[pDrv.ID()]

	if _, hasKey := pCfg.Domain[dDrv.FQDN]; hasKey {
		el := pCfg.Domain[dDrv.FQDN]
		el.Aliases = append(pCfg.Domain[dDrv.FQDN].Aliases, dDrv.Aliases...)

		pCfg.Domain[dDrv.FQDN] = el
	} else {
		pCfg.Domain[dDrv.FQDN] = *dDrv
	}
	Store.data[pCfg.ID()] = pCfg

	return nil
}

func runners(pCfg pConfig.Config) []runner.Runnable {
	runners := make([]runner.Runnable, 0)

	// FIXME: Order matters?
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

	return runners
}
