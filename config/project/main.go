// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config/project defines the configuration structure of a project as
// defined by the Hoifile.
package project

import (
	"errors"
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"
)

func New() (*Config, error) {
	cfg := &Config{}
	return cfg, nil
}
func NewFromFile(f string) (*Config, error) {
	cfg := &Config{}

	b, err := ioutil.ReadFile(f)
	if err != nil {
		return cfg, err
	}

	cfg, err = decodeInto(cfg, string(b))
	if err != nil {
		return cfg, fmt.Errorf("Failed to parse config file %s: %s", f, err)

	}
	cfg.Path = filepath.Dir(f)
	return cfg, nil
}
func NewFromString(s string) (*Config, error) {
	cfg := &Config{}
	return decodeInto(cfg, s)
}

type Config struct {
	// The absolute path to the project root; required but will
	// be provided by hoictl mostly automatically.
	Path string
	// The name of the project; optional; if not provided the
	// basename of the project's path will be used.
	Name string
	// The name of the context the project is running in. Usually
	// one of "dev", "stage" or "prod"; required.
	Context string
	// Whether PHP is used at all; optional, will be autodetected.
	UsePHP bool
	// Whether we can use try_files in NGINX for rewrites into the front
	// controller or not; optional will be autodetected. Older PHP frameworks
	// will need this.
	UsePHPLegacyRewrites bool
	// The PHP Version in short simple form (5.6.3 -> 56); optional, defaults to "56".
	// Will be used to run projects without PHP 7.0 compatibility side by side with
	// those that are compatible.
	PHPVersion string
	// Whether we should enable large uploads inside NGINX (>100MB an <550MB); will be
	// autodetected.
	UseLargeUploads bool
	// Whether media versions should be served.
	UseMediaVersions bool
	// Whether media transfers should be served.
	UseMediaTransfers bool
	// Whether generic files should be served.
	UseFiles bool
	// Whether assets should be served.
	UseAssets bool
	// Whether to use classic img/js/css dirs instead of a single assets dir.
	UseClassicAssets bool
	// Whether media and assets and all other sub-resurce should be served
	// with a prefixed undersore i.e. /media under /_media, so that they don't
	// conflict with paths routed through the app.
	UseNoConflict bool
	// Domains for the project.
	Domain map[string]DomainDirective
	// Crons for the project.
	Cron map[string]CronDirective
	// Workers for the project.
	Worker map[string]WorkerDirective
	// Databases for the project.
	Database map[string]DatabaseDirective
}

func (c Config) ID() string {
	if c.Path == "" {
		log.Fatal(errors.New("no path to generate ID"))
	}
	return fmt.Sprintf("%x", adler32.Checksum([]byte(c.Path)))
}

func (c Config) PrettyName() string {
	if c.Name != "" {
		if c.Context != "" {
			return fmt.Sprintf("%s@%s", c.Name, c.Context)
		}
		return fmt.Sprintf("%s@?", c.Name)
	}
	return fmt.Sprintf("? in %s", filepath.Base(c.Path))
}

func ProjectPathToID(path string) string {
	return fmt.Sprintf("%x", adler32.Checksum([]byte(path)))
}

// Extracts username/password pairs from domain configuration.
func (c Config) GetCreds() (map[string]string, error) {
	creds := make(map[string]string)

	for _, v := range c.Domain {
		if !v.Auth.IsEnabled() {
			continue
		}
		creds[v.Auth.User] = v.Auth.Password
	}
	return creds, nil
}

