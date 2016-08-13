// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"database/sql"
	"strings"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
	"github.com/atelierdisko/hoi/system"
)

// The minimum set of database level privileges that are granted to project
// database users. Includes privileges to migrate databases.
const DB_PRIVS = "DELETE,INSERT,SELECT,UPDATE"
const DB_ADMIN_PRIVS = "LOCK TABLES,ALTER,DROP,CREATE,INDEX"

func NewDBRunner(s sConfig.Config, p pConfig.Config, conn *sql.DB) *DBRunner {
	return &DBRunner{
		s:   s,
		p:   p,
		sys: system.NewMySQL(p, s, conn),
	}
}

// Ensures that database and user for the project are available
// and the user has a minimum set of privileges assigned to her.
type DBRunner struct {
	s   sConfig.Config
	p   pConfig.Config
	sys *system.MySQL
}

func (r DBRunner) Build() error {
	return nil // nothing to build
}

func (r DBRunner) Clean() error {
	return nil // nothing to build
}

func (r DBRunner) Enable() error {
	privs := strings.Split(DB_PRIVS+","+DB_ADMIN_PRIVS, ",")

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

func (r DBRunner) Disable() error {
	for _, db := range r.p.Database {
		if err := r.sys.EnsureNoGrant(db.User, db.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r DBRunner) Commit() error {
	return r.sys.ReloadIfDirty()
}
