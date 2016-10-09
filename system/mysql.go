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

	res, err := sys.conn.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed creating MySQL database '%s': %s", database, err)
	}
	if num, _ := res.RowsAffected(); num > 0 {
		MySQLDirty = true
	}
	return nil
}

func (sys MySQL) HasUser(user string) (bool, error) {
	sql := `SELECT COUNT(*) FROM mysql.user WHERE User = ? AND Host = 'localhost'`

	rows, err := sys.conn.Query(sql, user)
	if err != nil {
		return false, fmt.Errorf("failed to check for MySQL user '%s': %s", user, err)
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return false, err
		}
	}
	return count > 0, nil
}

func (sys MySQL) HasPassword(user string, password string) (bool, error) {
	sql := `SELECT COUNT(*) FROM mysql.user WHERE User = ? AND Host = 'localhost' AND Password = PASSWORD(?)`

	rows, err := sys.conn.Query(sql, user, password)
	if err != nil {
		return false, fmt.Errorf("failed to to verify MySQL user '%s' has password '%s': %s", user, password, err)
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
	var sql string

	hasUser, err := sys.HasUser(user)
	if err != nil {
		return err
	}
	if hasUser {
		hasPassword, err := sys.HasPassword(user, password)
		if err != nil {
			return err
		}
		if hasPassword {
			return nil
		}

		if sys.s.MySQL.UseLegacy {
			// PASSWORD() is deprecated and should be used in legacy systems only.
			sql = fmt.Sprintf("SET PASSWORD FOR '%s'@'localhost' = PASSWORD('%s')", user, password)
		} else {
			// ALTER USER to change password is supported since MySQL 5.7.6
			sql = fmt.Sprintf("ALTER USER '%s'@'localhost' IDENTIFIED BY '%s'", user, password)
		}
		res, err := sys.conn.Exec(sql)
		if err != nil {
			return fmt.Errorf("failed setting new password '%s' for MySQL user '%s': %s", password, user, err)
		}
		if num, _ := res.RowsAffected(); num > 0 {
			MySQLDirty = true
		}
		return nil
	}

	sql = fmt.Sprintf("CREATE USER '%s'@'localhost' IDENTIFIED BY '%s'", user, password)
	res, err := sys.conn.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed creating MySQL user '%s' with password '%s': %s", user, password, err)
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
		sql := fmt.Sprintf("GRANT %s ON %s.* TO '%s'@'localhost'", priv, database, user)
		res, err := sys.conn.Exec(sql)
		if err != nil {
			return fmt.Errorf("failed granting MySQL user '%s' privilege '%s' on '%s': %s", user, priv, database, err)
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
		sql := fmt.Sprintf("REVOKE %s ON %s.* FROM '%s'@'localhost'", priv, database, user)
		res, err := sys.conn.Exec(sql)
		if err != nil {
			// Ignore "there is no such grant" errors, querying for privs is tedious.
			log.Printf("skipped revoke for MySQL user %s, has no privilege %s on %s; skipped", user, priv, database)
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
	sql := "FLUSH PRIVILEGES"
	if _, err := sys.conn.Exec(sql); err != nil {
		return fmt.Errorf("failed to reload MySQL, has been left in dirty state: %s", err)
	}
	MySQLDirty = false
	return nil
}
