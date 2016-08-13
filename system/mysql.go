// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"database/sql"
	"log"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

func NewMySQL(p project.Config, s server.Config, conn *sql.DB) *MySQL {
	return &MySQL{p: p, s: s, conn: conn}
}

type MySQL struct {
	p     project.Config
	s     server.Config
	conn  *sql.DB
	dirty bool
}

func (sys MySQL) EnsureDatabase(database string) error {
	sql := `CREATE DATABASE IF NOT EXISTS ? `

	res, err := sys.conn.Exec(sql, database)
	if err != nil {
		return err
	}
	if num, _ := res.RowsAffected(); num > 0 {
		sys.dirty = true
	}
	return nil
}

func (sys MySQL) EnsureUser(user string, password string) error {
	sql := `CREATE USER IF NOT EXISTS '?'@'localhost' IDENTIFIED BY '?'`

	res, err := sys.conn.Exec(sql, user, password)
	if err != nil {
		return err
	}
	if num, _ := res.RowsAffected(); num > 0 {
		sys.dirty = true
	}
	return nil
}

// Ensures at least the given privileges are granted to the user on database
// level.
func (sys MySQL) EnsureGrant(user string, database string, privs []string) error {
	for _, priv := range privs {
		sql := `GRANT ? ON ?.* TO '?'@'localhost'`
		res, err := sys.conn.Exec(sql, priv, database, user)
		if err != nil {
			return err
		}
		if num, _ := res.RowsAffected(); num > 0 {
			sys.dirty = true
		}
	}
	return nil
}

// Ensures at least the given privileges are granted to the user on database
// level. MySQL does not include GRANT OPTION in ALL.
func (sys MySQL) EnsureNoGrant(user string, database string) error {
	sql := `REVOKE ALL PRIVILEGES, GRANT OPTION ON ?.* TO '?'@'localhost'`
	res, err := sys.conn.Exec(sql, database, user)
	if err != nil {
		return err
	}
	if num, _ := res.RowsAffected(); num > 0 {
		sys.dirty = true
	}
	return nil
}

func (sys *MySQL) ReloadIfDirty() error {
	if !sys.dirty {
		return nil
	}
	sql := `FLUSH PRIVILEGES`
	if _, err := sys.conn.Exec(sql); err != nil {
		log.Printf("MySQL reload: left in dirty state")
		return err
	}
	sys.dirty = false
	return nil
}