// Validates several aspects and looks for typical human errors.
func (c Config) Validate() error {
	stringInSlice := func(a string, list []string) bool {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	if c.Context == "" {
		return fmt.Errorf("project has no context: %s", c.Path)
	}

	creds := make(map[string]string)
	for k, v := range c.Domain {
		if !v.Auth.IsEnabled() {
			continue
		}
		if v.Auth.Password == "" {
			return fmt.Errorf("user %s has empty password for domain: %s", v.Auth.User, v.FQDN)
		}
		if _, hasKey := creds[k]; hasKey {
			if creds[k] == v.Auth.Password {
				return fmt.Errorf("auth user %s given multiple times but with differing passwords for domain: %s", v.Auth.User, v.FQDN)
			}
		}
		creds[v.Auth.User] = v.Auth.Password
	}

	seenDatabases := make([]string, 0)
	for _, db := range c.Database {
		if stringInSlice(db.Name, seenDatabases) {
			return fmt.Errorf("found duplicate database name: %s", db.Name)
		}
		if db.Password == "" {
			return fmt.Errorf("user %s has empty password for database: %s", db.User, db.Name)
		}
		seenDatabases = append(seenDatabases, db.Name)
	}

	return nil
}

// Augments a project configuration as read from a Hoifile, so that
// most configuration does not have to be given explictly and project
// configuration can stay lean.
func (c *Config) Augment() error {
	log.Printf("discovering project config: %s", c.Path)

	if c.Name == "" {
		// Strips the directory name from known context suffix, the context
		// may be added as suffixed later (see database name).
		c.Name = strings.TrimSuffix(filepath.Base(c.Path), fmt.Sprintf("_%s", c.Context))
		log.Printf("- guessed project name: %s", c.Name)
	}

	if _, err := os.Stat(c.Path + "/app/webroot/index.php"); err == nil {
		log.Print("- using PHP")
		c.UsePHP = true

		legacy, err := fileContainsString(c.Path+"/app/webroot/index.php", "cake")
		if err != nil {
			return err
		}
		if legacy {
			log.Print("- using legacy rewrites")
			c.UsePHPLegacyRewrites = true
		}
		log.Print("- using large uploads")
		c.UseLargeUploads = true
	}

	if _, err := os.Stat(c.Path + "/assets"); err == nil {
		log.Print("- will serve unified assets directory from: /assets")
		c.UseAssets = true
	}
	if _, err := os.Stat(c.Path + "/media_versions"); err == nil {
		log.Print("- will serve media versions from: /media_versions")
		c.UseMediaVersions = true
	}
	if _, err := os.Stat(c.Path + "/media"); err == nil {
		log.Print("- will serve media transfers from: /media")
		c.UseMediaTransfers = true
	}
	if _, err := os.Stat(c.Path + "/files"); err == nil {
		log.Print("- will serve files from: /files")
		c.UseFiles = true
	}
	if _, err := os.Stat(c.Path + "/app/webroot/css"); err == nil {
		log.Print("- using classic assets")
		c.UseAssets = true
		c.UseClassicAssets = true
	}

	// Guessing will always give the same result, we can therefore only guess once.
	guessedDBName := false
	for k, _ := range c.Database {
		e := c.Database[k]
		if e.Name == "" {
			if guessedDBName {
				return fmt.Errorf("more than one database name to guess; giving up on augmenting: %s", c.Path)
			}
			// Production databases are not suffixed with context name. For other
			// contexts the database name will look like "example_stage".
			if c.Context == "prod" {
				e.Name = c.Name
			} else {
				e.Name = fmt.Sprintf("%s_%s", c.Name, c.Context)
			}
			log.Printf("- guessed database name: %s", e.Name)
			guessedDBName = true
		}
		if e.User == "" {
			// It's OK to have the same user being reused for multiple database (not optimal but OK).
			// The limitations as to the database names (which need to be unique) do not apply here.
			e.User = c.Name
			log.Printf("- guessed database user: %s", e.User)
		}
		c.Database[k] = e
	}
	return nil
}

func fileContainsString(file string, search string) (bool, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}
	s := string(b)
	return strings.Contains(s, search), nil
}

func decodeInto(cfg *Config, s string) (*Config, error) {
	if err := hcl.Decode(cfg, s); err != nil {
		return cfg, err
	}
	for k, _ := range cfg.Domain {
		e := cfg.Domain[k]
		e.FQDN = k
		cfg.Domain[k] = e
	}
	for k, _ := range cfg.Cron {
		e := cfg.Cron[k]
		e.Name = k
		cfg.Cron[k] = e
	}
	for k, _ := range cfg.Worker {
		e := cfg.Worker[k]
		e.Name = k
		cfg.Worker[k] = e
	}
	for k, _ := range cfg.Database {
		e := cfg.Database[k]
		e.Name = k
		cfg.Database[k] = e
	}
	return cfg, nil
}
