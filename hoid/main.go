// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
	sRPC "github.com/atelierdisko/hoi/rpc"
	"github.com/jawher/mow.cli"
)

var (
	App = cli.App("hoid", "hoid is a host project manager")

	// Set via ldflags.
	Version    string
	ConfigPath string
	SocketPath string

	Config    *sConfig.Config
	RPCServer *sRPC.Server
	Store     *MemoryStore
)

func main() {
	log.SetFlags(0) // disable prefix, we are invoked directly.

	App.Version("v version", "hoid "+Version)

	App.Action = func() {
		cfg, err := sConfig.NewFromFile(ConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		Config = cfg // Assign to global.
		log.Printf("loaded configuration: %s", ConfigPath)

		rpcServer := &sRPC.Server{
			Socket: SocketPath,
			ServerAPI: &sRPC.ServerAPI{
				StatusHandler: handleStatus,
			},
			ProjectAPI: &sRPC.ProjectAPI{
				LoadHandler:   handleLoad,
				UnloadHandler: handleUnload,
				DomainHandler: handleDomain,
			},
		}
		RPCServer = rpcServer // Assign to global.

		if err := RPCServer.Run(); err != nil {
			log.Fatal(err)
		}
		log.Printf("listening for RPC calls on: %s", SocketPath)

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
			log.Printf("caught signal %s: currently noop", sig)
		default:
			log.Printf("caught signal %s: shutting down", sig)
			RPCServer.Close()
			os.Exit(0)
		}
	}(sigc)

	App.Run(os.Args)
	<-make(chan int) // Do not exit.
}
