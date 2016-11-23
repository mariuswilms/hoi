// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Provides a communication channel between client
// and server.
package rpc

import (
	"log"
	"net"
	"net/rpc"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/store"
)

type Server struct {
	Socket     string
	ProjectAPI *ProjectAPI
	listener   net.Listener
}

func (s *Server) Run() error {
	rpc.RegisterName("Project", s.ProjectAPI)

	lis, err := net.Listen("unix", s.Socket)
	if err != nil {
		return err
	}
	s.listener = lis
	go rpc.Accept(s.listener)
	log.Printf("listening for RPC calls on: %s", s.Socket)
	return nil
}

func (s *Server) Close() {
	log.Print("closing RPC server socket")
	s.listener.Close()
}

type ProjectAPI struct {
	StatusHandler    func(path string) (store.Entity, error)
	StatusAllHandler func() ([]store.Entity, error)
	LoadHandler      func(path string) error
	UnloadHandler    func(path string) error
	UnloadAllHandler func() error
	ReloadHandler    func(path string) error
	ReloadAllHandler func() error
	DomainHandler    func(path string, dDrv *project.DomainDirective) error
}

type ProjectAPIArgs struct {
	Path   string
	Domain *project.DomainDirective
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
	*reply = true
	return logIfError(p.LoadHandler(args.Path))
}
func (p *ProjectAPI) Unload(args *ProjectAPIArgs, reply *bool) error {
	*reply = true
	return logIfError(p.UnloadHandler(args.Path))
}
func (p *ProjectAPI) UnloadAll(args *ProjectAPIArgs, reply *bool) error {
	*reply = true
	return logIfError(p.UnloadAllHandler())
}

func (p *ProjectAPI) Reload(args *ProjectAPIArgs, reply *bool) error {
	*reply = true
	return logIfError(p.ReloadHandler(args.Path))
}
func (p *ProjectAPI) ReloadAll(args *ProjectAPIArgs, reply *bool) error {
	*reply = true
	return logIfError(p.ReloadAllHandler())
}

func (p *ProjectAPI) Domain(args *ProjectAPIArgs, reply *bool) error {
	*reply = true
	return logIfError(p.DomainHandler(args.Path, args.Domain))
}

func logIfError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}
