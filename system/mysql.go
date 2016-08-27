// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// BUG(nperson) Revoke/Grant does not set dirty flag.
package system

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

var (
	// no need for mutex: all actions are atomic, we
	// do not reload the whole configuration
	MySQLDirty bool
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
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database)

	log.Printf("MySQL is ensuring database '%s' exists", database)
	res, err := sys.conn.Exec(sql)
	if err != nil {
		return err
	}
	if num, _ := res.RowsAffected(); num > 0 {
		MySQLDirty = true
	}
	return nil
}

func (sys MySQL) HasUser(user string) (bool, error) {
	sql := `SELECT COUNT(*) FROM mysql.user WHERE User = ? AND Host = 'localhost'`

	log.Printf("MySQL is checking for user %s", user)
	rows, err := sys.conn.Query(sql, user)
	if err != nil {
		return false, err
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return false, err
		}
	}
	return count > 0, nil
}

func (sys MySQL) EnsureUser(user string, password string) error {
	log.Printf("MySQL is ensuring user '%s' with password '%s' exists", user, password)

	// MySQL < 5.7.6 and MariaDB < 10.1.3 do not support IF NOT EXISTS.
	var sql string
	if sys.s.MySQL.UseLegacy {
		hasUser, err := sys.HasUser(user)
		if err != nil {
			return err
		}
		if hasUser {
			return nil
		}
		sql = fmt.Sprintf("CREATE USER '%s'@'localhost' IDENTIFIED BY '%s'", user, password)
	} else {
		sql = fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'", user, password)
	}

	res, err := sys.conn.Exec(sql)
	if err != nil {
		return err
	}
	if num, _ := res.RowsAffected(); num > 0 {
		MySQLDirty = true
	}
	return nil
}

// Ensures at least the given privileges are granted to the user on database
// level.
func (sys MySQL) EnsureGrant(user string, database string, privs []string) error {
	hasUser, err := sys.HasUser(user)
	if err != nil {
		return err
	}
	if !hasUser {
		return nil // do not even try to grant
	}

	for _, priv := range privs {
		log.Printf("MySQL is granting user %s privilege %s on %s", user, priv, database)
		sql := fmt.Sprintf("GRANT %s ON %s.* TO '%s'@'localhost'", priv, database, user)
		res, err := sys.conn.Exec(sql)
		if err != nil {
			return err
		}
		if num, _ := res.RowsAffected(); num > 0 {
			MySQLDirty = true
		}
	}
	return nil
}

// Ensures at least the given privileges are granted to the user on database
// level. MySQL does not include GRANT OPTION in ALL.
func (sys MySQL) EnsureNoGrant(user string, database string, privs []string) error {
	hasUser, err := sys.HasUser(user)
	if err != nil {
		return err
	}
	if !hasUser {
		return nil // do not even try to revoke grants
	}

	for _, priv := range privs {
		log.Printf("MySQL is revoking user %s privilege %s on %s", user, priv, database)
		sql := fmt.Sprintf("REVOKE %s ON %s.* FROM '%s'@'localhost'", priv, database, user)
		res, err := sys.conn.Exec(sql)
		if err != nil {
			// Ignore "there is no such grant" errors, querying for privs is tedious.
			log.Printf("MySQL skipped revoke for user %s, has no privilege %s on %s; skipped", user, priv, database)
			continue
		}
		if num, _ := res.RowsAffected(); num > 0 {
			MySQLDirty = true
		}
	}
	return nil
}

func (sys *MySQL) ReloadIfDirty() error {
	if !MySQLDirty {
		return nil
	}
	log.Printf("MySQL is reloading")

	sql := "FLUSH PRIVILEGES"
	if _, err := sys.conn.Exec(sql); err != nil {
		return fmt.Errorf("MySQL left in dirty state")
	}
	MySQLDirty = false
	return nil
}
