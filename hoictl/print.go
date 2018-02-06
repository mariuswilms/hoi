// Copyright 2018 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/atelierdisko/hoi/store"
)

// Outputs information about a project entity.
//
// Roughly modelled aftret the systemctl status output:
//   ● nginx.service - A high performance web server and a reverse proxy server
//      Loaded: loaded (/lib/systemd/system/nginx.service; enabled)
//      Active: active (running) since Thu 2016-10-13 14:07:20 CEST; 8h ago
//     Process: 15315 ExecReload=/usr/sbin/nginx -g daemon on; master_process on; -s reload (code=exited, status=0/SUCCESS)
//    Main PID: 1693 (nginx)
//      CGroup: /system.slice/nginx.service
//              ├─ 1693 nginx: master process /usr/sbin/nginx -g daemon on; master_process on;
//              ├─15320 nginx: worker process
//              ├─15321 nginx: worker process
//              ├─15322 nginx: worker process
//              ├─15323 nginx: worker process
//              ├─15324 nginx: worker process
//              ├─15325 nginx: worker process
//              ├─15326 nginx: worker process
//              └─15327 nginx: worker process
//
// FIXME: Use go text template for generating output.
func printProject(e store.Entity) {
	fmt.Printf("● %-20s\n", e.Project.PrettyName())
	fmt.Printf(" %14s: %s\n", "ID", e.Project.ID)
	fmt.Printf(" %14s: **%s**\n", "Status", e.Meta.Status)
	fmt.Printf(" %14s: %s\n", "Path", e.Project.Path)
	fmt.Printf(" %14s: %d\n", "Format Version", e.Project.FormatVersion)

	fmt.Printf(" %8s: %s\n", "App", e.Project.App.Kind)
	if e.Project.App.Version != "" {
		fmt.Printf("          - %s: %s\n", "Version", e.Project.App.Version)
	}
	if e.Project.App.HasCommand() {
		fmt.Printf("          - %s: %s\n", "Command", e.Project.App.Command)
	}
	if e.Project.App.Host != "" || e.Project.App.Port != 0 {
		fmt.Printf("          - %s: %s:%d\n", "Address", e.Project.App.Host, e.Project.App.Port)
	}

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
				fmt.Printf("            A %s\n", a)
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
			fmt.Printf("          - %s (x%d)\n", w.Name, w.Instances)
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
