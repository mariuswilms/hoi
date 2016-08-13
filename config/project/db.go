// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

type DatabaseDirective struct {
	// optional database name; defaults to project name
	Name string
	// optional user; defaults to project name
	User string
	// required; must be non-empty
	Password string
}
