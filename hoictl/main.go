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

// TODO: hoictl workers start/stop
// TODO: hoictl crons start/stop
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
			// Outputs information about a project entity, vis a vis the systemctl output.
			//
			//● nginx.service - A high performance web server and a reverse proxy server
			//   Loaded: loaded (/lib/systemd/system/nginx.service; enabled)
			//   Active: active (running) since Thu 2016-10-13 14:07:20 CEST; 8h ago
			//  Process: 15315 ExecReload=/usr/sbin/nginx -g daemon on; master_process on; -s reload (code=exited, status=0/SUCCESS)
			// Main PID: 1693 (nginx)
			//   CGroup: /system.slice/nginx.service
			//           ├─ 1693 nginx: master process /usr/sbin/nginx -g daemon on; master_process on;
			//           ├─15320 nginx: worker process
			//           ├─15321 nginx: worker process
			//           ├─15322 nginx: worker process
			//           ├─15323 nginx: worker process
			//           ├─15324 nginx: worker process
			//           ├─15325 nginx: worker process
			//           ├─15326 nginx: worker process
			//           └─15327 nginx: worker process
			printProject := func(e store.Entity) {
				fmt.Printf("● %-20s\n", e.Project.PrettyName())
				fmt.Printf(" %8s: **%s**\n", "Status", e.Meta.Status)
				fmt.Printf(" %8s: %s\n", "Path", e.Project.Path)

				if len(e.Project.Domain) > 0 {
					fmt.Printf(" %8s: %d\n", "Domain", len(e.Project.Domain))
					for _, d := range e.Project.Domain {
						fmt.Printf("          - %s\n", d.FQDN)
						if d.SSL.IsEnabled() {
							fmt.Printf("            + SSL\n")
						}
						if d.Auth.IsEnabled() {
							fmt.Printf("            + Authentication\n")
							fmt.Printf("              - %8s: %s\n", "User", d.Auth.User)
							if d.Auth.Password == "" {
								fmt.Printf("              - %8s: <empty>\n", "Password")
							} else {
								fmt.Printf("              - %8s: %s\n", "Password", d.Auth.Password)
							}
						}
						for _, r := range d.Redirects {
							fmt.Printf("            R %s\n", r)
						}
						for _, a := range d.Aliases {
							fmt.Printf("            R %s\n", a)
						}
					}
				}

				if len(e.Project.Cron) > 0 {
					fmt.Printf(" %8s: %d\n", "Cron", len(e.Project.Cron))
					for _, c := range e.Project.Cron {
						fmt.Printf("          - %s\n", c.Name)
					}
				}

				if len(e.Project.Worker) > 0 {
					fmt.Printf(" %8s: %d\n", "Worker", len(e.Project.Worker))
					for _, w := range e.Project.Worker {
						fmt.Printf("          - %s\n", w.Name)
					}
				}

				if len(e.Project.Database) > 0 {
					fmt.Printf(" %8s: %d\n", "Database", len(e.Project.Database))
					for _, db := range e.Project.Database {
						fmt.Printf("          - %s\n", db.Name)
						fmt.Printf("            - %8s: %s\n", "User", db.User)
						if db.Password == "" {
							fmt.Printf("            - %8s: <empty>\n", "Password")
						} else {
							fmt.Printf("            - %8s: %s\n", "Password", db.Password)
						}
					}
				}

				if len(e.Project.Volume) > 0 {
					fmt.Printf(" %8s: %d\n", "Volume", len(e.Project.Volume))
					for _, v := range e.Project.Volume {
						if v.IsTemporary {
							fmt.Printf("          T %s\n", v.Path)
						} else {
							fmt.Printf("          P %s\n", v.Path)
						}
					}
				}
			}

			if *all {
				var reply []store.Entity
				args := &sRPC.ProjectAPIArgs{}
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
			Desc: "alias for the domain (repeat for multiple)",
		})

		cmd.Action = func() {
			args := &sRPC.ProjectAPIArgs{
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

	App.Run(os.Args)
}
