// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The client command component. Interfaces with the
// server via an UNIX socket.
package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"path/filepath"

	"github.com/atelierdisko/hoi/project"
	sRPC "github.com/atelierdisko/hoi/rpc"
	"github.com/atelierdisko/hoi/store"
	"github.com/jawher/mow.cli"
)

var (
	App = cli.App("hoictl", "hoictl is the command line interface to hoid")

	// Set via ldflags.
	Version    string
	SocketPath string

	RPCClient *rpc.Client
)

// Searches current than parent directories until it finds Hoifile or
// reaches root.
func projectDirectory(path string) string {
	if path == "." {
		path, _ = os.Getwd()
	} else {
		path, _ = filepath.Abs(path)
	}

	for path != "." {
		if _, err := os.Stat(path + "/Hoifile"); err == nil {
			return path
		}
		path, err := filepath.Abs(path + "/..")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", "not able to detect project directory")
			os.Exit(1)
		}
		return path
	}
	log.Fatal("not able to detect project directory")
	return ""
}

func main() {
	log.SetFlags(0) // disable prefix, we are invoked directly.

	App.Version("v version", "hoictl "+Version)

	// Overload commands to operate on a single (default) or multiple
	// (all) projects.
	App.Spec = "[--project | --all]"

	// TODO: Move into commands?
	path := App.String(cli.StringOpt{
		Name:  "project",
		Desc:  "path to a single project",
		Value: ".",
	})
	all := App.Bool(cli.BoolOpt{
		Name: "all",
		Desc: "operate on all projects",
	})

	App.Before = func() {
		client, err := rpc.Dial("unix", SocketPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed, got error: %s\n", err)
			os.Exit(1)
		}
		RPCClient = client // Assign to global.
	}

	App.Command("status", "show status", func(cmd *cli.Cmd) {
		cmd.Action = func() {

			if *all {
				args := &sRPC.ProjectAPIArgs{}
				var reply []store.Entity

				if err := RPCClient.Call("Project.StatusAll", args, &reply); err != nil {
					fmt.Fprintf(os.Stderr, "failed, got error: %s\n", err)
					os.Exit(1)
				}
				if len(reply) <= 0 {
					fmt.Println("no projects loaded, yet")
					os.Exit(0)
				}
				fmt.Printf("%d total project/s loaded\n\n", len(reply))

				for _, e := range reply {
					printProject(e)
					fmt.Print("\n")
				}
				return
			}
			args := &sRPC.ProjectAPIArgs{
				Path: projectDirectory(*path),
			}
			var reply store.Entity
			if err := RPCClient.Call("Project.Status", args, &reply); err != nil {
				fmt.Fprintf(os.Stderr, "failed, got error: %s\n", err)
				os.Exit(1)
			}
			printProject(reply)
		}
	})

	App.Command("load", "loads project configuration", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			args := &sRPC.ProjectAPIArgs{
				Path: projectDirectory(*path),
			}
			var reply bool

			if err := RPCClient.Call("Project.Load", args, &reply); err != nil {
				fmt.Fprintf(os.Stderr, "failed, got error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("project successfully loaded :)")
		}
	})

	App.Command("reload", "reloads project configuration", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			var reply bool

			if *all {
				args := &sRPC.ProjectAPIArgs{}
				if err := RPCClient.Call("Project.ReloadAll", args, &reply); err != nil {
					fmt.Fprintf(os.Stderr, "failed reloading, got error: %s\n", err)
					os.Exit(1)
				}
				fmt.Println("all projects successfully reloaded :)")
			} else {
				args := &sRPC.ProjectAPIArgs{
					Path: projectDirectory(*path),
				}
				if err := RPCClient.Call("Project.Reload", args, &reply); err != nil {
					fmt.Fprintf(os.Stderr, "failed reloading, got error: %s\n", err)
					os.Exit(1)
				}
				fmt.Println("project successfully reloaded :)")
			}
		}
	})

	App.Command("unload", "removes project configuration", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			var reply bool

			if *all {
				args := &sRPC.ProjectAPIArgs{}
				if err := RPCClient.Call("Project.UnloadAll", args, &reply); err != nil {
					fmt.Fprintf(os.Stderr, "failed unloading, got error: %s\n", err)
					os.Exit(1)
				}
				fmt.Println("all projects successfully unloaded :(")
			} else {
				args := &sRPC.ProjectAPIArgs{
					Path: projectDirectory(*path),
				}
				if err := RPCClient.Call("Project.Unload", args, &reply); err != nil {
					fmt.Fprintf(os.Stderr, "failed unloading, got error: %s\n", err)
					os.Exit(1)
				}
				fmt.Println("project successfully unloaded :(")
			}
		}
	})

	App.Command("domain", "adds or modifies domain configuration", func(cmd *cli.Cmd) {
		fqdn := cmd.StringArg("FQDN", "", "")

		www := cmd.String(cli.StringOpt{
			Name:  "www",
			Value: "keep",
			Desc:  "either _drop_, _add_ or _keep_ www prefix untouched",
		})
		aliases := cmd.Strings(cli.StringsOpt{
			Name: "alias",
			Desc: "alias for the domain (repeat for multiple), when FQDN exists merges with present aliases",
		})

		cmd.Action = func() {
			args := &sRPC.DomainAPIArgs{
				Path: projectDirectory(*path),
				Domain: &project.DomainDirective{
					FQDN:    *fqdn,
					WWW:     *www,
					Aliases: *aliases,
				},
			}
			var reply bool

			if err := RPCClient.Call("Project.Domain", args, &reply); err != nil {
				fmt.Fprintf(os.Stderr, "failed, got error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("domain added/modified in project")
		}
	})

	App.Command("dump", "exports databases and persistent volumes", func(cmd *cli.Cmd) {
		targetArg := cmd.StringArg("FILE", "", "The name and path under which to store the dump.")
		var target string

		if !filepath.IsAbs(*targetArg) {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get current working directory: %s\n", err)
				os.Exit(1)
			}
			target = filepath.Join(wd, *targetArg)
		} else {
			target = *targetArg
		}

		cmd.Action = func() {
			var reply bool

			if *all {
				fmt.Fprint(os.Stderr, "dumping all projects is not supported")
				os.Exit(1)
			}

			args := &sRPC.DumpAPIArgs{
				Path: projectDirectory(*path),
				File: target,
			}
			if err := RPCClient.Call("Project.Dump", args, &reply); err != nil {
				fmt.Fprintf(os.Stderr, "failed dumping, got error: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("project successfully dumped: %s created\n", target)
		}
	})

	App.Run(os.Args)
}
