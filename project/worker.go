// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"fmt"
	"hash/adler32"
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
	// reference P (the project configuration):
	//
	//   bin/cute-worker --scope={{.P.Name}}_{{.P.Context}}
	//
	// Commands will be executed with the project root path as the current working
	// directory.
	Command `hcl:",squash"`
}

// Generates the ID for the directive, prefers the plain Name, if that
// is not present falls back to hasing the contents of Command, as
// these (together with the project ID) are assumed to be unique enough.
func (drv WorkerDirective) GetID() string {
	if drv.Name == "" {
		return fmt.Sprintf("%x", adler32.Checksum([]byte(drv.Command.Command)))
	}
	return drv.Name
}

// Returns number of instances converting to correct unsigned integer type.
func (drv WorkerDirective) GetInstances() uint {
	return uint(drv.Instances)
}
