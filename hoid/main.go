// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
	sRPC "github.com/atelierdisko/hoi/hoid/rpc"
	"github.com/jawher/mow.cli"
)

var (
	App       = cli.App("hoid", "hoid is a host project manager")
	Version   string
	Config    *sConfig.Config
	RPCServer *sRPC.Server
	Store     *MemoryStore
)

type MemoryStore struct {
	sync.RWMutex
	// no pointer as it then would be possible to modify data outside lock
	data map[string]pConfig.Config
}

func (s *MemoryStore) Stats() string {
	Store.RLock()
	out := fmt.Sprintf("STATS STORE count:%d", len(s.data))
	Store.RUnlock()
	return out
}

func main() {
	log.SetFlags(0) // disable prefix, we are invoked directly.

	App.Version("v version", "hoid "+Version)

	socket := App.String(cli.StringOpt{
		Name:   "socket",
		Value:  "/var/run/hoid.socket",
		Desc:   "UNIX socket file",
		EnvVar: "HOID_SOCKET",
	})
	config := App.String(cli.StringOpt{
		Name:  "config",
		Value: "/etc/hoi/hoid.conf",
		Desc:  "server configuration file",
	})

	App.Action = func() {
		cfg, err := sConfig.NewFromFile(*config)
		if err != nil {
			log.Fatal(err)
		}
		Config = cfg // Assign to global.
		log.Printf("loaded configuration from %s", *config)

		rpcServer := &sRPC.Server{
			Socket: *socket,
			ServerAPI: &sRPC.ServerAPI{
				StatusHandler: status,
			},
			ProjectAPI: &sRPC.ProjectAPI{
				LoadHandler:      load,
				AddDomainHandler: addDomain,
			},
		}
		RPCServer = rpcServer // Assign to global.

		if err := RPCServer.Run(); err != nil {
			log.Fatal(err)
		}
		log.Printf("listening for RPC calls on %s", *socket)

		Store = &MemoryStore{
			data: make(map[string]pConfig.Config),
		}
		log.Printf("in-memory store ready")
	}

	// Shutdown gracefully.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)

	go func(c chan os.Signal) {
		sig := <-c
		switch sig {
		case syscall.SIGHUP:
			log.Printf("Caught signal %s: currently noop", sig)
		default:
			log.Printf("Caught signal %s: shutting down", sig)
			RPCServer.Close()
			os.Exit(0)
		}
	}(sigc)

	App.Run(os.Args)
	<-make(chan int) // Do not exit.
}

func status() (map[string]pConfig.Config, error) {
	Store.RLock()
	defer Store.RUnlock()
	return Store.data, nil
}

func load(pDrv *pConfig.ProjectDirective) error {
	log.Printf("[project %s] loading from path %s", "?", prettyPath(pDrv.Path))

	cfg, err := pConfig.NewFromFile(pDrv.Path + "/Hoifile")
	if err != nil {
		log.Printf("[project %s] failed to parse Hoifile: %s", cfg.PrettyName(), err)
		return err
	}

	err = cfg.Augment()
	if err != nil {
		log.Printf("[project %s] failed to discover config: %s", cfg.PrettyName(), err)
		return err
	}

	Store.Lock()
	Store.data[cfg.Id()] = *cfg
	Store.Unlock()

	log.Print(Store.Stats())
	// Now overwrite everything in etc and regenerate everything
	log.Printf("[project %s] loaded", cfg.PrettyName())

	prettyFunctionName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}
	doStep := func(f func(*pConfig.Config, *sConfig.Config) error) {
		if err != nil {
			return
		}
		err = f(cfg, Config)
		if err != nil {
			log.Printf("[project %s] failed to execute step %s: %s", cfg.PrettyName(), prettyFunctionName(f), err)
		} else {
			log.Printf("[project %s] step %s succeeded", cfg.PrettyName(), prettyFunctionName(f))
		}
	}
	doStep(generateWeb)
	doStep(deactivateWeb)
	doStep(activateWeb)
	doStep(generateCron)
	doStep(deactivateCron)
	doStep(activateCron)

	return err
}

func addDomain(pDrv *pConfig.ProjectDirective, dDrv *pConfig.DomainDirective) error {
	Store.Lock()

	if _, hasKey := Store.data[pDrv.Id()]; !hasKey {
		Store.Unlock()
		return fmt.Errorf("no project %s in store to add domain to", pDrv.Id())
	}
	cfg := Store.data[pDrv.Id()]

	if _, hasKey := cfg.Domain[dDrv.FQDN]; hasKey {
		el := cfg.Domain[dDrv.FQDN]
		el.Aliases = append(cfg.Domain[dDrv.FQDN].Aliases, dDrv.Aliases...)

		cfg.Domain[dDrv.FQDN] = el
	} else {
		cfg.Domain[dDrv.FQDN] = *dDrv
	}
	Store.data[cfg.Id()] = cfg

	Store.Unlock()
	return nil
}
