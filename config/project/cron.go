// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package project

import (
	"fmt"
	"hash/adler32"
)

type CronDirective struct {
	// Descriptive name, to identify the cron.
	Name string
	// See systemd.time for valid syntaxes:
	// https://www.freedesktop.org/software/systemd/man/systemd.time.html
	Schedule string
	Commando `hcl:",squash"`
}

func (drv CronDirective) ID() string {
	if drv.Name == "" {
		// Fallback to hashing command. Project ID is already prefixed.
		return fmt.Sprintf("%x", adler32.Checksum([]byte(drv.Command)))
	}
	return drv.Name
}
