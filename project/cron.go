// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"bytes"
	"fmt"
	"hash/adler32"
	"log"
	"path/filepath"
	"strings"
	"text/template"
)

// Jobs that are run on a regular basis are configured via the cron
// directive. The schedule option supports expressions from
// systemd.time.
//
// https://www.freedesktop.org/software/systemd/man/systemd.time.html
type CronDirective struct {
	// An optional descriptive name which allows to identify the
	// cron later easily. If no name is given a hash of Command
	// is used to identify the cron uniquely.
	Name string
	// A valid systemd.time time and date specification expresssion that
	// allows us to determine in which interval the cron should be run, i.e.:
	// "hourly", "daily", "weekly", "monthly", "yearly"
	Schedule string
	// Holds a command string which can be either a path (relative to project root
	// or absolute) or a template which evaluates to one of both. Templates may
	// reference P (the project configuration).
	//
	// Commands will be executed with the project root path as the current working
	// directory.
	Command string
}

// Generates the ID for the directive, prefers the plain Name, if that
// is not present falls back to hasing the contents of Command, as
// these (together with the project ID) are assumed to be unique enough.
func (drv CronDirective) ID() string {
	if drv.Name == "" {
		return fmt.Sprintf("%x", adler32.Checksum([]byte(drv.Command)))
	}
	return drv.Name
}

// Returns (parsed and) absolute command string.
//
// Command strings may use template syntax (project configuration
// is made available as P). Will parse only when necessarry, most
// commands will not use templating.
//
// When used inside systemd service unit files paths need to be
// absolute. When a command string is non absolute it will be treated
// as being relative to the project root directory and made absolute.
func (drv CronDirective) GetCommand(p Config) (string, error) {
	var cmd string

	if !strings.Contains(drv.Command, "{{") {
		cmd = drv.Command
	} else {
		log.Printf("parsing command template: %s", drv.Command)

		cmdTmplData := struct {
			P Config
		}{
			P: p,
		}
		buf := new(bytes.Buffer)
		cmdT := template.New("cmd")
		cmdT.Parse(drv.Command)

		if err := cmdT.Execute(buf, cmdTmplData); err != nil {
			return "", err
		}
		cmd = buf.String()
	}
	if !filepath.IsAbs(cmd) {
		cmd = filepath.Join(p.Path, cmd)
	}
	return cmd, nil
}
