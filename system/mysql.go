// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// BUG(nperson) Revoke/Grant does not set dirty flag.
package system

import (
	"archive/tar"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

var (
	// no need for mutex: all actions are atomic, we
	// do not reload the whole configuration
	MySQLDirty bool
)

func NewMySQL(p *project.Config, s *server.Config, conn *sql.DB) *MySQL {
	return &MySQL{p: p, s: s, conn: conn}
}

type MySQL struct {
	p     *project.Config
	s     *server.Config
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
	sql := `SELECT COUNT(*) FROM mysql.user WHERE User = ? AND Host = 'localhost' AND CAST(Password as Binary) = PASSWORD(?)`

	rows, err := sys.conn.Query(sql, user, password)
	if err != nil {
		return false, fmt.Errorf("failed to verify MySQL user '%s' has password '%s': %s", user, password, err)
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

	if err := sys.CheckRestrictedUser(user); err != nil {
		return err
	}

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
		// Let's give a heads up on this, as changing password
		// especially with shared user accounts can lead to unintended
		// side effects. Current hoi versions will not use shared
		// accounts, but older ones did.
		log.Printf("changing MySQL password for user '%s' to '%s'", user, password)

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
	if err := sys.CheckRestrictedUser(user); err != nil {
		return err
	}

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
	if err := sys.CheckRestrictedUser(user); err != nil {
		return err
	}

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

// Ensures we don't manipulate general protected (admin) users.
func (sys MySQL) CheckRestrictedUser(user string) error {
	if user == "root" {
		return fmt.Errorf("is MySQL restricted user: %s", user)
	}
	return nil
}

// Dumps can grow quite large (several GB large), so we're using
// a disk-backed buffer as to to keep memory usage low.
//
// TODO: Implement gzipping: SQL dumps usually compress very well, but
// media data does not. So we chose to compress the dump inside the
// tar archive and not the archive as a whole. The name of the dump
// inside the archive will be <database>.sql.gz.
func (sys MySQL) DumpDatabase(database string, tw *tar.Writer) error {
	tmp, err := ioutil.TempFile("", "hoi_")
	if err != nil {
		return err
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	cmdArgs := []string{
		"--opt",
		fmt.Sprintf("-u%s", sys.s.MySQL.User),
	}
	if sys.s.MySQL.Password != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-p%s", sys.s.MySQL.Password))
	}
	cmdArgs = append(cmdArgs, database)

	cmd := exec.Command("mysqldump", cmdArgs...)
	cmd.Stdout = tmp

	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	// Calculate final size and reset for reading from.
	_, err = tmp.Seek(0, 0)
	if err != nil {
		return err
	}
	stat, err := tmp.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	log.Printf("database %s dump created, is %d bytes", database, size)

	header := &tar.Header{
		Name:     fmt.Sprintf("%s.sql", database),
		Size:     size,
		ModTime:  time.Now(),
		Typeflag: tar.TypeReg,
		Mode:     0660,
		ModTime:  time.Now(),
		Uid:      0,
		Gid:      0,
		Uname:    "root",
		Gname:    "root",
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, tmp); err != nil {
		return err
	}
	return nil
}
