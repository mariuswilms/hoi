// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Server configuration.
package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

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
		return cfg, fmt.Errorf("failed to parse config file %s: %s", f, err)
	}
	log.Printf("loaded configuration: %s", f)
	return cfg, err
}
func NewFromString(s string) (*Config, error) {
	cfg := &Config{}
	return decodeInto(cfg, s)
}

type Config struct {
	// Use these user/group when possible i.e. in
	// systemd unit definitions.
	User  string
	Group string
	// E-Mail of administrator, who receives
	// passwords and notifications.
	Email        string
	TemplatePath string
	BuildPath    string
	Web          WebDirective
	NGINX        NGINXDirective
	SSL          SSLDirective
	PHP          PHPDirective
	Cron         CronDirective
	Worker       WorkerDirective
	Systemd      SystemdDirective
	Database     DatabaseDirective
	MySQL        MySQLDirective
}

type WebDirective struct {
	Enabled bool
}
type NGINXDirective struct {
	RunPath   string
	UseLegacy bool
}
type SSLDirective struct {
	Enabled bool
	RunPath string
}
type PHPDirective struct {
	Enabled bool
	RunPath string
}
type CronDirective struct {
	Enabled bool
}
type WorkerDirective struct {
	Enabled bool
}
type SystemdDirective struct {
	RunPath   string
	UseLegacy bool
}
type DatabaseDirective struct {
	Enabled bool
}
type MySQLDirective struct {
	Host      string
	User      string
	Password  string
	UseLegacy bool
}

func decodeInto(cfg *Config, s string) (*Config, error) {
	err := hcl.Decode(cfg, s)

	if err != nil {
		return cfg, err
	}
	cfg.TemplatePath, _ = filepath.Abs(cfg.TemplatePath)
	cfg.BuildPath, _ = filepath.Abs(cfg.BuildPath)

	cfg.NGINX.RunPath, _ = filepath.Abs(cfg.NGINX.RunPath)
	cfg.Systemd.RunPath, _ = filepath.Abs(cfg.Systemd.RunPath)
	cfg.PHP.RunPath, _ = filepath.Abs(cfg.PHP.RunPath)

	return cfg, nil
}
