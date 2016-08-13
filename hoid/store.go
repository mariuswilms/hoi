// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"sync"

	pConfig "github.com/atelierdisko/hoi/config/project"
)

type MemoryStore struct {
	sync.RWMutex
	// no pointer as it then would be possible to modify data outside lock
	data map[string]pConfig.Config
}

func (s *MemoryStore) Stats() string {
	Store.RLock()
	out := fmt.Sprintf("STATS STORE count:%d", len(s.data))
	Store.RUnlock()
	return out
}
