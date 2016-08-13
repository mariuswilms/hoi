// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

type DatabaseDirective struct {
	// Optional database name; defaults to project name.
	Name string
	// Optional user; defaults to project name.
	User string
	// Password to access the database; required; must be non-empty.
	Password string
}
