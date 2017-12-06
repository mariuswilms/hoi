// Copyright 2017 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"fmt"
	"net"

	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/util"
	"github.com/coreos/go-semver/semver"
)

type AppKind string

const (
	AppKindUnknown AppKind = ""
	// Project consisting of static contents only which can directly be
	// served by a HTTP server.
	AppKindStatic = "static"
	// A generic service that starts a HTTP server and we proxy to.
	AppKindService = "service"
	// A project that uses .php files and optionally routes all requests
	// through a front controller.
	AppKindPHP = "php"
)

type AppDirective struct {
	// The kind of app we are using.
	Kind AppKind
	// The semantic version of the app language to use. For an PHP app
	// can switch the FPM socket by looking at the major part of the
	// version, to run projects side by side.
	Version string
	// Used only for service backends. Defaults to localhost.
	Host string
	// Used only for service backends. By default picks the next free
	// non-privileged port from range.
	Port uint16
	// Holds a command string, that starts a HTTP server. The command
	// can either be a path (relative to project root or absolute)
	// or a template which evaluates to one of both. Templates may
	// reference P (the project configuration).
	//
	//   bin/server -l {.P.App.Host}:{.P.App.Port}
	//
	// Used only for service apps.
	Command `hcl:",squash"`
	// Whether we want to use "pretty URLs" by rewriting the incoming
	// URLs as a GET parameter of the front controller file.
	//
	//   /foo/bar -> /index.html?/foo/bar
	//
	// Used only for static and PHP apps.
	UseFrontController bool
	// Whether we can use try_files in NGINX for rewrites into the
	// front controller or not; optional and will be autodetected.
	// Older PHP frameworks will need this.
	//
	// Used only for static and PHP apps.
	UseLegacyFrontController bool
}

// Certain apps (i.e. PHP) have a corresponding service unit that we need to reload
// on configuration changes. Returns a systemd service unit name including suffix.
func (drv AppDirective) GetService(p *Config, s *server.Config) (string, error) {
	if drv.Kind != AppKindPHP {
		return "", fmt.Errorf("app kind %s has no service unit", drv.Kind)
	}
	service, err := util.ParseAndExecuteTemplate("service", s.PHP.Service, struct {
		P *Config
		S *server.Config
	}{
		P: p,
		S: s,
	})
	return fmt.Sprintf("%s.service", service), err
}

// Certain apps (i.e. PHP) need further outside configuration.
func (drv AppDirective) GetRunPath(p *Config, s *server.Config) (string, error) {
	if drv.Kind != AppKindPHP {
		return "", fmt.Errorf("app kind %s has no run path", drv.Kind)
	}
	return util.ParseAndExecuteTemplate("runPath", s.PHP.RunPath, struct {
		P *Config
		S *server.Config
	}{
		P: p,
		S: s,
	})
}

// Returns next available port number we want to assign to the app.
func (drv AppDirective) GetFreePort(p *Config) (uint16, error) {
	port := uint16(0)
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", drv.Host, port))

	if err != nil {
		return port, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return port, err
	}
	port = uint16(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port, nil
}

// Returns major part of version string. Also takes default versions into account.
func (drv AppDirective) GetMajorVersion(s *server.Config) (int64, error) {
	v, err := drv.getVersion(s)
	if err != nil {
		return 0, err
	}
	return v.Major, nil
}

// Returns minor part of version string. Also takes default versions into account.
func (drv AppDirective) GetMinorVersion(s *server.Config) (int64, error) {
	v, err := drv.getVersion(s)
	if err != nil {
		return 0, err
	}
	return v.Minor, nil
}

// Returns a semantic version type, which allows accessing the major,
// minor parts of the version string. When a version was not given
// in project configuration will try to find a default version from
// server configuration.
func (drv AppDirective) getVersion(s *server.Config) (*semver.Version, error) {
	if drv.Version != "" {
		return semver.NewVersion(drv.Version)
	}
	if drv.Kind == AppKindPHP {
		return semver.NewVersion(s.PHP.Version)
	}
	return nil, fmt.Errorf("failed to get app version: no default version for kind %s found", drv.Kind)
}
