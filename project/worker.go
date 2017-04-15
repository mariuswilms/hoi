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

type WorkerDirective struct {
	// An optional descriptive name which allows to identify the
	// worker later easily. If no name is given a hash of Command
	// is used to identify the cron uniquely.
	Name string
	// How many instances of the worker should be spawned; optional;
	// defaults to 1.
	Instances int
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
func (drv WorkerDirective) GetID() string {
	if drv.Name == "" {
		return fmt.Sprintf("%x", adler32.Checksum([]byte(drv.Command)))
	}
	return drv.Name
}

// Returns number of instances converting to correct unsigned integer type.
func (drv WorkerDirective) GetInstances() uint {
	return uint(drv.Instances)
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
func (drv WorkerDirective) GetCommand(p *Config) (string, error) {
	var cmd string

	if !strings.Contains(drv.Command, "{{") {
		cmd = drv.Command
	} else {
		log.Printf("parsing command template: %s", drv.Command)

		cmdTmplData := struct {
			P *Config
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
