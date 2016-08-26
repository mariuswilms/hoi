// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The server command component.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/atelierdisko/hoi/rpc"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/store"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jawher/mow.cli"
)

var (
	App = cli.App("hoid", "hoid is a host project manager")

	// Set via ldflags.
	Version    string // hoi version
	ConfigPath string // path to configuration file
	SocketPath string // path to socket for RPC
	DataPath   string // path to store database file

	Config    *server.Config
	RPCServer *rpc.Server
	Store     *store.Store
	MySQLConn *sql.DB
)

func main() {
	log.SetFlags(0) // disable prefix, we are invoked directly.

	App.Version("v version", "hoid "+Version)

	App.Action = func() {
		cfg, err := server.NewFromFile(ConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		Config = cfg // Assign to global.

		rpcServer := &rpc.Server{
			Socket: SocketPath,
			ServerAPI: &rpc.ServerAPI{
				StatusHandler: handleStatus,
			},
			ProjectAPI: &rpc.ProjectAPI{
				LoadHandler:   handleLoad,
				UnloadHandler: handleUnload,
				DomainHandler: handleDomain,
			},
		}
		RPCServer = rpcServer // Assign to global.
		if err := RPCServer.Run(); err != nil {
			log.Fatal(err)
		}

		_store := store.New(DataPath)
		if err := _store.Load(); err != nil {
			log.Fatal(err)
		}
		_store.InstallAutoStore()
		Store = _store // Assign to global
		log.Printf("store backend ready")

		// Only connect if we need a connection later.
		if Config.Database.Enabled {
			dsn := fmt.Sprintf("%s:%s@tcp(%s)/", Config.MySQL.User, Config.MySQL.Password, Config.MySQL.Host)
			conn, err := sql.Open("mysql", dsn)
			if err != nil {
				log.Fatal(err)
			}
			MySQLConn = conn // Assign to global.
			log.Printf("MySQL connection ready")
		}
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
			Store.Lock()
			Store.Close()
			Store.Unlock()

			RPCServer.Close()

			if MySQLConn != nil {
				MySQLConn.Close()
			}
			os.Exit(0)
		}
	}(sigc)

	App.Run(os.Args)
	<-make(chan int) // Do not exit.
}
