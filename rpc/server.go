// Copyright 2018 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"log"
	"net"
	"net/rpc"
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
