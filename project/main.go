// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Defines the configuration structure of a project, usually
// populated from contents of the Hoifile.
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

func New(id string) (*Config, error) {
	cfg := &Config{
		ID: id,
	}
	return cfg, nil
}

// Assumes f is in the root of the project.
func NewFromFile(f string) (*Config, error) {
	cfg := &Config{
		ID:   PathToID(filepath.Dir(f)),
		Path: filepath.Dir(f),
	}

	b, err := ioutil.ReadFile(f)
	if err != nil {
		return cfg, err
	}

	cfg, err = decodeInto(cfg, string(b))
	if err != nil {
		return cfg, fmt.Errorf("failed to parse config file %s: %s", f, err)

	}
	return cfg, nil
}
func NewFromString(s string) (*Config, error) {
	cfg := &Config{
		ID: fmt.Sprintf("memory:%x", adler32.Checksum([]byte(s))),
	}
	return decodeInto(cfg, s)
}

// The main project configuration is provided by the Hoifile: a per
// project configuration file which defines the needs of a project hoi
// will try to fullfill.
//
// A project provides as much configuration as needed, as the remaining
// configuration is filled in by discovering the projects needs (through
// Augment()).
type Config struct {
	// The ID of the project, will be computed for you.
	ID string
	// The absolute path to the project root; required but will
	// be provided by hoictl mostly automatically.
	Path string
	// The name of the project; optional; if not provided the
	// basename of the project's path will be used, stripped off
	// any context information.
	//   acme       -> acme
	//   acme_stage -> acme
	Name string
	// The name of the context the project is running in. Usually
	// one of "dev", "stage" or "prod"; required.
	Context string
	// A path relative to the project path. If the special value "." is given
	// webroot is equal to the project path. A webroot is the directory exposed
	// under the root of the domains any may contain a front controller; optional,
	// will be autodetected.
	Webroot string
	// Whether PHP is used at all; optional, will be autodetected.
	UsePHP bool
	// Whether we can use try_files in NGINX for rewrites into the
	// front controller or not; optional will be autodetected. Older
	// PHP frameworks will need this.
	UsePHPLegacyRewrites bool
	// The PHP Version in short simple form (5.6.3 -> 56); optional,
	// defaults to "56". Will be used to run projects without PHP 7.0
	// compatibility side by side with those that are compatible.
	PHPVersion string
	// Whether the app can receive large uploads. Normally upload size
	// is limited to 20MB. With large uploads enabled the new limit is 550MB.
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
	// Whether media and assets and all other sub-resurce should be
	// served with a prefixed undersore i.e. /media under /_media, so
	// that they don't conflict with paths routed through the app.
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

func (cfg Config) PrettyName() string {
	if cfg.Name != "" {
		if cfg.Context != "" {
			return fmt.Sprintf("%s@%s", cfg.Name, cfg.Context)
		}
		return fmt.Sprintf("%s@?", cfg.Name)
	}
	return fmt.Sprintf("? in %s", filepath.Base(cfg.Path))
}

func PathToID(path string) string {
	return fmt.Sprintf("%x", adler32.Checksum([]byte(path)))
}

// Extracts username/password pairs from domain configuration.
func (cfg Config) GetCreds() map[string]string {
	creds := make(map[string]string)

	for _, v := range cfg.Domain {
		if !v.Auth.IsEnabled() {
			continue
		}
		creds[v.Auth.User] = v.Auth.Password
	}
	return creds
}

// Extracts cert/cert key pairs mapped to domain FQDN from domain configuration.
func (cfg Config) GetCerts() map[string]SSLDirective {
	certs := make(map[string]SSLDirective)

	for _, v := range cfg.Domain {
		if !v.SSL.IsEnabled() {
			continue
		}
		certs[v.FQDN] = v.SSL
	}
	return certs
}

func (cfg Config) GetAbsoluteWebroot() string {
	return filepath.Join(cfg.Path, cfg.Webroot)
}

// Validates several aspects and looks for typical human errors.
func (cfg Config) Validate() error {
	stringInSlice := func(a string, list []string) bool {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	// Must have context, we can't autodetect this.
	if cfg.Context == "" {
		return fmt.Errorf("project has no context: %s", cfg.Path)
	}

	if cfg.Webroot == "" {
		return fmt.Errorf("project has no webroot: %s", cfg.Path)
	} else if filepath.IsAbs(cfg.Webroot) {
		return fmt.Errorf("webroot must not be absolute: %s", cfg.Webroot)
	}

	creds := make(map[string]string)
	for k, v := range cfg.Domain {
		// Auth credentials should be complete and not vary passwords between
		// same users. The credentials are stored in one single file per project.
		if !v.Auth.IsEnabled() {
			continue
		}
		if v.Auth.User == "" {
			return fmt.Errorf("empty user for domain: %s", v.FQDN)
		}
		if cfg.Context != "dev" && v.Auth.Password == "" {
			return fmt.Errorf("user %s has empty password for domain: %s", v.Auth.User, v.FQDN)
		}
		if _, hasKey := creds[k]; hasKey {
			if creds[k] == v.Auth.Password {
				return fmt.Errorf("auth user %s given multiple times but with differing passwords for domain: %s", v.Auth.User, v.FQDN)
			}
		}
		creds[v.Auth.User] = v.Auth.Password

		if v.SSL.IsEnabled() {
			if v.SSL.CertificateKey == "" {
				return fmt.Errorf("SSL enabled but no certificate key for domain: %s", v.FQDN)
			}
			if string(v.SSL.CertificateKey[0]) == "!" {
				if v.SSL.Certificate != v.SSL.CertificateKey {
					return fmt.Errorf("special action requested for key but not for cert: %s != %s", v.SSL.Certificate, v.SSL.CertificateKey)
				}
			} else {
				if filepath.IsAbs(v.SSL.CertificateKey) {
					return fmt.Errorf("certificate key path is not relative: %s", v.SSL.CertificateKey)
				}
			}

			if v.SSL.Certificate == "" {
				return fmt.Errorf("SSL enabled but no certificate for domain: %s", v.FQDN)
			}
			if string(v.SSL.Certificate[0]) == "!" {
				if v.SSL.CertificateKey != v.SSL.Certificate {
					return fmt.Errorf("special action requested for cert but not for key: %s != %s", v.SSL.Certificate, v.SSL.CertificateKey)
				}
				if cfg.Context == "prod" && v.SSL.Certificate == CertSelfSigned {
					return fmt.Errorf("self-signed certs are not allowed in %s contexts", cfg.Context)
				}
			} else {
				if filepath.IsAbs(v.SSL.Certificate) {
					return fmt.Errorf("certificate path is not relative: %s", v.SSL.Certificate)
				}
			}
		}
	}

	// Database names must be unique and users should for security reasons not
	// have an empty password (not even for dev contexts).
	seenDatabases := make([]string, 0)
	for _, db := range cfg.Database {
		if stringInSlice(db.Name, seenDatabases) {
			return fmt.Errorf("found duplicate database name: %s", db.Name)
		}
		if cfg.Context != "dev" && db.Password == "" {
			return fmt.Errorf("user %s has empty password for database: %s", db.User, db.Name)
		}
		seenDatabases = append(seenDatabases, db.Name)
	}

	return nil
}

// Augments a project configuration as read from a Hoifile, so that
// most configuration does not have to be given explictly and project
// configuration can stay lean.
func (cfg *Config) Augment() error {
	log.Printf("discovering project config: %s", cfg.Path)

	if cfg.Name == "" {
		// Strips the directory name from known context suffix, the
		// context may be added as suffixed later (see database name).
		cfg.Name = strings.TrimSuffix(filepath.Base(cfg.Path), fmt.Sprintf("_%s", cfg.Context))
		log.Printf("- guessed project name: %s", cfg.Name)
	}

	// Discover the webroot by looking a common names and files
	// contained within such a directory. We must take care to not
	// mistakenly expose a directory publicly with contains sensitive
	// material.
	//
	// If we find a directory named "webroot" this is a strong
	// indication it is intended as such.
	//
	// When not finding any directory with this name we'll start
	// looking into the root directory for index.php or index.html
	// files in order to confirm root is the webroot.
	//
	// No other directories except they are named "webroot" or the
	// root directory can become webroot.
	var breakWalk = errors.New("stopped walk early")

	// For performance reasons look in common places first, than
	// fallback to walking the entire tree.
	if _, err := os.Stat(cfg.Path + "/app/webroot"); err == nil {
		cfg.Webroot = "app/webroot"
	} else {
		err := filepath.Walk(cfg.Path, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !f.IsDir() {
				return filepath.SkipDir
			}
			if f.Name() != "webroot" {
				return filepath.SkipDir
			}
			cfg.Webroot = path
			return breakWalk
		})
		if err != nil && err != breakWalk {
			return fmt.Errorf("failed to detect webroot: %s", err)
		}

		if cfg.Webroot == "" {
			_, errPHP := os.Stat(cfg.Path + "/index.php")
			_, errHTML := os.Stat(cfg.Path + "/index.html")
			if errPHP == nil || errHTML == nil {
				cfg.Webroot = "."
			}
		}
	}
	if cfg.Webroot == "" {
		return fmt.Errorf("failed to detect webroot in: %s", cfg.Path)
	} else {
		log.Printf("- found webroot in: %s", cfg.Webroot)
	}

	if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/index.php"); err == nil {
		log.Print("- using PHP")
		cfg.UsePHP = true

		// Detect oldish versions of CakePHP by inspecting the front controller
		// file for certain string patterns. CakePHP version >= use uppercased "Cake"
		// string.
		legacy, err := fileContainsString(cfg.GetAbsoluteWebroot()+"/index.php", "cake")
		if err != nil {
			return err
		}
		if legacy {
			log.Print("- using legacy rewrites")
			cfg.UsePHPLegacyRewrites = true
		}
	}

	// FIXME Check if these are in project root or webroot.
	if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/css"); err == nil {
		log.Print("- using classic assets")
		cfg.UseAssets = true
		cfg.UseClassicAssets = true
	} else if _, err := os.Stat(cfg.Path + "/assets"); err == nil {
		log.Print("- will serve unified assets directory from: /assets")
		cfg.UseAssets = true
	}
	if _, err := os.Stat(cfg.Path + "/media_versions"); err == nil {
		log.Print("- will serve media versions from: /media_versions")
		cfg.UseMediaVersions = true
	}
	if _, err := os.Stat(cfg.Path + "/media"); err == nil {
		log.Print("- will serve media transfers from: /media")
		cfg.UseMediaTransfers = true
	}
	if _, err := os.Stat(cfg.Path + "/files"); err == nil {
		log.Print("- will serve files from: /files")
		cfg.UseFiles = true
	}

	// Guesses auth user names. An empty user name usually indicates
	// that auth is disabled. However, here we interpret non empty
	// passwords as an indicator for enabled auth. This will than
	// trigger the correct behavior in GetCreds().
	for k, _ := range cfg.Domain {
		e := cfg.Domain[k]

		if e.Auth.Password != "" {
			e.Auth.User = cfg.Name
			log.Printf("- guessed auth user: %s", e.Auth.User)
		}
		cfg.Domain[k] = e
	}

	// Guessing will always give the same result, we can therefore
	// only guess once.
	guessedDBName := false
	for k, _ := range cfg.Database {
		e := cfg.Database[k]
		if e.Name == "" {
			if guessedDBName {
				return fmt.Errorf("more than one database name to guess; giving up on augmenting: %s", cfg.Path)
			}
			// Production and local development databases are not
			// suffixed with context name. For other contexts the
			// database name will look like "example_stage".
			if cfg.Context == "prod" || cfg.Context == "dev" {
				e.Name = cfg.Name
			} else {
				e.Name = fmt.Sprintf("%s_%s", cfg.Name, cfg.Context)
			}
			log.Printf("- guessed database name: %s", e.Name)
			guessedDBName = true
		}
		if e.User == "" {
			// User name corresponds to database name and follows the
			// same suffixing rules as the database name.
			e.User = e.Name
			log.Printf("- guessed database user: %s", e.User)
		}
		cfg.Database[k] = e
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
