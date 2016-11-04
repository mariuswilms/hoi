// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import "path/filepath"

type VolumeDirective struct {
	// Path relative to project root.
	Path string
	// Whether this volume will get its data wiped
	// on each mount.
	IsTemporary bool
}

func (drv VolumeDirective) GetAbsolutePath(p Config) string {
	return filepath.Join(p.Path, drv.Path)
}
