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
	// Descriptive name, to identify the worker.
	Name string
	// How many instances of the worker should be spawned.
	Instances int
	// Can be relative to project root or absolute.
	Commando `hcl:",squash"`
}

func (drv WorkerDirective) ID() string {
	if drv.Name == "" {
		// Fallback to hashing command. Project ID is already prefixed.
		return fmt.Sprintf("%x", adler32.Checksum([]byte(drv.Command)))
	}
	return drv.Name
}

// Default to at least 1 instance.
func (drv WorkerDirective) GetInstances() uint {
	if drv.Instances == 0 {
		return uint(1)
	}
	return uint(drv.Instances)
}
