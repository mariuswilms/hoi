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
	systemd "github.com/coreos/go-systemd/dbus"
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

	Config *server.Config

	// Connections kept centrally here so they can be shared and
	// reused by runners and systems. Also allows us to manage
	// shutdown/cleanup of connections more easily, as it's clear when
	// these are not needed any more.
	//
	// Connections established as per demand, some may never be
	// setup at all.
	RPCServer   *rpc.Server
	Store       *store.Store
	MySQLConn   *sql.DB
	SystemdConn *systemd.Conn
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
			ProjectAPI: &rpc.ProjectAPI{
				StatusHandler:    handleStatus,
				StatusAllHandler: handleStatusAll,
				LoadHandler:      handleLoad,
				UnloadHandler:    handleUnload,
				ReloadHandler:    handleReload,
				ReloadAllHandler: handleReloadAll,
				DomainHandler:    handleDomain,
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

		// Only connect if we need a connection objects later.
		if Config.Database.Enabled {
			dsn := fmt.Sprintf("%s:%s@tcp(%s)/", Config.MySQL.User, Config.MySQL.Password, Config.MySQL.Host)
			conn, err := sql.Open("mysql", dsn)
			if err != nil {
				log.Fatal(err)
			}
			MySQLConn = conn // Assign to global.
			log.Printf("MySQL connection ready")
		}
		if Config.Cron.Enabled || Config.Worker.Enabled {
			conn, err := systemd.New()
			if err != nil {
				log.Fatal(err)
			}
			SystemdConn = conn // Assign to global
			log.Printf("Systemd DBUS connection ready")
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
			Store.Close()
			RPCServer.Close()

			if MySQLConn != nil {
				MySQLConn.Close()
			}
			if SystemdConn != nil {
				SystemdConn.Close()
			}
			os.Exit(0)
		}
	}(sigc)

	App.Run(os.Args)
	<-make(chan int) // Do not exit.
}
