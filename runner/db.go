// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"database/sql"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
)

// Sets of database-level privileges granted to each database user
// on a per project basis.
const (
	// The minimum set of database level privileges for general project
	// usage (non-administrative tasks).
	DBPrivs = "DELETE,INSERT,SELECT,UPDATE"
	// The minimum set of database level privileges for migrating
	// the database in use by the project.
	DBAdminPrivs = "LOCK TABLES,ALTER,DROP,CREATE,INDEX"
)

func NewDBRunner(s *server.Config, p *project.Config, conn *sql.DB) *DBRunner {
	return &DBRunner{
		s:   s,
		p:   p,
		sys: system.NewMySQL(p, s, conn),
	}
}

// Ensures that database and user for the project are available
// and the user has a minimum set of privileges assigned to her.
type DBRunner struct {
	s   *server.Config
	p   *project.Config
	sys *system.MySQL
}

func (r DBRunner) Disable() error {
	privs := strings.Split(DBPrivs+","+DBAdminPrivs, ",")

	for _, db := range r.p.Database {
		if err := r.sys.EnsureNoGrant(db.User, db.Name, privs); err != nil {
			return err
		}
	}
	return nil
}

func (r DBRunner) Enable() error {
	privs := strings.Split(DBPrivs+","+DBAdminPrivs, ",")

	for _, db := range r.p.Database {
		if err := r.sys.EnsureDatabase(db.Name); err != nil {
			return err
		}
		if err := r.sys.EnsureUser(db.User, db.Password); err != nil {
			return err
		}
		if err := r.sys.EnsureGrant(db.User, db.Name, privs); err != nil {
			return err
		}
	}
	return nil
}

func (r DBRunner) Commit() error {
	return r.sys.ReloadIfDirty()
}

func (r DBRunner) Clean() error {
	return nil // nothing to build
}

func (r DBRunner) Build() error {
	return nil // nothing to build
}
