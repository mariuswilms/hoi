// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"fmt"
	"path/filepath"

	"github.com/atelierdisko/hoi/server"
)

type VolumeDirective struct {
	// Path relative to project root.
	Path string
	// Whether this volume will get its data wiped
	// on each mount.
	IsTemporary bool
}

// The source directory outside the project.
func (drv VolumeDirective) GetSource(p *Config, s *server.Config) string {
	ns := fmt.Sprintf("project_%s", p.ID)

	var src string
	if drv.IsTemporary {
		src = s.Volume.TemporaryRunPath
	} else {
		src = s.Volume.PersistentRunPath
	}
	return filepath.Join(src, ns, drv.Path)
}

// The target directory inside the project.
func (drv VolumeDirective) GetTarget(p *Config) string {
	return filepath.Join(p.Path, drv.Path)
}
