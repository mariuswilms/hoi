// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

// Hoi can manage the database creation and users for you. It will
// create a database and users with minimum sets of privileges if they
// do not exist.
type DatabaseDirective struct {
	// Optional database name; defaults to project name with the context
	// suffixed if context is not prod or dev.
	Name string
	// Optional user; defaults to database name.
	User string
	// Password to access the database; required; must be non-empty
	// except in "dev" context where empty passwords are permitted
	// to ease development.
	Password string
}
