// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Provides in-memory store with a naive persisting option.
package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/atelierdisko/hoi/project"
)

func New(file string) *Store {
	return &Store{
		file: file,
		data: make(map[string]Entity),
	}
}

type Entity struct {
	Project *project.Config
	Meta    *project.Meta
}

// Store can savely be accessed concurrently. It will persist
// data written to it in a file.
type Store struct {
	// Mutex protecting access to data.
	sync.RWMutex
	file string
	data map[string]Entity
}

// Loads database file contents into memory.
func (s *Store) Load() error {
	if _, err := os.Stat(s.file); os.IsNotExist(err) {
		return nil // nothing to do
	}
	log.Printf("loading store file: %s", s.file)

	f, err := os.Open(s.file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "#", 2)

		if len(fields) != 2 {
			return errors.New("store file corrupt or in unrecognized format")
		}
		entity := &Entity{}
		err := json.Unmarshal([]byte(fields[1]), entity)

		if err != nil {
			return err
		}
		s.Lock()
		s.data[fields[0]] = *entity
		s.Unlock()
	}
	return scanner.Err()
}

// Persists in-memory data to store db file. Automatically called when
// in-memory data has been modified.
func (s Store) Persist() error {
	// Swap contents at the very end, when we are sure that
	// everything else worked.
	var buf []byte
	b := bytes.NewBuffer(buf)

	s.RLock()
	for id, entity := range s.data {
		c, err := json.Marshal(entity)
		if err != nil {
			return err
		}
		b.WriteString(fmt.Sprintf("%s#%s\n", id, string(c)))
	}
	s.RUnlock()

	// Atomic, we don't need locks, last writer wins.
	f, err := os.OpenFile(s.file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, b); err != nil {
		return err
	}
	return f.Sync()
}

func (s *Store) Close() error {
	return nil
}

func (s Store) Has(id string) bool {
	s.RLock()
	defer s.RUnlock()

	_, hasKey := s.data[id]
	return hasKey
}

func (s Store) Read(id string) (Entity, error) {
	s.RLock()
	defer s.RUnlock()

	entity, hasKey := s.data[id]
	if !hasKey {
		return entity, fmt.Errorf("failed to read from store: no id %s", id)
	}
	return entity, nil
}

func (s Store) ReadAll() []Entity {
	s.RLock()
	defer s.RUnlock()

	all := make([]Entity, 0, len(s.data))

	for _, entity := range s.data {
		all = append(all, entity)
	}
	return all
}

func (s *Store) Write(id string, pCfg *project.Config) error {
	s.Lock()

	s.data[id] = Entity{
		Project: pCfg,
		Meta:    &project.Meta{Status: project.StatusUnknown},
	}
	s.Unlock()

	return s.Persist()
}

func (s *Store) Delete(id string) error {
	s.Lock()

	if _, hasKey := s.data[id]; !hasKey {
		return fmt.Errorf("failed to delete from store: no id %s", id)
	}
	delete(s.data, id)
	s.Unlock()

	return s.Persist()
}

func (s Store) ReadStatus(id string) (project.MetaStatus, error) {
	s.RLock()
	defer s.RUnlock()

	if _, hasKey := s.data[id]; !hasKey {
		return project.StatusUnknown, fmt.Errorf("failed to read status: no id %s", id)
	}
	return s.data[id].Meta.Status, nil
}

func (s *Store) WriteStatus(id string, status project.MetaStatus) error {
	s.Lock()

	if _, hasKey := s.data[id]; !hasKey {
		return fmt.Errorf("failed to write status %s: no id %s", status, id)
	}
	entity := s.data[id]
	entity.Meta.Status = status
	s.data[id] = entity
	s.Unlock()

	return s.Persist()
}
