// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
	"github.com/atelierdisko/hoi/system"
)

func NewVolumeRunner(s server.Config, p project.Config) *VolumeRunner {
	return &VolumeRunner{
		s:   s,
		p:   p,
		sys: system.NewStorage(p, s),
	}
}

type VolumeRunner struct {
	s server.Config
	p project.Config
	// does not need any configuration, thus no builder
	sys *system.Storage
}

func (r VolumeRunner) Build() error {
	// No configuration building happening.
	return nil
}

func (r VolumeRunner) Clean() error {
	// No configuration building happening.
	return nil
}

func (r VolumeRunner) Enable() error {
	for _, v := range r.p.Volume {
		if err := r.sys.Install(v); err != nil {
			return err
		}
	}
	return nil
}

func (r VolumeRunner) Disable() error {
	for _, v := range r.p.Volume {
		if err := r.sys.Uninstall(v); err != nil {
			return err
		}
	}
	return nil
}

func (r VolumeRunner) Commit() error {
	// No txn possible, changes are committed when enabling.
	return nil
}
