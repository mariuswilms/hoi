// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Defines the configuration structure of a project, usually
// populated from contents of the Hoifile.
package project

import (
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"path/filepath"

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

type ContextType string

// List of possible project contexts.
const (
	ContextUnknown     ContextType = ""
	ContextDevelopment             = "dev"
	ContextStaging                 = "stage"
	ContextProduction              = "prod"
)

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
	// The name of the context the project is running in:
	// one of "dev", "stage" or "prod"; required.
	Context ContextType
	// App backend configuration; mostly detected automatically.
	App AppDirective
	// A path relative to the project path. If the special value "."
	// is given webroot is equal to the project path. A webroot is the
	// directory exposed under the root of the domains any may contain
	// a front controller; optional, will be autodetected.
	Webroot string
	// Whether the app can receive uploads at all (limited to 20MB).
	UseUploads bool
	// Whether the app can receive large uploads. Normally upload size
	// is limited to 20MB. With large uploads enabled the new limit is 550MB.
	UseLargeUploads bool
	// Whether media versions, transfers and assets are nested under
	// the webroot instead of the project root.
	UseWebrootNesting bool
	// Whether media versions should be served.
	UseMediaVersions bool
	// Whether internal media transfers should be served.
	UseMediaTransfers bool
	// Whether internal generic files should be served.
	UseFiles bool
	// Whether assets should be served.
	UseAssets bool
	// Whether to use classic img/js/css directories nested under
	// webroot instead of a single assets dir.
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
	// Volumes for the project
	Volume map[string]VolumeDirective
}

func (cfg Config) PrettyName() string {
	if cfg.Name != "" {
		if cfg.Context != ContextUnknown {
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

func decodeInto(cfg *Config, s string) (*Config, error) {
	if err := hcl.Decode(cfg, s); err != nil {
		return cfg, err
	}

	// key is FQDN
	for k, _ := range cfg.Domain {
		e := cfg.Domain[k]
		e.FQDN = k
		cfg.Domain[k] = e
	}

	// key is Name
	for k, _ := range cfg.Cron {
		e := cfg.Cron[k]
		e.Name = k
		cfg.Cron[k] = e
	}
	for k, _ := range cfg.Worker {
		e := cfg.Worker[k]
		e.Name = k

		if e.Instances == 0 {
			e.Instances = 1
		}
		cfg.Worker[k] = e
	}
	for k, _ := range cfg.Database {
		e := cfg.Database[k]
		e.Name = k
		cfg.Database[k] = e
	}
	for k, _ := range cfg.Volume {
		e := cfg.Volume[k]
		e.Path = k
		cfg.Volume[k] = e
	}
	return cfg, nil
}
