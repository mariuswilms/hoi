// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"path/filepath"
	"strings"
)

// http://kubernetes.io/docs/user-guide/volumes/#types-of-volumes
const (
	// mount -t tmpfs -o size=100M tmpfs /home/me/tmp
	VolumeKindTemporary  = "temporary"
	VolumeKindPersistent = "persistent"
)

type VolumeDirective struct {
	// Path relative to project root.
	Path string
	Kind string
}

func (drv VolumeDirective) GetSafeName() string {
	// Replace unsafe chars.
	return strings.Replace(drv.Path, "/", "-", -1)
}

func (drv VolumeDirective) GetAbsolutePath(p Config) string {
	return filepath.Join(p.Path, drv.Path)
}
