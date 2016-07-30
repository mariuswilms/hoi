// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sRPC "github.com/atelierdisko/hoi/hoid/rpc"
	"github.com/jawher/mow.cli"
)

var (
	App     = cli.App("hoictl", "hoictl is the command line interface to hoid")
	Version string
	// Searches current than parent directories until it finds Hoifile or
	// reaches root.
	RPCClient *rpc.Client
)

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
	log.Fatalf("Not able to detect project directory")
	return ""
}

func main() {
	log.SetFlags(0) // disable prefix, we are invoked directly.

	App.Version("v version", "hoictl "+Version)

	socket := App.String(cli.StringOpt{
		Name:   "socket",
		Value:  "/var/run/hoid.socket",
		Desc:   "UNIX socket file",
		EnvVar: "HOID_SOCKET",
	})

	path := App.String(cli.StringOpt{
		Name:  "project",
		Desc:  "path to project root",
		Value: ".",
	})

	App.Before = func() {
		client, err := rpc.Dial("unix", *socket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
		RPCClient = client // Assign to global.
	}

	App.Command("status", "show status", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			args := &sRPC.ServerAPIArgs{}
			var reply map[string]pConfig.Config
			err := RPCClient.Call("Server.Status", args, &reply)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}

			if len(reply) != 0 {
				fmt.Print("Projects:\n")
				for _, p := range reply {
					fmt.Printf("- %-20s in %s\n", p.PrettyName(), prettyPath(p.Path))
					fmt.Printf("  **%s**\n", "loaded")
					fmt.Printf("  %-10s: %d\n", "domain", len(p.Domain))
					fmt.Printf("  %-10s: %d\n", "cron", len(p.Cron))
					fmt.Printf("  %-10s: %d\n", "worker", len(p.Worker))

					// fmt.Printf("%#v\n\n", p)
				}
			} else {
				fmt.Println("no projects loaded :(")
			}
		}
	})

	App.Command("load", "initialize or update a project's configuration using a Hoifile", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			args := &sRPC.ProjectAPIArgs{
				Project: &pConfig.ProjectDirective{
					Path: projectDirectory(*path),
				},
			}
			var reply bool
			err := RPCClient.Call("Project.Load", args, &reply)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("project loaded")
		}
	})

	App.Command("enable", "enables a project", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			args := &sRPC.ProjectAPIArgs{
				Project: &pConfig.ProjectDirective{
					Path: projectDirectory(*path),
				},
			}
			var reply bool
			err := RPCClient.Call("Project.Enable", args, &reply)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("project enabled")
		}
	})

	App.Command("domain", "adds or modifies a domain configuration", func(cmd *cli.Cmd) {
		fqdn := cmd.StringArg("FQDN", "", "")

		www := cmd.String(cli.StringOpt{
			Name:  "www",
			Value: "keep",
			Desc:  "either _drop_, _add_ or _keep_ www prefix untouched",
		})
		aliases := cmd.Strings(cli.StringsOpt{
			Name: "alias",
			Desc: "alias for the domain (repeat for multiple)",
		})

		cmd.Action = func() {
			args := &sRPC.ProjectAPIArgs{
				Project: &pConfig.ProjectDirective{
					Path: projectDirectory(*path),
				},
				Domain: &pConfig.DomainDirective{
					FQDN:    *fqdn,
					WWW:     *www,
					Aliases: *aliases,
				},
			}
			var reply bool
			err := RPCClient.Call("Project.Domain", args, &reply)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("domain added/modified in project")
		}
	})

	App.Run(os.Args)
}

func prettyPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	return strings.Replace(path, cwd, ".", 1)
}
