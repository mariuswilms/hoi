// Copyright 2018 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import "github.com/atelierdisko/hoi/project"

type ProjectAPIArgs struct {
	// Path is an absolute path to project root; required field.
	Path string
}

type DomainAPIArgs struct {
	Path   string
	Domain *project.DomainDirective
}

type DumpAPIArgs struct {
	Path string
	// Absolute path to target or source file. May be outside project root.
	File string
}
