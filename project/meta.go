// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

const (
	StatusLoading MetaStatus = iota
	StatusUnloading
	StatusUpdating
	StatusActive
	StatusFailed
	StatusUnknown
)

//go:generate stringer -type=MetaStatus
type MetaStatus int

type Meta struct {
	Status MetaStatus
}
