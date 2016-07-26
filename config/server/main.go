// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	pConfig "github.com/atelierdisko/hoi/config/project"
	"github.com/hashicorp/hcl"
)

type Config struct {
	User     string
	Group    string
	NGINX    NGINXDirective
	Systemd  SystemdDirective
	Backup   BackupDirective
	Project  map[string]ProjectDirective
	Database map[string]DatabaseDirective
	// E-Mail of administrator, who receives
	// passwords and notifications.
	Email string
}

type Paths struct {
	TemplatePath string
	BuildPath    string
	RunPath      string
}

func (ps *Paths) GetTemplatePath() (string, error) {
	if ps.TemplatePath == "" {
		return "", errors.New("refusing to return empty template path")
	}
	return filepath.Abs(ps.TemplatePath)
}
func (ps *Paths) GetBuildPathForProject(pCfg *pConfig.Config) (string, error) {
	if ps.BuildPath == "" {
		return "", fmt.Errorf("refusing to use empty build path as base for project %s", pCfg.PrettyName())
	}
	base, err := filepath.Abs(ps.BuildPath)
	if err != nil {
		return ps.BuildPath, err
	}
	return filepath.Join(base, pCfg.Id()), nil
}
func (ps *Paths) GetRunPath() (string, error) {
	if ps.RunPath == "" {
		return "", errors.New("refusing to return empty run path")
	}
	return filepath.Abs(ps.RunPath)
}

type NGINXDirective struct {
	Paths `hcl:",squash"`
}

type SystemdDirective struct {
	Paths `hcl:",squash"`
}

type DatabaseDirective struct {
	User     string
	Password string
}
type BackupDirective struct {
	Offsite OffsiteBackupDirective
	Onsite  OnsiteBackupDirective
}
type OnsiteBackupDirective struct {
	Path string
}
type OffsiteBackupDirective struct {
	Host   string
	User   string
	Secret string
}
type ProjectDirective struct {
	Path string
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
	return cfg, err
}

func New() (*Config, error) {
	cfg := &Config{}
	return cfg, nil
}
func NewFromString(s string) (*Config, error) {
	cfg := &Config{}
	return decodeInto(cfg, s)
}

func decodeInto(cfg *Config, s string) (*Config, error) {
	err := hcl.Decode(cfg, s)

	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
