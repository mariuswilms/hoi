// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

const (
	StatusUnknown MetaStatus = iota
	StatusLoading
	StatusUnloading
	StatusReloading
	StatusUpdating
	StatusActive
	StatusFailed
)

//go:generate stringer -type=MetaStatus
type MetaStatus int

type Meta struct {
	Status MetaStatus
}
