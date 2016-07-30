// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package rpc

import (
	"log"
	"net"
	"net/rpc"

	pConfig "github.com/atelierdisko/hoi/config/project"
)

type Server struct {
	Socket     string
	ProjectAPI *ProjectAPI
	ServerAPI  *ServerAPI
	listener   net.Listener
}

func (s *Server) Run() error {
	rpc.RegisterName("Project", s.ProjectAPI)
	rpc.RegisterName("Server", s.ServerAPI)

	lis, err := net.Listen("unix", s.Socket)
	if err != nil {
		return err
	}
	s.listener = lis
	go rpc.Accept(s.listener)
	return nil
}

func (s *Server) Close() {
	log.Print("closing RPC server socket")
	s.listener.Close()
}

type ServerAPIArgs struct{}

type ServerAPI struct {
	StatusHandler func() (map[string]pConfig.Config, error)
}

func (s *ServerAPI) Status(args *ServerAPIArgs, reply *map[string]pConfig.Config) error {
	log.Print("client request for: status")
	data, err := s.StatusHandler()
	*reply = data
	return err
}

type ProjectAPI struct {
	LoadHandler   func(pDrv *pConfig.ProjectDirective) error
	DomainHandler func(pDrv *pConfig.ProjectDirective, dDrv *pConfig.DomainDirective) error
}

type ProjectAPIArgs struct {
	Project *pConfig.ProjectDirective
	Domain  *pConfig.DomainDirective
}

func (p *ProjectAPI) Load(args *ProjectAPIArgs, reply *bool) error {
	log.Print("client request for: load")
	*reply = true
	return p.LoadHandler(args.Project)
}
func (p *ProjectAPI) Enable(args *ProjectAPIArgs, reply *bool) error {
	log.Print("client request for: enable")
	*reply = true
	return nil
}
func (p *ProjectAPI) Domain(args *ProjectAPIArgs, reply *bool) error {
	log.Print("client request for: domain")
	*reply = true
	return p.DomainHandler(args.Project, args.Domain)
}
