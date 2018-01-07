// Copyright 2018 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"log"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/store"
)

type ProjectAPI struct {
	StatusHandler    func(path string) (store.Entity, error)
	StatusAllHandler func() ([]store.Entity, error)
	LoadHandler      func(path string) error
	UnloadHandler    func(path string) error
	UnloadAllHandler func() error
	ReloadHandler    func(path string) error
	ReloadAllHandler func() error
	DomainHandler    func(path string, dDrv *project.DomainDirective) error
	DumpHandler      func(path string, target string) error
}

func (p *ProjectAPI) Status(args *ProjectAPIArgs, reply *store.Entity) error {
	data, err := p.StatusHandler(args.Path)
	*reply = data
	return logIfError(err)
}
func (p *ProjectAPI) StatusAll(args *ProjectAPIArgs, reply *[]store.Entity) error {
	data, err := p.StatusAllHandler()
	*reply = data
	return logIfError(err)
}

func (p *ProjectAPI) Load(args *ProjectAPIArgs, reply *bool) error {
	return logIfError(p.LoadHandler(args.Path))
}
func (p *ProjectAPI) Unload(args *ProjectAPIArgs, reply *bool) error {
	return logIfError(p.UnloadHandler(args.Path))
}
func (p *ProjectAPI) UnloadAll(args *ProjectAPIArgs, reply *bool) error {
	return logIfError(p.UnloadAllHandler())
}

func (p *ProjectAPI) Reload(args *ProjectAPIArgs, reply *bool) error {
	return logIfError(p.ReloadHandler(args.Path))
}
func (p *ProjectAPI) ReloadAll(args *ProjectAPIArgs, reply *bool) error {
	return logIfError(p.ReloadAllHandler())
}

func (p *ProjectAPI) Domain(args *DomainAPIArgs, reply *bool) error {
	return logIfError(p.DomainHandler(args.Path, args.Domain))
}

func (p *ProjectAPI) Dump(args *DumpAPIArgs, reply *bool) error {
	return logIfError(p.DumpHandler(args.Path, args.File))
}

func logIfError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}
